package media

import (
	"time"

	commonv1 "github.com/uniedit/server/api/pb/common"
	mediav1 "github.com/uniedit/server/api/pb/media"
	"github.com/uniedit/server/internal/model"
)

// toGenerateImageResponseFromResult builds a GenerateImageResponse from adapter result.
func toGenerateImageResponseFromResult(resp *model.ImageResponse) *mediav1.GenerateImageResponse {
	if resp == nil {
		return nil
	}

	out := &mediav1.GenerateImageResponse{
		Model:     resp.Model,
		CreatedAt: resp.CreatedAt,
		Images:    make([]*mediav1.GeneratedImage, 0, len(resp.Images)),
	}

	for _, img := range resp.Images {
		if img == nil {
			continue
		}
		out.Images = append(out.Images, &mediav1.GeneratedImage{
			Url:           img.URL,
			B64Json:       img.B64JSON,
			RevisedPrompt: img.RevisedPrompt,
		})
	}

	if resp.Usage != nil {
		out.Usage = &mediav1.ImageUsage{
			TotalImages: int32(resp.Usage.TotalImages),
			CostUsd:     resp.Usage.CostUSD,
		}
	}

	return out
}

// toVideoStatusFromTask builds a VideoGenerationStatus from a MediaTask.
func toVideoStatusFromTask(task *model.MediaTask, video *model.GeneratedVideo) *mediav1.VideoGenerationStatus {
	if task == nil {
		return nil
	}

	out := &mediav1.VideoGenerationStatus{
		TaskId:    task.ID.String(),
		Status:    string(taskStatusToVideoState(task.Status)),
		Progress:  int32(task.Progress),
		CreatedAt: task.CreatedAt.Unix(),
	}

	if task.Error != "" {
		out.Error = task.Error
	}

	if video != nil {
		out.Video = &mediav1.GeneratedVideo{
			Url:      video.URL,
			Duration: int32(video.Duration),
			Width:    int32(video.Width),
			Height:   int32(video.Height),
			Fps:      int32(video.FPS),
			FileSize: video.FileSize,
			Format:   video.Format,
		}
	}

	return out
}

// toVideoStatusPending builds a pending VideoGenerationStatus.
func toVideoStatusPending(task *model.MediaTask) *mediav1.VideoGenerationStatus {
	return &mediav1.VideoGenerationStatus{
		TaskId:    task.ID.String(),
		Status:    string(model.VideoStatePending),
		Progress:  0,
		CreatedAt: task.CreatedAt.Unix(),
	}
}

// toMediaTaskPB converts a MediaTask model to proto.
func toMediaTaskPB(task *model.MediaTask) *mediav1.MediaTask {
	if task == nil {
		return nil
	}
	return &mediav1.MediaTask{
		Id:        task.ID.String(),
		OwnerId:   task.OwnerID.String(),
		Type:      task.Type,
		Status:    string(task.Status),
		Progress:  int32(task.Progress),
		Error:     task.Error,
		CreatedAt: task.CreatedAt.Unix(),
		UpdatedAt: task.UpdatedAt.Unix(),
	}
}

func toProvider(in *model.MediaProvider) *mediav1.MediaProvider {
	if in == nil {
		return nil
	}
	return &mediav1.MediaProvider{
		Id:        in.ID.String(),
		Name:      in.Name,
		Type:      string(in.Type),
		BaseUrl:   in.BaseURL,
		Enabled:   in.Enabled,
		CreatedAt: formatTime(in.CreatedAt),
		UpdatedAt: formatTime(in.UpdatedAt),
	}
}

func toModel(in *model.MediaModel) *mediav1.MediaModel {
	if in == nil {
		return nil
	}

	caps := make([]string, 0, len(in.Capabilities))
	for _, c := range in.Capabilities {
		caps = append(caps, string(c))
	}

	return &mediav1.MediaModel{
		Id:           in.ID,
		ProviderId:   in.ProviderID.String(),
		Name:         in.Name,
		Capabilities: caps,
		Enabled:      in.Enabled,
		CreatedAt:    formatTime(in.CreatedAt),
		UpdatedAt:    formatTime(in.UpdatedAt),
	}
}

func empty() *commonv1.Empty { return &commonv1.Empty{} }

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
