package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	tasks   map[uuid.UUID]*Task
	err     error
	created []*Task
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		tasks:   make(map[uuid.UUID]*Task),
		created: make([]*Task, 0),
	}
}

func (m *MockRepository) Create(_ context.Context, task *Task) error {
	if m.err != nil {
		return m.err
	}
	m.tasks[task.ID] = task
	m.created = append(m.created, task)
	return nil
}

func (m *MockRepository) Get(_ context.Context, id uuid.UUID) (*Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	task, ok := m.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *MockRepository) GetByExternalID(_ context.Context, externalID string) (*Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, task := range m.tasks {
		if task.ExternalTaskID == externalID {
			return task, nil
		}
	}
	return nil, ErrTaskNotFound
}

func (m *MockRepository) List(_ context.Context, filter *Filter) ([]*Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*Task
	for _, task := range m.tasks {
		if filter != nil {
			if filter.UserID != nil && task.UserID != *filter.UserID {
				continue
			}
			if filter.Type != nil && task.Type != *filter.Type {
				continue
			}
			if filter.Status != nil && task.Status != *filter.Status {
				continue
			}
		}
		result = append(result, task)
	}
	return result, nil
}

func (m *MockRepository) Update(_ context.Context, task *Task) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.tasks[task.ID]; !ok {
		return ErrTaskNotFound
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockRepository) UpdateStatus(_ context.Context, id uuid.UUID, status Status, progress int) error {
	if m.err != nil {
		return m.err
	}
	task, ok := m.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	task.Status = status
	task.Progress = progress
	return nil
}

