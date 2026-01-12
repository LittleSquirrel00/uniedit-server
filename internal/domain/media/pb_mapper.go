package media

import (
	"time"

	commonv1 "github.com/uniedit/server/api/pb/common"
	mediav1 "github.com/uniedit/server/api/pb/media"
	"github.com/uniedit/server/internal/model"
)

func toGenerateImageInput(in *mediav1.GenerateImageRequest) *model.MediaImageGenerationInput {
	if in == nil {
		return &model.MediaImageGenerationInput{}
	}
	return &model.MediaImageGenerationInput{
		Prompt:         in.GetPrompt(),
		NegativePrompt: in.GetNegativePrompt(),
		N:              int(in.GetN()),
		Size:           in.GetSize(),
		Quality:        in.GetQuality(),
		Style:          in.GetStyle(),
		Model:          in.GetModel(),
		ResponseFormat: in.GetResponseFormat(),
	}
}

func toGenerateVideoInput(in *mediav1.GenerateVideoRequest) *model.MediaVideoGenerationInput {
	if in == nil {
		return &model.MediaVideoGenerationInput{}
	}
	return &model.MediaVideoGenerationInput{
		Prompt:      in.GetPrompt(),
		InputImage:  in.GetInputImage(),
		InputVideo:  in.GetInputVideo(),
		Duration:    int(in.GetDuration()),
		AspectRatio: in.GetAspectRatio(),
		Resolution:  in.GetResolution(),
		FPS:         int(in.GetFps()),
		Model:       in.GetModel(),
	}
}

func toGenerateImageResponse(in *model.MediaImageGenerationOutput) *mediav1.GenerateImageResponse {
	if in == nil {
		return nil
	}

	out := &mediav1.GenerateImageResponse{
		Model:     in.Model,
		CreatedAt: in.CreatedAt,
		TaskId:    in.TaskID,
		Images:    make([]*mediav1.GeneratedImage, 0, len(in.Images)),
	}

	for _, img := range in.Images {
		if img == nil {
			continue
		}
		out.Images = append(out.Images, &mediav1.GeneratedImage{
			Url:           img.URL,
			B64Json:       img.B64JSON,
			RevisedPrompt: img.RevisedPrompt,
		})
	}

	if in.Usage != nil {
		out.Usage = &mediav1.ImageUsage{
			TotalImages: int32(in.Usage.TotalImages),
			CostUsd:     in.Usage.CostUSD,
		}
	}

	return out
}

func toVideoStatus(in *model.MediaVideoGenerationOutput) *mediav1.VideoGenerationStatus {
	if in == nil {
		return nil
	}

	out := &mediav1.VideoGenerationStatus{
		TaskId:    in.TaskID,
		Status:    string(in.Status),
		Progress:  int32(in.Progress),
		Error:     in.Error,
		CreatedAt: in.CreatedAt,
	}

	if in.Video != nil {
		out.Video = &mediav1.GeneratedVideo{
			Url:      in.Video.URL,
			Duration: int32(in.Video.Duration),
			Width:    int32(in.Video.Width),
			Height:   int32(in.Video.Height),
			Fps:      int32(in.Video.FPS),
			FileSize: in.Video.FileSize,
			Format:   in.Video.Format,
		}
	}

	return out
}

func toTask(in *model.MediaTaskOutput) *mediav1.MediaTask {
	if in == nil {
		return nil
	}
	return &mediav1.MediaTask{
		Id:        in.ID.String(),
		OwnerId:   in.OwnerID.String(),
		Type:      in.Type,
		Status:    string(in.Status),
		Progress:  int32(in.Progress),
		Error:     in.Error,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
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

