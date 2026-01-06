package media

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProviderType(t *testing.T) {
	assert.Equal(t, ProviderType("openai"), ProviderTypeOpenAI)
	assert.Equal(t, ProviderType("anthropic"), ProviderTypeAnthropic)
	assert.Equal(t, ProviderType("generic"), ProviderTypeGeneric)
}

func TestCapability(t *testing.T) {
	assert.Equal(t, Capability("image"), CapabilityImage)
	assert.Equal(t, Capability("video"), CapabilityVideo)
	assert.Equal(t, Capability("audio"), CapabilityAudio)
}

func TestModel_HasCapability(t *testing.T) {
	model := &Model{
		ID:           "test-model",
		ProviderID:   uuid.New(),
		Name:         "Test Model",
		Capabilities: []Capability{CapabilityImage, CapabilityVideo},
		Enabled:      true,
	}

	assert.True(t, model.HasCapability(CapabilityImage))
	assert.True(t, model.HasCapability(CapabilityVideo))
	assert.False(t, model.HasCapability(CapabilityAudio))
}

func TestTaskStatus(t *testing.T) {
	assert.Equal(t, TaskStatus("pending"), TaskStatusPending)
	assert.Equal(t, TaskStatus("running"), TaskStatusRunning)
	assert.Equal(t, TaskStatus("completed"), TaskStatusCompleted)
	assert.Equal(t, TaskStatus("failed"), TaskStatusFailed)
	assert.Equal(t, TaskStatus("cancelled"), TaskStatusCancelled)
}

func TestVideoState(t *testing.T) {
	assert.Equal(t, VideoState("pending"), VideoStatePending)
	assert.Equal(t, VideoState("processing"), VideoStateProcessing)
	assert.Equal(t, VideoState("completed"), VideoStateCompleted)
	assert.Equal(t, VideoState("failed"), VideoStateFailed)
}

func TestTaskStatusToVideoState(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected VideoState
	}{
		{TaskStatusPending, VideoStatePending},
		{TaskStatusRunning, VideoStateProcessing},
		{TaskStatusCompleted, VideoStateCompleted},
		{TaskStatusFailed, VideoStateFailed},
		{TaskStatusCancelled, VideoStateFailed},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := taskStatusToVideoState(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}
