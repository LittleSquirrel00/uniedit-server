package inbound

import (
	"context"

	"github.com/google/uuid"
	commonv1 "github.com/uniedit/server/api/pb/common"
	mediav1 "github.com/uniedit/server/api/pb/media"
)

// MediaDomain defines the media domain service interface.
type MediaDomain interface {
	// MediaService
	GenerateImage(ctx context.Context, userID uuid.UUID, in *mediav1.GenerateImageRequest) (*mediav1.GenerateImageResponse, error)
	GenerateVideo(ctx context.Context, userID uuid.UUID, in *mediav1.GenerateVideoRequest) (*mediav1.VideoGenerationStatus, error)
	GetVideoStatus(ctx context.Context, userID uuid.UUID, in *mediav1.GetByTaskIDRequest) (*mediav1.VideoGenerationStatus, error)
	ListTasks(ctx context.Context, userID uuid.UUID, in *mediav1.ListTasksRequest) (*mediav1.ListTasksResponse, error)
	GetTask(ctx context.Context, userID uuid.UUID, in *mediav1.GetByTaskIDRequest) (*mediav1.MediaTask, error)
	CancelTask(ctx context.Context, userID uuid.UUID, in *mediav1.GetByTaskIDRequest) (*commonv1.Empty, error)

	// MediaAdminService
	ListProviders(ctx context.Context, _ *commonv1.Empty) (*mediav1.ListProvidersResponse, error)
	GetProvider(ctx context.Context, in *mediav1.GetByIDRequest) (*mediav1.MediaProvider, error)
	ListModels(ctx context.Context, in *mediav1.ListModelsRequest) (*mediav1.ListModelsResponse, error)
}
