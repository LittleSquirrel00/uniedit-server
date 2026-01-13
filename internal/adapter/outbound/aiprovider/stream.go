package aiprovider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/uniedit/server/internal/model"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// SSEParser parses SSE streams.
type SSEParser struct {
	reader *bufio.Reader
}

// NewSSEParser creates a new SSE parser.
func NewSSEParser(r io.Reader) *SSEParser {
	return &SSEParser{
		reader: bufio.NewReader(r),
	}
}

// Next reads the next SSE event.
func (p *SSEParser) Next() (*SSEEvent, error) {
	event := &SSEEvent{}
	var dataLines []string

	for {
		line, err := p.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(dataLines) > 0 {
				event.Data = strings.Join(dataLines, "\n")
				return event, nil
			}
			return nil, err
		}

		line = bytes.TrimRight(line, "\r\n")

		// Empty line signals end of event
		if len(line) == 0 {
			if len(dataLines) > 0 || event.Event != "" {
				event.Data = strings.Join(dataLines, "\n")
				return event, nil
			}
			continue
		}

		// Comment line
		if bytes.HasPrefix(line, []byte(":")) {
			continue
		}

		// Parse field
		field, value := parseSSELine(line)
		switch field {
		case "event":
			event.Event = value
		case "data":
			dataLines = append(dataLines, value)
		case "id":
			event.ID = value
		case "retry":
			// Parse retry value if needed
		}
	}
}

// parseSSELine parses a single SSE line into field and value.
func parseSSELine(line []byte) (field, value string) {
	if idx := bytes.IndexByte(line, ':'); idx >= 0 {
		field = string(line[:idx])
		value = string(bytes.TrimPrefix(line[idx+1:], []byte(" ")))
	} else {
		field = string(line)
	}
	return
}

// OpenAIStreamChunk represents an OpenAI streaming chunk.
type OpenAIStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string              `json:"role,omitempty"`
			Content   string              `json:"content,omitempty"`
			ToolCalls []*model.AIToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *model.AIUsage `json:"usage,omitempty"`
}

// ParseOpenAIChunk parses an OpenAI streaming chunk.
func ParseOpenAIChunk(data string) (*model.AIChatChunk, error) {
	if data == "[DONE]" {
		return nil, io.EOF
	}

	var chunk OpenAIStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, fmt.Errorf("parse openai chunk: %w", err)
	}

	if len(chunk.Choices) == 0 {
		return &model.AIChatChunk{
			ID:    chunk.ID,
			Model: chunk.Model,
			Usage: chunk.Usage,
		}, nil
	}

	choice := chunk.Choices[0]
	return &model.AIChatChunk{
		ID:    chunk.ID,
		Model: chunk.Model,
		Delta: &model.AIDelta{
			Role:      choice.Delta.Role,
			Content:   choice.Delta.Content,
			ToolCalls: choice.Delta.ToolCalls,
		},
		FinishReason: choice.FinishReason,
		Usage:        chunk.Usage,
	}, nil
}

// AnthropicStreamEvent represents an Anthropic streaming event.
type AnthropicStreamEvent struct {
	Type  string          `json:"type"`
	Index int             `json:"index,omitempty"`
	Delta json.RawMessage `json:"delta,omitempty"`
	Usage *model.AIUsage  `json:"usage,omitempty"`
}

// AnthropicContentDelta represents an Anthropic content delta.
type AnthropicContentDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ParseAnthropicEvent parses an Anthropic streaming event.
func ParseAnthropicEvent(eventType, data string) (*model.AIChatChunk, error) {
	var event AnthropicStreamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, fmt.Errorf("parse anthropic event: %w", err)
	}

	switch event.Type {
	case "content_block_delta":
		var delta AnthropicContentDelta
		if err := json.Unmarshal(event.Delta, &delta); err != nil {
			return nil, fmt.Errorf("parse anthropic delta: %w", err)
		}
		return &model.AIChatChunk{
			Delta: &model.AIDelta{
				Content: delta.Text,
			},
			Usage: event.Usage,
		}, nil

	case "message_delta":
		// Message completed
		return &model.AIChatChunk{
			FinishReason: "stop",
			Usage:        event.Usage,
		}, nil

	case "message_stop":
		return nil, io.EOF

	default:
		// Skip other event types
		return nil, nil
	}
}
