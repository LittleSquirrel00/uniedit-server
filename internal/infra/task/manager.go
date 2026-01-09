package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Executor defines the function signature for task executors.
// Executors should update task output via the provided callback.
type Executor func(ctx context.Context, task *Task, onProgress func(progress int, output map[string]any)) error

// ExternalTaskPoller defines the interface for polling external task status.
type ExternalTaskPoller interface {
	// PollStatus polls the status of an external task.
	// Returns progress (0-100), completed status, output (if completed), and error.
	PollStatus(ctx context.Context, task *Task) (progress int, completed bool, output map[string]any, err error)
}

// Manager manages async tasks with pluggable executors.
type Manager struct {
	mu sync.RWMutex

	repo      Repository
	executors map[string]Executor
	pollers   map[string]ExternalTaskPoller
	logger    *zap.Logger

	// Configuration
	config *Config

	// Concurrency control
	semaphore     chan struct{}
	maxConcurrent int

	// Progress subscriptions
	subscribers map[uuid.UUID][]func(*Task)

	// Lifecycle
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// Config contains manager configuration.
type Config struct {
	MaxConcurrent   int           `json:"max_concurrent" yaml:"max_concurrent"`
	PollInterval    time.Duration `json:"poll_interval" yaml:"poll_interval"`
	PollTimeout     time.Duration `json:"poll_timeout" yaml:"poll_timeout"`
	MaxPollAttempts int           `json:"max_poll_attempts" yaml:"max_poll_attempts"`
}

// DefaultConfig returns the default manager configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrent:   10,
		PollInterval:    5 * time.Second,
		PollTimeout:     30 * time.Minute,
		MaxPollAttempts: 360, // 30 minutes at 5 second intervals
	}
}

// NewManager creates a new task manager.
func NewManager(repo Repository, logger *zap.Logger, config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Manager{
		repo:          repo,
		executors:     make(map[string]Executor),
		pollers:       make(map[string]ExternalTaskPoller),
		logger:        logger.Named("task-manager"),
		config:        config,
		semaphore:     make(chan struct{}, config.MaxConcurrent),
		maxConcurrent: config.MaxConcurrent,
		subscribers:   make(map[uuid.UUID][]func(*Task)),
		stopCh:        make(chan struct{}),
	}
}

// Start starts the task manager and recovers pending tasks.
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("starting task manager",
		zap.Int("max_concurrent", m.config.MaxConcurrent),
		zap.Duration("poll_interval", m.config.PollInterval))

	// Recover pending tasks
	if err := m.recoverPendingTasks(ctx); err != nil {
		return fmt.Errorf("recover pending tasks: %w", err)
	}

	// Recover external polling tasks
	if err := m.recoverExternalTasks(ctx); err != nil {
		return fmt.Errorf("recover external tasks: %w", err)
	}

	return nil
}

// Stop stops the task manager gracefully.
func (m *Manager) Stop() {
	m.logger.Info("stopping task manager")
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info("task manager stopped")
}

// RegisterExecutor registers a task executor for a specific task type.
func (m *Manager) RegisterExecutor(taskType string, executor Executor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executors[taskType] = executor
	m.logger.Debug("registered executor", zap.String("task_type", taskType))
}

// RegisterPoller registers an external task poller for a task type.
func (m *Manager) RegisterPoller(taskType string, poller ExternalTaskPoller) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollers[taskType] = poller
	m.logger.Debug("registered poller", zap.String("task_type", taskType))
}

