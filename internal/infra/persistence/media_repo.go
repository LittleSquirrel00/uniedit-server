package persistence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/domain/media"
	"github.com/uniedit/server/internal/infra/persistence/entity"
)

// MediaTaskRepository implements media.TaskRepository.
type MediaTaskRepository struct {
	db *gorm.DB
}

// NewMediaTaskRepository creates a new media task repository.
func NewMediaTaskRepository(db *gorm.DB) *MediaTaskRepository {
	return &MediaTaskRepository{db: db}
}

var _ media.TaskRepository = (*MediaTaskRepository)(nil)

// Create creates a new task.
func (r *MediaTaskRepository) Create(ctx context.Context, task *media.Task) error {
	e := entity.FromDomainMediaTask(task)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("create media task: %w", err)
	}
	return nil
}

// GetByID retrieves a task by ID.
func (r *MediaTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*media.Task, error) {
	var e entity.MediaTaskEntity
	if err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, media.ErrTaskNotFound
		}
		return nil, fmt.Errorf("get media task by ID: %w", err)
	}
	return e.ToDomain(), nil
}

// Update updates a task.
func (r *MediaTaskRepository) Update(ctx context.Context, task *media.Task) error {
	e := entity.FromDomainMediaTask(task)
	if err := r.db.WithContext(ctx).Save(e).Error; err != nil {
		return fmt.Errorf("update media task: %w", err)
	}
	return nil
}

// ListByOwner lists tasks by owner.
func (r *MediaTaskRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*media.Task, error) {
	var entities []entity.MediaTaskEntity
	if err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("list media tasks by owner: %w", err)
	}

	tasks := make([]*media.Task, len(entities))
	for i, e := range entities {
		tasks[i] = e.ToDomain()
	}
	return tasks, nil
}

// ListPending lists pending tasks.
func (r *MediaTaskRepository) ListPending(ctx context.Context, limit int) ([]*media.Task, error) {
	var entities []entity.MediaTaskEntity
	if err := r.db.WithContext(ctx).
		Where("status = ?", media.TaskStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("list pending media tasks: %w", err)
	}

	tasks := make([]*media.Task, len(entities))
	for i, e := range entities {
		tasks[i] = e.ToDomain()
	}
	return tasks, nil
}
