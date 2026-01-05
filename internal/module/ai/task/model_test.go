package task

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
	t.Run("Constants are defined correctly", func(t *testing.T) {
		assert.Equal(t, Status("pending"), StatusPending)
		assert.Equal(t, Status("running"), StatusRunning)
		assert.Equal(t, Status("completed"), StatusCompleted)
		assert.Equal(t, Status("failed"), StatusFailed)
		assert.Equal(t, Status("cancelled"), StatusCancelled)
	})
}

func TestType(t *testing.T) {
	t.Run("Constants are defined correctly", func(t *testing.T) {
		assert.Equal(t, Type("chat"), TypeChat)
		assert.Equal(t, Type("image_generation"), TypeImageGeneration)
		assert.Equal(t, Type("video_generation"), TypeVideoGeneration)
		assert.Equal(t, Type("audio_generation"), TypeAudioGeneration)
		assert.Equal(t, Type("embedding"), TypeEmbedding)
	})
}

func TestTaskTableName(t *testing.T) {
	task := Task{}
	assert.Equal(t, "ai_tasks", task.TableName())
}

func TestTask_IsTerminal(t *testing.T) {
	t.Run("Completed is terminal", func(t *testing.T) {
		task := &Task{Status: StatusCompleted}
		assert.True(t, task.IsTerminal())
	})

	t.Run("Failed is terminal", func(t *testing.T) {
		task := &Task{Status: StatusFailed}
		assert.True(t, task.IsTerminal())
	})

	t.Run("Cancelled is terminal", func(t *testing.T) {
		task := &Task{Status: StatusCancelled}
		assert.True(t, task.IsTerminal())
	})

	t.Run("Pending is not terminal", func(t *testing.T) {
		task := &Task{Status: StatusPending}
		assert.False(t, task.IsTerminal())
	})

	t.Run("Running is not terminal", func(t *testing.T) {
		task := &Task{Status: StatusRunning}
		assert.False(t, task.IsTerminal())
	})
}

func TestTask_IsPending(t *testing.T) {
	t.Run("Pending status returns true", func(t *testing.T) {
		task := &Task{Status: StatusPending}
		assert.True(t, task.IsPending())
	})

	t.Run("Running status returns false", func(t *testing.T) {
		task := &Task{Status: StatusRunning}
		assert.False(t, task.IsPending())
	})

	t.Run("Completed status returns false", func(t *testing.T) {
		task := &Task{Status: StatusCompleted}
		assert.False(t, task.IsPending())
	})
}

func TestTask_IsRunning(t *testing.T) {
	t.Run("Running status returns true", func(t *testing.T) {
		task := &Task{Status: StatusRunning}
		assert.True(t, task.IsRunning())
	})

	t.Run("Pending status returns false", func(t *testing.T) {
		task := &Task{Status: StatusPending}
		assert.False(t, task.IsRunning())
	})

	t.Run("Completed status returns false", func(t *testing.T) {
		task := &Task{Status: StatusCompleted}
		assert.False(t, task.IsRunning())
	})
}

func TestError(t *testing.T) {
	err := &Error{
		Code:    "test_error",
		Message: "Test error message",
		Details: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test_error", err.Code)
	assert.Equal(t, "Test error message", err.Message)
	assert.NotNil(t, err.Details)
}

func TestFilter(t *testing.T) {
	userID := uuid.New()
	taskType := TypeChat
	status := StatusRunning

	filter := &Filter{
		UserID:   &userID,
		Type:     &taskType,
		Status:   &status,
		Limit:    10,
		Offset:   5,
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}

	assert.Equal(t, userID, *filter.UserID)
	assert.Equal(t, TypeChat, *filter.Type)
	assert.Equal(t, StatusRunning, *filter.Status)
	assert.Equal(t, 10, filter.Limit)
	assert.Equal(t, 5, filter.Offset)
	assert.Equal(t, "created_at", filter.OrderBy)
	assert.Equal(t, "DESC", filter.OrderDir)
}

func TestInput(t *testing.T) {
	input := &Input{
		Type: TypeChat,
		Payload: map[string]any{
			"messages": []string{"Hello"},
		},
		Priority: 1,
		Timeout:  30 * time.Second,
		Retry: &RetryConfig{
			MaxAttempts: 3,
			Delay:       time.Second,
		},
	}

	assert.Equal(t, TypeChat, input.Type)
	assert.NotNil(t, input.Payload)
	assert.Equal(t, 1, input.Priority)
	assert.Equal(t, 30*time.Second, input.Timeout)
	assert.NotNil(t, input.Retry)
	assert.Equal(t, 3, input.Retry.MaxAttempts)
	assert.Equal(t, time.Second, input.Retry.Delay)
}

func TestRetryConfig(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 5,
		Delay:       2 * time.Second,
	}

	assert.Equal(t, 5, config.MaxAttempts)
	assert.Equal(t, 2*time.Second, config.Delay)
}
