package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Executor defines the function signature for task executors.
type Executor func(ctx context.Context, task *Task, onProgress func(int)) error

// Manager manages AI tasks.
type Manager struct {
	mu sync.RWMutex

	repo      Repository
	executors map[Type]Executor

	// Concurrency control
	semaphore chan struct{}
	maxConcurrent int

	// Progress subscriptions
	subscribers map[uuid.UUID][]func(*Task)

	// Lifecycle
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// ManagerConfig contains manager configuration.
type ManagerConfig struct {
	MaxConcurrent int
}

// DefaultManagerConfig returns the default manager configuration.
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		MaxConcurrent: 10,
	}
}

// NewManager creates a new task manager.
func NewManager(repo Repository, config *ManagerConfig) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &Manager{
		repo:          repo,
		executors:     make(map[Type]Executor),
		semaphore:     make(chan struct{}, config.MaxConcurrent),
		maxConcurrent: config.MaxConcurrent,
		subscribers:   make(map[uuid.UUID][]func(*Task)),
		stopCh:        make(chan struct{}),
	}
}

// Start starts the task manager.
func (m *Manager) Start(ctx context.Context) error {
	// Recover pending tasks
	if err := m.RecoverPendingTasks(ctx); err != nil {
		return fmt.Errorf("recover pending tasks: %w", err)
	}

	return nil
}

// Stop stops the task manager.
func (m *Manager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// RegisterExecutor registers a task executor.
func (m *Manager) RegisterExecutor(taskType Type, executor Executor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executors[taskType] = executor
}

// Submit submits a new task.
func (m *Manager) Submit(ctx context.Context, userID uuid.UUID, input *Input) (*Task, error) {
	task := &Task{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      input.Type,
		Status:    StatusPending,
		Progress:  0,
		Input:     input.Payload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Start execution in background
	m.wg.Add(1)
	go m.executeTask(task)

	return task, nil
}

// Get retrieves a task by ID.
func (m *Manager) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	return m.repo.Get(ctx, id)
}

// List lists tasks for a user.
func (m *Manager) List(ctx context.Context, userID uuid.UUID, filter *Filter) ([]*Task, error) {
	if filter == nil {
		filter = &Filter{}
	}
	filter.UserID = &userID
	return m.repo.List(ctx, filter)
}

// Cancel cancels a task.
func (m *Manager) Cancel(ctx context.Context, id uuid.UUID) error {
	task, err := m.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if task.IsTerminal() {
		return fmt.Errorf("task already in terminal state: %s", task.Status)
	}

	task.Status = StatusCancelled
	task.UpdatedAt = time.Now()

	if err := m.repo.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	m.notifySubscribers(task)
	return nil
}

// OnProgress subscribes to task progress updates.
func (m *Manager) OnProgress(id uuid.UUID, callback func(*Task)) func() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscribers[id] = append(m.subscribers[id], callback)

	// Return unsubscribe function
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		subs := m.subscribers[id]
		for i, sub := range subs {
			// Compare function pointers (not reliable in Go, but best effort)
			if &sub == &callback {
				m.subscribers[id] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
}

// RecoverPendingTasks recovers pending tasks after restart.
func (m *Manager) RecoverPendingTasks(ctx context.Context) error {
	tasks, err := m.repo.ListPendingOrRunning(ctx)
	if err != nil {
		return fmt.Errorf("list pending tasks: %w", err)
	}

	for _, task := range tasks {
		// Reset running tasks to pending
		if task.Status == StatusRunning {
			task.Status = StatusPending
			task.UpdatedAt = time.Now()
			if err := m.repo.Update(ctx, task); err != nil {
				continue
			}
		}

		// Re-queue for execution
		m.wg.Add(1)
		go m.executeTask(task)
	}

	return nil
}

// executeTask executes a task.
func (m *Manager) executeTask(task *Task) {
	defer m.wg.Done()

	// Acquire semaphore
	select {
	case <-m.stopCh:
		return
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	}

	ctx := context.Background()

	// Get executor
	m.mu.RLock()
	executor, ok := m.executors[task.Type]
	m.mu.RUnlock()

	if !ok {
		m.failTask(ctx, task, "no executor registered for task type")
		return
	}

	// Update status to running
	task.Status = StatusRunning
	task.UpdatedAt = time.Now()
	if err := m.repo.Update(ctx, task); err != nil {
		return
	}
	m.notifySubscribers(task)

	// Execute
	onProgress := func(progress int) {
		task.Progress = progress
		task.UpdatedAt = time.Now()
		_ = m.repo.UpdateStatus(ctx, task.ID, task.Status, progress)
		m.notifySubscribers(task)
	}

	if err := executor(ctx, task, onProgress); err != nil {
		m.failTask(ctx, task, err.Error())
		return
	}

	// Complete
	task.Status = StatusCompleted
	task.Progress = 100
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := m.repo.Update(ctx, task); err != nil {
		return
	}
	m.notifySubscribers(task)
}

// failTask marks a task as failed.
func (m *Manager) failTask(ctx context.Context, task *Task, message string) {
	task.Status = StatusFailed
	task.Error = &Error{
		Code:    "execution_failed",
		Message: message,
	}
	task.UpdatedAt = time.Now()

	_ = m.repo.Update(ctx, task)
	m.notifySubscribers(task)
}

// notifySubscribers notifies all subscribers of a task update.
func (m *Manager) notifySubscribers(task *Task) {
	m.mu.RLock()
	subs := m.subscribers[task.ID]
	m.mu.RUnlock()

	for _, sub := range subs {
		sub(task)
	}
}
