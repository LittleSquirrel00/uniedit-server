package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// ChatHandler implements inbound.AIChatHttpPort.
type ChatHandler struct {
	domain ai.AIDomain
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(domain ai.AIDomain) *ChatHandler {
	return &ChatHandler{domain: domain}
}

// Chat handles non-streaming chat requests.
func (h *ChatHandler) Chat(c *gin.Context) {
	var req model.AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	req.UserID = userID

	// Validate request
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages required"})
		return
	}

	// Execute
	resp, err := h.domain.Chat(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ChatStream handles streaming chat requests.
func (h *ChatHandler) ChatStream(c *gin.Context) {
	var req model.AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	req.UserID = userID

	// Validate request
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages required"})
		return
	}

	// Force stream mode
	req.Stream = true

	// Start streaming
	chunks, routingInfo, err := h.domain.ChatStream(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Send routing info as first event
	if routingInfo != nil {
		routingData, _ := json.Marshal(routingInfo)
		fmt.Fprintf(c.Writer, "event: routing\ndata: %s\n\n", routingData)
		c.Writer.Flush()
	}

	// Stream chunks
	for chunk := range chunks {
		data, err := json.Marshal(chunk)
		if err != nil {
			continue
		}

		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()

		// Check if client disconnected
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
	}

	// Send done event
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

// SSE helper types

// StreamWriter wraps a gin.ResponseWriter for SSE streaming.
type StreamWriter struct {
	w       gin.ResponseWriter
	flusher http.Flusher
}

// NewStreamWriter creates a new stream writer.
func NewStreamWriter(w gin.ResponseWriter) *StreamWriter {
	flusher, _ := w.(http.Flusher)
	return &StreamWriter{w: w, flusher: flusher}
}

// WriteEvent writes an SSE event.
func (sw *StreamWriter) WriteEvent(event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if event != "" {
		fmt.Fprintf(sw.w, "event: %s\n", event)
	}
	fmt.Fprintf(sw.w, "data: %s\n\n", jsonData)

	if sw.flusher != nil {
		sw.flusher.Flush()
	}
	return nil
}

// WriteData writes a data-only SSE event.
func (sw *StreamWriter) WriteData(data interface{}) error {
	return sw.WriteEvent("", data)
}

// WriteDone writes the done signal.
func (sw *StreamWriter) WriteDone() {
	fmt.Fprintf(sw.w, "data: [DONE]\n\n")
	if sw.flusher != nil {
		sw.flusher.Flush()
	}
}

// WriteError writes an error event.
func (sw *StreamWriter) WriteError(err error) {
	sw.WriteEvent("error", gin.H{"error": err.Error()})
}

// KeepAlive sends a keep-alive comment.
func (sw *StreamWriter) KeepAlive() {
	fmt.Fprintf(sw.w, ": keepalive\n\n")
	if sw.flusher != nil {
		sw.flusher.Flush()
	}
}

// StartKeepAlive starts a goroutine that sends keep-alive comments.
func (sw *StreamWriter) StartKeepAlive(interval time.Duration, stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sw.KeepAlive()
			case <-stop:
				return
			}
		}
	}()
}

// Compile-time interface check
var _ inbound.AIChatHttpPort = (*ChatHandler)(nil)
