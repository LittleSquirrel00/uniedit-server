package provider

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestProviderType(t *testing.T) {
	t.Run("Constants are defined correctly", func(t *testing.T) {
		assert.Equal(t, ProviderType("openai"), ProviderTypeOpenAI)
		assert.Equal(t, ProviderType("anthropic"), ProviderTypeAnthropic)
		assert.Equal(t, ProviderType("google"), ProviderTypeGoogle)
		assert.Equal(t, ProviderType("azure"), ProviderTypeAzure)
		assert.Equal(t, ProviderType("ollama"), ProviderTypeOllama)
		assert.Equal(t, ProviderType("generic"), ProviderTypeGeneric)
	})
}

func TestCapability(t *testing.T) {
	t.Run("Constants are defined correctly", func(t *testing.T) {
		assert.Equal(t, Capability("chat"), CapabilityChat)
		assert.Equal(t, Capability("stream"), CapabilityStream)
		assert.Equal(t, Capability("vision"), CapabilityVision)
		assert.Equal(t, Capability("tools"), CapabilityTools)
		assert.Equal(t, Capability("json_mode"), CapabilityJSON)
		assert.Equal(t, Capability("embedding"), CapabilityEmbedding)
		assert.Equal(t, Capability("image_generation"), CapabilityImage)
		assert.Equal(t, Capability("video_generation"), CapabilityVideo)
		assert.Equal(t, Capability("audio_generation"), CapabilityAudio)
	})
}

func TestProviderTableName(t *testing.T) {
	p := Provider{}
	assert.Equal(t, "ai_providers", p.TableName())
}

func TestModelTableName(t *testing.T) {
	m := Model{}
	assert.Equal(t, "ai_models", m.TableName())
}

func TestModel_HasCapability(t *testing.T) {
	model := &Model{
		ID:           "gpt-4o",
		ProviderID:   uuid.New(),
		Name:         "GPT-4o",
		Capabilities: pq.StringArray{"chat", "vision", "tools"},
	}

	t.Run("Returns true for existing capability", func(t *testing.T) {
		assert.True(t, model.HasCapability(CapabilityChat))
		assert.True(t, model.HasCapability(CapabilityVision))
		assert.True(t, model.HasCapability(CapabilityTools))
	})

	t.Run("Returns false for non-existing capability", func(t *testing.T) {
		assert.False(t, model.HasCapability(CapabilityEmbedding))
		assert.False(t, model.HasCapability(CapabilityImage))
	})
}

func TestModel_HasAllCapabilities(t *testing.T) {
	model := &Model{
		ID:           "gpt-4o",
		ProviderID:   uuid.New(),
		Name:         "GPT-4o",
		Capabilities: pq.StringArray{"chat", "vision", "tools"},
	}

	t.Run("Returns true when all capabilities exist", func(t *testing.T) {
		caps := []Capability{CapabilityChat, CapabilityVision}
		assert.True(t, model.HasAllCapabilities(caps))
	})

	t.Run("Returns true for empty capabilities", func(t *testing.T) {
		assert.True(t, model.HasAllCapabilities([]Capability{}))
	})

	t.Run("Returns false when any capability is missing", func(t *testing.T) {
		caps := []Capability{CapabilityChat, CapabilityEmbedding}
		assert.False(t, model.HasAllCapabilities(caps))
	})
}

func TestModel_AverageCostPer1K(t *testing.T) {
	t.Run("Calculates average correctly", func(t *testing.T) {
		model := &Model{
			InputCostPer1K:  0.01,
			OutputCostPer1K: 0.03,
		}
		assert.Equal(t, 0.02, model.AverageCostPer1K())
	})

	t.Run("Returns zero for zero costs", func(t *testing.T) {
		model := &Model{
			InputCostPer1K:  0,
			OutputCostPer1K: 0,
		}
		assert.Equal(t, float64(0), model.AverageCostPer1K())
	})
}
