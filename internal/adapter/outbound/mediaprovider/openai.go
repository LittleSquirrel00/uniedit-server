package mediaprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// OpenAIAdapter implements the MediaVendorAdapterPort for OpenAI DALL-E.
type OpenAIAdapter struct {
	client *http.Client
}

// NewOpenAIAdapter creates a new OpenAI media adapter with the given HTTP client.
func NewOpenAIAdapter(client *http.Client) *OpenAIAdapter {
	return &OpenAIAdapter{
		client: client,
	}
}

// Type returns the provider type.
func (a *OpenAIAdapter) Type() model.MediaProviderType {
	return model.MediaProviderTypeOpenAI
}

// SupportsCapability checks if the adapter supports a capability.
func (a *OpenAIAdapter) SupportsCapability(cap model.MediaCapability) bool {
	switch cap {
	case model.MediaCapabilityImage:
		return true
	default:
		return false
	}
}

// openAIImageRequest represents an OpenAI image generation request.
type openAIImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// openAIImageResponse represents an OpenAI image generation response.
type openAIImageResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url,omitempty"`
		B64JSON       string `json:"b64_json,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateImage generates images using OpenAI DALL-E.
func (a *OpenAIAdapter) GenerateImage(ctx context.Context, req *model.ImageRequest, m *model.MediaModel, prov *model.MediaProvider, apiKey string) (*model.ImageResponse, error) {
	// Build request
	openAIReq := &openAIImageRequest{
		Model:          m.ID,
		Prompt:         req.Prompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		ResponseFormat: req.ResponseFormat,
	}

	// Set defaults
	if openAIReq.N == 0 {
		openAIReq.N = 1
	}
	if openAIReq.Size == "" {
		openAIReq.Size = "1024x1024"
	}
	if openAIReq.ResponseFormat == "" {
		openAIReq.ResponseFormat = "url"
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := prov.BaseURL + "/v1/images/generations"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute request
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Parse response
	var openAIResp openAIImageResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Check for errors
	if openAIResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", openAIResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Convert to ImageResponse
	images := make([]*model.GeneratedImage, len(openAIResp.Data))
	for i, data := range openAIResp.Data {
		images[i] = &model.GeneratedImage{
			URL:           data.URL,
			B64JSON:       data.B64JSON,
			RevisedPrompt: data.RevisedPrompt,
		}
	}

	return &model.ImageResponse{
		Images:    images,
		Model:     m.ID,
		CreatedAt: openAIResp.Created,
		Usage: &model.ImageUsage{
			TotalImages: len(images),
		},
	}, nil
}

// GenerateVideo is not supported by OpenAI image adapter.
func (a *OpenAIAdapter) GenerateVideo(ctx context.Context, req *model.VideoRequest, m *model.MediaModel, prov *model.MediaProvider, apiKey string) (*model.VideoResponse, error) {
	return nil, fmt.Errorf("video generation not supported by OpenAI adapter")
}

// GetVideoStatus is not supported by OpenAI image adapter.
func (a *OpenAIAdapter) GetVideoStatus(ctx context.Context, taskID string, prov *model.MediaProvider, apiKey string) (*model.VideoStatus, error) {
	return nil, fmt.Errorf("video status not supported by OpenAI adapter")
}

// HealthCheck performs a health check.
func (a *OpenAIAdapter) HealthCheck(ctx context.Context, prov *model.MediaProvider, apiKey string) error {
	url := prov.BaseURL + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// Compile-time interface check
var _ outbound.MediaVendorAdapterPort = (*OpenAIAdapter)(nil)
