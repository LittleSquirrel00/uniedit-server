package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/module/ai/llm"
)

// ChatHandler handles chat API requests.
type ChatHandler struct {
	chatService      llm.ChatService
	embeddingService llm.EmbeddingService
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(llmService llm.LLMService) *ChatHandler {
	return &ChatHandler{
		chatService:      llmService,
		embeddingService: llmService,
	}
}

// RegisterRoutes registers chat routes.
func (h *ChatHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/chat", h.Chat)
	r.POST("/chat/stream", h.ChatStream)
	r.POST("/embeddings", h.Embeddings)
}

// Chat handles non-streaming chat requests.
//
//	@Summary		Send chat message
//	@Description	Send a chat completion request to the AI model (non-streaming)
//	@Tags			AI Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		llm.ChatRequest	true	"Chat request"
//	@Success		200		{object}	llm.ChatResponse
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		429		{object}	map[string]string	"Rate limit exceeded"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/ai/chat [post]
func (h *ChatHandler) Chat(c *gin.Context) {
	var req llm.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages required"})
		return
	}

	// Execute
	resp, err := h.chatService.Chat(c.Request.Context(), userID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ChatStream handles streaming chat requests.
//
//	@Summary		Send chat message (streaming)
//	@Description	Send a chat completion request with Server-Sent Events (SSE) streaming response
//	@Tags			AI Chat
//	@Accept			json
//	@Produce		text/event-stream
//	@Security		BearerAuth
//	@Param			request	body	llm.ChatRequest	true	"Chat request"
//	@Success		200		"SSE stream of chat chunks"
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		429		{object}	map[string]string	"Rate limit exceeded"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/ai/chat/stream [post]
func (h *ChatHandler) ChatStream(c *gin.Context) {
	var req llm.ChatRequest
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

	// Validate request
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages required"})
		return
	}

	// Force stream mode
	req.Stream = true

	// Start streaming
	chunks, routingInfo, err := h.chatService.ChatStream(c.Request.Context(), userID, &req)
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

// EmbeddingRequest represents an embedding API request.
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingObject represents a single embedding in the response.
type EmbeddingObject struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage represents token usage for embeddings.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingResponse represents an embedding API response.
type EmbeddingResponse struct {
	Object string             `json:"object"`
	Data   []*EmbeddingObject `json:"data"`
	Model  string             `json:"model"`
	Usage  *EmbeddingUsage    `json:"usage"`
}

// Embeddings handles embedding requests.
//
//	@Summary		Create embeddings
//	@Description	Create vector embeddings for the given input text
//	@Tags			AI Chat
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		EmbeddingRequest	true	"Embedding request"
//	@Success		200		{object}	EmbeddingResponse
//	@Failure		400		{object}	map[string]string	"Invalid request"
//	@Failure		401		{object}	map[string]string	"Unauthorized"
//	@Failure		429		{object}	map[string]string	"Rate limit exceeded"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/ai/embeddings [post]
func (h *ChatHandler) Embeddings(c *gin.Context) {
	var req EmbeddingRequest
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

	// Validate request
	if len(req.Input) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input required"})
		return
	}

	// Execute
	resp, err := h.embeddingService.Embed(c.Request.Context(), userID, &llm.EmbedRequest{
		Model: req.Model,
		Input: req.Input,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to OpenAI-compatible response format
	data := make([]*EmbeddingObject, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		data[i] = &EmbeddingObject{
			Object:    "embedding",
			Embedding: emb,
			Index:     i,
		}
	}

	var usage *EmbeddingUsage
	if resp.Usage != nil {
		usage = &EmbeddingUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	c.JSON(http.StatusOK, &EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  resp.Model,
		Usage:  usage,
	})
}

// SSE helper types
type sseEvent struct {
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data"`
}

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
