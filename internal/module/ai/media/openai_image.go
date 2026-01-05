package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/uniedit/server/internal/module/ai/provider"
)

// OpenAIImageAdapter implements the Adapter interface for OpenAI DALL-E.
type OpenAIImageAdapter struct {
	client *http.Client
}

// NewOpenAIImageAdapter creates a new OpenAI image adapter.
func NewOpenAIImageAdapter() *OpenAIImageAdapter {
	return &OpenAIImageAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Type returns the provider type.
func (a *OpenAIImageAdapter) Type() provider.ProviderType {
	return provider.ProviderTypeOpenAI
}

// SupportsCapability checks if the adapter supports a capability.
func (a *OpenAIImageAdapter) SupportsCapability(cap provider.Capability) bool {
	switch cap {
	case provider.CapabilityImageGeneration:
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
func (a *OpenAIImageAdapter) GenerateImage(ctx context.Context, req *ImageRequest, model *provider.Model, prov *provider.Provider) (*ImageResponse, error) {
	// Build request
	openAIReq := &openAIImageRequest{
		Model:          model.ID,
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
	httpReq.Header.Set("Authorization", "Bearer "+prov.APIKey)

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
	images := make([]*GeneratedImage, len(openAIResp.Data))
	for i, data := range openAIResp.Data {
		images[i] = &GeneratedImage{
			URL:           data.URL,
			B64JSON:       data.B64JSON,
			RevisedPrompt: data.RevisedPrompt,
		}
	}

	return &ImageResponse{
		Images:    images,
		Model:     model.ID,
		CreatedAt: openAIResp.Created,
		Usage: &ImageUsage{
			TotalImages: len(images),
		},
	}, nil
}

// GenerateVideo is not supported by OpenAI image adapter.
func (a *OpenAIImageAdapter) GenerateVideo(ctx context.Context, req *VideoRequest, model *provider.Model, prov *provider.Provider) (*VideoResponse, error) {
	return nil, fmt.Errorf("video generation not supported by OpenAI image adapter")
}

// GetVideoStatus is not supported by OpenAI image adapter.
func (a *OpenAIImageAdapter) GetVideoStatus(ctx context.Context, taskID string, prov *provider.Provider) (*VideoStatus, error) {
	return nil, fmt.Errorf("video status not supported by OpenAI image adapter")
}

// HealthCheck performs a health check.
func (a *OpenAIImageAdapter) HealthCheck(ctx context.Context, prov *provider.Provider) error {
	url := prov.BaseURL + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+prov.APIKey)

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