func (m *MockRepository) Delete(_ context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.tasks[id]; !ok {
		return ErrTaskNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *MockRepository) ListPendingOrRunning(_ context.Context) ([]*Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*Task
	for _, task := range m.tasks {
		if task.Status == StatusPending || task.Status == StatusRunning {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *MockRepository) ListByExternalTaskID(_ context.Context) ([]*Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*Task
	for _, task := range m.tasks {
		if task.ExternalTaskID != "" && task.Status == StatusRunning {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *MockRepository) CountByUserAndStatus(_ context.Context, userID uuid.UUID, status Status) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	var count int64
	for _, task := range m.tasks {
		if task.UserID == userID && task.Status == status {
			count++
		}
	}
	return count, nil
}

// MockExecutor for testing.
type MockExecutor struct {
	err      error
	executed bool
}

func (e *MockExecutor) Execute(_ context.Context, _ *Task, _ func(int)) error {
	e.executed = true
	return e.err
}

// MockPoller for testing.
type MockPoller struct {
	progress  int
	completed bool
	output    map[string]any
	err       error
}

func (p *MockPoller) PollStatus(_ context.Context, _ *Task) (int, bool, map[string]any, error) {
	return p.progress, p.completed, p.output, p.err
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 10, config.MaxConcurrent)
	assert.Equal(t, 5*time.Second, config.PollInterval)
	assert.Equal(t, 30*time.Minute, config.PollTimeout)
	assert.Equal(t, 360, config.MaxPollAttempts)
}

func TestNewManager(t *testing.T) {
	t.Run("Creates with default config", func(t *testing.T) {
		repo := NewMockRepository()
		manager := NewManager(repo, nil)

		assert.NotNil(t, manager)
		assert.NotNil(t, manager.executors)
		assert.NotNil(t, manager.pollers)
		assert.NotNil(t, manager.subscribers)
	})

	t.Run("Creates with custom config", func(t *testing.T) {
		repo := NewMockRepository()
		config := &ManagerConfig{
			MaxConcurrent: 5,
			PollInterval:  10 * time.Second,
		}
		manager := NewManager(repo, config)

		assert.NotNil(t, manager)
		assert.Equal(t, 5, manager.maxConcurrent)
	})
}

func TestManager_RegisterExecutor(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	executor := func(_ context.Context, _ *Task, _ func(int)) error {
		return nil
	}

	manager.RegisterExecutor(TypeChat, executor)

	manager.mu.RLock()
	_, ok := manager.executors[TypeChat]
	manager.mu.RUnlock()

	assert.True(t, ok)
}

func TestManager_RegisterPoller(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	poller := &MockPoller{}
	manager.RegisterPoller(TypeImageGeneration, poller)

	manager.mu.RLock()
	_, ok := manager.pollers[TypeImageGeneration]
	manager.mu.RUnlock()

	assert.True(t, ok)
}

func TestManager_Get(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	taskID := uuid.New()
	userID := uuid.New()
	task := &Task{
		ID:     taskID,
		UserID: userID,
		Type:   TypeChat,
		Status: StatusPending,
	}
	repo.tasks[taskID] = task

	t.Run("Returns task by ID", func(t *testing.T) {
		result, err := manager.Get(context.Background(), taskID)
		require.NoError(t, err)
		assert.Equal(t, taskID, result.ID)
	})

	t.Run("Returns error for non-existing task", func(t *testing.T) {
		_, err := manager.Get(context.Background(), uuid.New())
		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
	})
}

func TestManager_List(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	userID := uuid.New()
	otherUserID := uuid.New()

	repo.tasks[uuid.New()] = &Task{ID: uuid.New(), UserID: userID, Type: TypeChat, Status: StatusPending}
	repo.tasks[uuid.New()] = &Task{ID: uuid.New(), UserID: userID, Type: TypeChat, Status: StatusCompleted}
	repo.tasks[uuid.New()] = &Task{ID: uuid.New(), UserID: otherUserID, Type: TypeChat, Status: StatusPending}

	t.Run("Lists tasks for user", func(t *testing.T) {
		tasks, err := manager.List(context.Background(), userID, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, len(tasks))
	})

	t.Run("Lists tasks with status filter", func(t *testing.T) {
		status := StatusPending
		tasks, err := manager.List(context.Background(), userID, &Filter{Status: &status})
		require.NoError(t, err)
		assert.Equal(t, 1, len(tasks))
	})
}

func TestManager_Cancel(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	t.Run("Cancels pending task", func(t *testing.T) {
		taskID := uuid.New()
		task := &Task{
			ID:     taskID,
			UserID: uuid.New(),
			Type:   TypeChat,
			Status: StatusPending,
		}
		repo.tasks[taskID] = task

		err := manager.Cancel(context.Background(), taskID)
		require.NoError(t, err)
		assert.Equal(t, StatusCancelled, repo.tasks[taskID].Status)
	})

	t.Run("Cancels running task", func(t *testing.T) {
		taskID := uuid.New()
		task := &Task{
			ID:     taskID,
			UserID: uuid.New(),
			Type:   TypeChat,
			Status: StatusRunning,
		}
		repo.tasks[taskID] = task

		err := manager.Cancel(context.Background(), taskID)
		require.NoError(t, err)
		assert.Equal(t, StatusCancelled, repo.tasks[taskID].Status)
	})

	t.Run("Cannot cancel completed task", func(t *testing.T) {
		taskID := uuid.New()
		task := &Task{
			ID:     taskID,
			UserID: uuid.New(),
			Type:   TypeChat,
			Status: StatusCompleted,
		}
		repo.tasks[taskID] = task

		err := manager.Cancel(context.Background(), taskID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "terminal state")
	})

	t.Run("Returns error for non-existing task", func(t *testing.T) {
		err := manager.Cancel(context.Background(), uuid.New())
		assert.Error(t, err)
	})
}

func TestManager_OnProgress(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	taskID := uuid.New()
	callCount := 0

	unsubscribe := manager.OnProgress(taskID, func(task *Task) {
		callCount++
	})

	assert.NotNil(t, unsubscribe)

	manager.mu.RLock()
	subs := manager.subscribers[taskID]
	manager.mu.RUnlock()

	assert.Equal(t, 1, len(subs))

	// Call unsubscribe
	unsubscribe()
}

func TestManager_RecoverPendingTasks(t *testing.T) {
	repo := NewMockRepository()
	config := &ManagerConfig{
		MaxConcurrent:   1,
		PollInterval:    100 * time.Millisecond,
		PollTimeout:     time.Second,
		MaxPollAttempts: 5,
	}
	manager := NewManager(repo, config)

	// Add a pending task
	taskID := uuid.New()
	task := &Task{
		ID:     taskID,
		UserID: uuid.New(),
		Type:   TypeChat,
		Status: StatusPending,
	}
	repo.tasks[taskID] = task

	// Register executor that does nothing
	manager.RegisterExecutor(TypeChat, func(_ context.Context, _ *Task, _ func(int)) error {
		return nil
	})

	err := manager.RecoverPendingTasks(context.Background())
	assert.NoError(t, err)

	// Stop manager to clean up goroutines
	manager.Stop()
}

func TestManager_RecoverExternalTasks(t *testing.T) {
	repo := NewMockRepository()
	config := &ManagerConfig{
		MaxConcurrent:   1,
		PollInterval:    100 * time.Millisecond,
		PollTimeout:     time.Second,
		MaxPollAttempts: 5,
	}
	manager := NewManager(repo, config)

	// Add an external task
	taskID := uuid.New()
	task := &Task{
		ID:             taskID,
		UserID:         uuid.New(),
		Type:           TypeImageGeneration,
		Status:         StatusRunning,
		ExternalTaskID: "ext-123",
	}
	repo.tasks[taskID] = task

	// Register poller that completes immediately
	poller := &MockPoller{
		progress:  100,
		completed: true,
		output:    map[string]any{"result": "done"},
	}
	manager.RegisterPoller(TypeImageGeneration, poller)

	err := manager.RecoverExternalTasks(context.Background())
	assert.NoError(t, err)

	// Stop manager
	manager.Stop()
}

func TestManager_Submit(t *testing.T) {
	repo := NewMockRepository()
	config := &ManagerConfig{
		MaxConcurrent:   1,
		PollInterval:    100 * time.Millisecond,
		PollTimeout:     time.Second,
		MaxPollAttempts: 5,
	}
	manager := NewManager(repo, config)

	// Register executor
	executed := false
	manager.RegisterExecutor(TypeChat, func(_ context.Context, _ *Task, _ func(int)) error {
		executed = true
		return nil
	})

	userID := uuid.New()
	input := &Input{
		Type:    TypeChat,
		Payload: map[string]any{"messages": []string{"Hello"}},
	}

	task, err := manager.Submit(context.Background(), userID, input)
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, TypeChat, task.Type)
	assert.Equal(t, StatusPending, task.Status)

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	// Stop manager
	manager.Stop()

	assert.True(t, executed)
}

func TestManager_SubmitExternal(t *testing.T) {
	repo := NewMockRepository()
	config := &ManagerConfig{
		MaxConcurrent:   1,
		PollInterval:    50 * time.Millisecond,
		PollTimeout:     time.Second,
		MaxPollAttempts: 5,
	}
	manager := NewManager(repo, config)

	// Register poller
	poller := &MockPoller{
		progress:  100,
		completed: true,
		output:    map[string]any{"result": "done"},
	}
	manager.RegisterPoller(TypeImageGeneration, poller)

	userID := uuid.New()
	providerID := uuid.New()
	input := &Input{
		Type:    TypeImageGeneration,
		Payload: map[string]any{"prompt": "A cat"},
	}

	task, err := manager.SubmitExternal(context.Background(), userID, input, "ext-123", &providerID, "dall-e-3")
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, TypeImageGeneration, task.Type)
	assert.Equal(t, StatusRunning, task.Status)
	assert.Equal(t, "ext-123", task.ExternalTaskID)

	// Wait for polling to complete
	time.Sleep(200 * time.Millisecond)

	// Stop manager
	manager.Stop()
}

func TestManager_failTask(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	taskID := uuid.New()
	task := &Task{
		ID:     taskID,
		UserID: uuid.New(),
		Type:   TypeChat,
		Status: StatusRunning,
	}
	repo.tasks[taskID] = task

	manager.failTask(context.Background(), task, "test error")

	assert.Equal(t, StatusFailed, task.Status)
	assert.NotNil(t, task.Error)
	assert.Equal(t, "execution_failed", task.Error.Code)
	assert.Equal(t, "test error", task.Error.Message)
}

func TestManager_notifySubscribers(t *testing.T) {
	repo := NewMockRepository()
	manager := NewManager(repo, nil)

	taskID := uuid.New()
	task := &Task{
		ID:     taskID,
		UserID: uuid.New(),
		Type:   TypeChat,
		Status: StatusRunning,
	}

	callCount := 0
	manager.OnProgress(taskID, func(t *Task) {
		callCount++
	})

	manager.notifySubscribers(task)

	assert.Equal(t, 1, callCount)
}

func TestManager_SubmitWithError(t *testing.T) {
	repo := NewMockRepository()
	repo.err = errors.New("database error")

	manager := NewManager(repo, nil)

	userID := uuid.New()
	input := &Input{
		Type:    TypeChat,
		Payload: map[string]any{"messages": []string{"Hello"}},
	}

	_, err := manager.Submit(context.Background(), userID, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create task")
}
