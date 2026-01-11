package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrTaskNotFound is returned when a task is not found.
	ErrTaskNotFound = errors.New("task not found")
)

// Repository defines the interface for task data access.
type Repository interface {
	Create(ctx context.Context, task *Task) error
	Get(ctx context.Context, id uuid.UUID) (*Task, error)
	GetByExternalID(ctx context.Context, externalID string) (*Task, error)
	List(ctx context.Context, filter *Filter) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status, progress int) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListPendingOrRunning(ctx context.Context) ([]*Task, error)
	ListByExternalTaskID(ctx context.Context) ([]*Task, error)
	CountByOwnerAndStatus(ctx context.Context, ownerID uuid.UUID, status Status) (int64, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new task repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create creates a new task.
func (r *repository) Create(ctx context.Context, task *Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// Get retrieves a task by ID.
func (r *repository) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	var task Task
	err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &task, nil
}

// GetByExternalID retrieves a task by external task ID.
func (r *repository) GetByExternalID(ctx context.Context, externalID string) (*Task, error) {
	var task Task
	err := r.db.WithContext(ctx).First(&task, "external_task_id = ?", externalID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by external id: %w", err)
	}
	return &task, nil
}

// List lists tasks with optional filters.
func (r *repository) List(ctx context.Context, filter *Filter) ([]*Task, error) {
	var tasks []*Task
	query := r.db.WithContext(ctx)

	if filter != nil {
		if filter.OwnerID != nil {
			query = query.Where("owner_id = ?", *filter.OwnerID)
		}
		if filter.Type != nil {
			query = query.Where("type = ?", *filter.Type)
		}
		if filter.Status != nil {
			query = query.Where("status = ?", *filter.Status)
		}

		orderBy := "created_at"
		if filter.OrderBy != "" {
			orderBy = filter.OrderBy
		}
		orderDir := "DESC"
		if filter.OrderDir != "" {
			orderDir = filter.OrderDir
		}
		query = query.Order(fmt.Sprintf("%s %s", orderBy, orderDir))

		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	} else {
		query = query.Order("created_at DESC")
	}

	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// Update updates a task.
func (r *repository) Update(ctx context.Context, task *Task) error {
	result := r.db.WithContext(ctx).Save(task)
	if result.Error != nil {
		return fmt.Errorf("update task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// UpdateStatus updates only the status and progress of a task.
func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status, progress int) error {
	result := r.db.WithContext(ctx).
		Model(&Task{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":   status,
			"progress": progress,
		})
	if result.Error != nil {
		return fmt.Errorf("update task status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// Delete deletes a task.
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&Task{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// ListPendingOrRunning lists all pending or running tasks.
func (r *repository) ListPendingOrRunning(ctx context.Context) ([]*Task, error) {
	var tasks []*Task
	err := r.db.WithContext(ctx).
		Where("status IN ?", []Status{StatusPending, StatusRunning}).
		Order("created_at ASC").
		Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("list pending or running tasks: %w", err)
	}
	return tasks, nil
}

// CountByOwnerAndStatus counts tasks for an owner with a specific status.
func (r *repository) CountByOwnerAndStatus(ctx context.Context, ownerID uuid.UUID, status Status) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Task{}).
		Where("owner_id = ? AND status = ?", ownerID, status).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}
	return count, nil
}

// ListByExternalTaskID lists all running tasks with an external task ID.
func (r *repository) ListByExternalTaskID(ctx context.Context) ([]*Task, error) {
	var tasks []*Task
	err := r.db.WithContext(ctx).
		Where("status = ? AND external_task_id IS NOT NULL AND external_task_id != ''", StatusRunning).
		Order("created_at ASC").
		Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("list external tasks: %w", err)
	}
	return tasks, nil
}