// Submit submits a new task for immediate execution.
func (m *Manager) Submit(ctx context.Context, ownerID uuid.UUID, req *SubmitRequest) (*Task, error) {
	task := &Task{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Type:      req.Type,
		Status:    StatusPending,
		Progress:  0,
		Input:     req.Payload,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	m.logger.Debug("task submitted",
		zap.String("task_id", task.ID.String()),
		zap.String("type", task.Type),
		zap.String("owner_id", ownerID.String()))

	// Start execution in background
	m.wg.Add(1)
	go m.executeTask(task)

	return task, nil
}

// SubmitExternal submits a task that requires external polling.
func (m *Manager) SubmitExternal(ctx context.Context, ownerID uuid.UUID, req *ExternalSubmitRequest) (*Task, error) {
	task := &Task{
		ID:             uuid.New(),
		OwnerID:        ownerID,
		Type:           req.Type,
		Status:         StatusRunning, // External tasks start as running
		Progress:       0,
		Input:          req.Payload,
		Metadata:       req.Metadata,
		ExternalTaskID: req.ExternalTaskID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := m.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	m.logger.Debug("external task submitted",
		zap.String("task_id", task.ID.String()),
		zap.String("type", task.Type),
		zap.String("external_task_id", req.ExternalTaskID))

	// Start polling in background
	m.wg.Add(1)
	go m.pollExternalTask(task)

	return task, nil
}

// Get retrieves a task by ID.
func (m *Manager) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	return m.repo.Get(ctx, id)
}

// List lists tasks for an owner.
func (m *Manager) List(ctx context.Context, ownerID uuid.UUID, filter *Filter) ([]*Task, error) {
	if filter == nil {
		filter = &Filter{}
	}
	filter.OwnerID = &ownerID
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

	m.logger.Debug("task cancelled", zap.String("task_id", id.String()))
	m.notifySubscribers(task)
	return nil
}

// Subscribe subscribes to task progress updates.
// Returns an unsubscribe function.
func (m *Manager) Subscribe(id uuid.UUID, callback func(*Task)) func() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscribers[id] = append(m.subscribers[id], callback)

	// Return unsubscribe function
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		subs := m.subscribers[id]
		for i, sub := range subs {
			if &sub == &callback {
				m.subscribers[id] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		// Clean up empty subscriber lists
		if len(m.subscribers[id]) == 0 {
			delete(m.subscribers, id)
		}
	}
}

// recoverPendingTasks recovers pending tasks after restart.
func (m *Manager) recoverPendingTasks(ctx context.Context) error {
	tasks, err := m.repo.ListPendingOrRunning(ctx)
	if err != nil {
		return fmt.Errorf("list pending tasks: %w", err)
	}

	m.logger.Info("recovering pending tasks", zap.Int("count", len(tasks)))

	for _, task := range tasks {
		// Skip tasks with external IDs (handled by recoverExternalTasks)
		if task.ExternalTaskID != "" {
			continue
		}

		// Reset running tasks to pending
		if task.Status == StatusRunning {
			task.Status = StatusPending
			task.UpdatedAt = time.Now()
			if err := m.repo.Update(ctx, task); err != nil {
				m.logger.Warn("failed to reset task status",
					zap.String("task_id", task.ID.String()),
					zap.Error(err))
				continue
			}
		}

		// Re-queue for execution
		m.wg.Add(1)
		go m.executeTask(task)
	}

	return nil
}

// recoverExternalTasks recovers external tasks that were polling when server stopped.
func (m *Manager) recoverExternalTasks(ctx context.Context) error {
	tasks, err := m.repo.ListByExternalTaskID(ctx)
	if err != nil {
		return fmt.Errorf("list external tasks: %w", err)
	}

	m.logger.Info("recovering external tasks", zap.Int("count", len(tasks)))

	for _, task := range tasks {
		if task.Status == StatusRunning && task.ExternalTaskID != "" {
			// Resume polling
			m.wg.Add(1)
			go m.pollExternalTask(task)
		}
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
		m.failTask(ctx, task, "unknown_task_type", "no executor registered for task type: "+task.Type)
		return
	}

	// Update status to running
	task.Status = StatusRunning
	task.UpdatedAt = time.Now()
	if err := m.repo.Update(ctx, task); err != nil {
		m.logger.Error("failed to update task status",
			zap.String("task_id", task.ID.String()),
			zap.Error(err))
		return
	}
	m.notifySubscribers(task)

	// Execute with progress callback
	onProgress := func(progress int, output map[string]any) {
		task.Progress = progress
		if output != nil {
			task.Output = output
		}
		task.UpdatedAt = time.Now()
		_ = m.repo.UpdateStatus(ctx, task.ID, task.Status, progress)
		m.notifySubscribers(task)
	}

	if err := executor(ctx, task, onProgress); err != nil {
		m.failTask(ctx, task, "execution_failed", err.Error())
		return
	}

	// Complete
	task.Status = StatusCompleted
	task.Progress = 100
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := m.repo.Update(ctx, task); err != nil {
		m.logger.Error("failed to update completed task",
			zap.String("task_id", task.ID.String()),
			zap.Error(err))
		return
	}

	m.logger.Debug("task completed", zap.String("task_id", task.ID.String()))
	m.notifySubscribers(task)
}

// pollExternalTask polls an external task until completion.
func (m *Manager) pollExternalTask(task *Task) {
	defer m.wg.Done()

	ctx := context.Background()

	// Get poller for task type
	m.mu.RLock()
	poller, ok := m.pollers[task.Type]
	m.mu.RUnlock()

	if !ok {
		m.failTask(ctx, task, "unknown_poller", "no poller registered for task type: "+task.Type)
		return
	}

	ticker := time.NewTicker(m.config.PollInterval)
	defer ticker.Stop()

	timeout := time.NewTimer(m.config.PollTimeout)
	defer timeout.Stop()

	attempts := 0

	for {
		select {
		case <-m.stopCh:
			return

		case <-timeout.C:
			m.failTask(ctx, task, "timeout", "task polling timed out")
			return

		case <-ticker.C:
			attempts++
			if attempts > m.config.MaxPollAttempts {
				m.failTask(ctx, task, "max_attempts", "exceeded maximum poll attempts")
				return
			}

			// Poll external task status
			pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			progress, completed, output, err := poller.PollStatus(pollCtx, task)
			cancel()

			if err != nil {
				m.logger.Warn("poll error",
					zap.String("task_id", task.ID.String()),
					zap.Error(err))
				continue
			}

			// Update progress
			if progress != task.Progress {
				task.Progress = progress
				task.UpdatedAt = time.Now()
				_ = m.repo.UpdateStatus(ctx, task.ID, task.Status, progress)
				m.notifySubscribers(task)
			}

			// Check if completed
			if completed {
				task.Status = StatusCompleted
				task.Progress = 100
				task.Output = output
				now := time.Now()
				task.CompletedAt = &now
				task.UpdatedAt = now

				if err := m.repo.Update(ctx, task); err != nil {
					m.logger.Error("failed to update completed external task",
						zap.String("task_id", task.ID.String()),
						zap.Error(err))
					return
				}

				m.logger.Debug("external task completed",
					zap.String("task_id", task.ID.String()),
					zap.String("external_task_id", task.ExternalTaskID))
				m.notifySubscribers(task)
				return
			}
		}
	}
}

// failTask marks a task as failed.
func (m *Manager) failTask(ctx context.Context, task *Task, code, message string) {
	task.Status = StatusFailed
	task.Error = &Error{
		Code:    code,
		Message: message,
	}
	task.UpdatedAt = time.Now()

	if err := m.repo.Update(ctx, task); err != nil {
		m.logger.Error("failed to update failed task",
			zap.String("task_id", task.ID.String()),
			zap.Error(err))
	}

	m.logger.Warn("task failed",
		zap.String("task_id", task.ID.String()),
		zap.String("code", code),
		zap.String("message", message))
	m.notifySubscribers(task)
}

// notifySubscribers notifies all subscribers of a task update.
func (m *Manager) notifySubscribers(task *Task) {
	m.mu.RLock()
	subs := make([]func(*Task), len(m.subscribers[task.ID]))
	copy(subs, m.subscribers[task.ID])
	m.mu.RUnlock()

	for _, sub := range subs {
		sub(task)
	}
}
