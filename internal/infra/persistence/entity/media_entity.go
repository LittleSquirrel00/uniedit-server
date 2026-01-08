package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/domain/media"
)

// MediaTaskEntity is the GORM entity for media tasks.
type MediaTaskEntity struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	TaskType  string         `gorm:"column:task_type;not null"`
	Status    string         `gorm:"not null;default:'pending';index"`
	Progress  int            `gorm:"default:0"`
	Input     JSON           `gorm:"type:jsonb"`
	Output    JSON           `gorm:"type:jsonb"`
	ErrorCode *string        `gorm:"column:error_code"`
	ErrorMsg  *string        `gorm:"column:error_message"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName returns the table name for MediaTaskEntity.
func (MediaTaskEntity) TableName() string {
	return "media_tasks"
}

// ToDomain converts to domain entity.
func (e *MediaTaskEntity) ToDomain() *media.Task {
	var taskErr *media.TaskError
	if e.ErrorCode != nil && e.ErrorMsg != nil {
		taskErr = media.NewTaskError(*e.ErrorCode, *e.ErrorMsg)
	}

	// Convert JSON to map
	input := make(map[string]any)
	if e.Input != nil {
		input = e.Input
	}

	output := make(map[string]any)
	if e.Output != nil {
		output = e.Output
	}

	return media.ReconstructTask(
		e.ID,
		e.OwnerID,
		media.TaskType(e.TaskType),
		media.TaskStatus(e.Status),
		e.Progress,
		input,
		output,
		taskErr,
		e.CreatedAt,
		e.UpdatedAt,
	)
}

// FromDomainMediaTask converts from domain entity.
func FromDomainMediaTask(t *media.Task) *MediaTaskEntity {
	var errorCode, errorMsg *string
	if t.Error() != nil {
		code := t.Error().Code()
		msg := t.Error().Message()
		errorCode = &code
		errorMsg = &msg
	}

	return &MediaTaskEntity{
		ID:        t.ID(),
		OwnerID:   t.OwnerID(),
		TaskType:  t.TaskType().String(),
		Status:    t.Status().String(),
		Progress:  t.Progress(),
		Input:     JSON(t.Input()),
		Output:    JSON(t.Output()),
		ErrorCode: errorCode,
		ErrorMsg:  errorMsg,
		CreatedAt: t.CreatedAt(),
		UpdatedAt: t.UpdatedAt(),
	}
}

// MediaProviderEntity is the GORM entity for media providers.
// Note: This may share the same table as AI providers.
type MediaProviderEntity struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name    string    `gorm:"not null"`
	Type    string    `gorm:"not null"`
	BaseURL string    `gorm:"column:base_url;not null"`
	APIKey  string    `gorm:"column:api_key;not null"`
	Enabled bool      `gorm:"default:true"`
}

// TableName returns the table name.
func (MediaProviderEntity) TableName() string {
	return "ai_providers"
}

// ToDomainMediaProvider converts to domain provider.
func (e *MediaProviderEntity) ToDomainMediaProvider() *media.Provider {
	return media.ReconstructProvider(
		e.ID,
		e.Name,
		media.ProviderType(e.Type),
		e.BaseURL,
		e.APIKey,
		e.Enabled,
	)
}

// MediaModelEntity is the GORM entity for media models.
// Note: This may share the same table as AI models.
type MediaModelEntity struct {
	ID           string    `gorm:"primaryKey"`
	ProviderID   uuid.UUID `gorm:"type:uuid;not null"`
	Name         string    `gorm:"not null"`
	Capabilities JSON      `gorm:"type:jsonb"`
	Enabled      bool      `gorm:"default:true"`
}

// TableName returns the table name.
func (MediaModelEntity) TableName() string {
	return "ai_models"
}

// ToDomainMediaModel converts to domain model.
func (e *MediaModelEntity) ToDomainMediaModel() *media.Model {
	var caps []media.Capability
	if e.Capabilities != nil {
		if capList, ok := e.Capabilities["list"].([]interface{}); ok {
			for _, c := range capList {
				if cs, ok := c.(string); ok {
					caps = append(caps, media.Capability(cs))
				}
			}
		}
	}

	return media.ReconstructModel(
		e.ID,
		e.ProviderID,
		e.Name,
		caps,
		e.Enabled,
	)
}
