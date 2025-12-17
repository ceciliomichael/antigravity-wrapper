package api

import (
	"io"
	"net/http"

	"github.com/anthropics/antigravity-wrapper/internal/auth"
	"github.com/anthropics/antigravity-wrapper/internal/executor"
	"github.com/anthropics/antigravity-wrapper/internal/models"
	"github.com/anthropics/antigravity-wrapper/internal/translator"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// messagesHandler handles Claude/Anthropic Messages API requests.
func (s *Server) messagesHandler(c *gin.Context) {
	if !s.hasCredentials() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "No credentials configured. Run 'antigravity-wrapper login' to authenticate.",
				"type":    "authentication_error",
			},
		})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Failed to read request body",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Extract model and stream flag
	modelName := gjson.GetBytes(body, "model").String()
	if modelName == "" {
		modelName = "gemini-3-flash"
	}
	stream := gjson.GetBytes(body, "stream").Bool()

	// Convert Claude request to Antigravity format
	payload := translator.ConvertClaudeRequestToAntigravity(modelName, body, stream)

	// Apply thinking normalization
	payload = models.ApplyDefaultThinkingIfNeeded(modelName, payload)
	payload = models.StripThinkingConfigIfUnsupported(modelName, payload)

	// Get credentials for this request (round-robin if available)
	creds := s.getNextCredentials()

	if stream {
		s.handleStreamingClaude(c, modelName, payload, creds)
	} else {
		s.handleNonStreamingClaude(c, modelName, payload, creds)
	}
}

// handleStreamingClaude handles streaming Claude responses.
func (s *Server) handleStreamingClaude(c *gin.Context, modelName string, payload []byte, creds *auth.Credentials) {
	streamChan, err := s.executor.ExecuteStream(c.Request.Context(), creds, executor.Request{
		Model:   modelName,
		Payload: payload,
		Stream:  true,
	})
	if err != nil {
		log.Errorf("Streaming request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "api_error",
			},
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	state := &translator.ClaudeStreamState{}

	for chunk := range streamChan {
		if chunk.Err != nil {
			log.Errorf("Stream chunk error: %v", chunk.Err)
			break
		}

		responses := translator.ConvertAntigravityResponseToClaude(modelName, chunk.Data, state)
		for _, resp := range responses {
			if resp != "" {
				c.Writer.WriteString(resp)
				c.Writer.Flush()
			}
		}
	}

	// Send final [DONE] through translator
	finalResponses := translator.ConvertAntigravityResponseToClaude(modelName, []byte("[DONE]"), state)
	for _, resp := range finalResponses {
		if resp != "" {
			c.Writer.WriteString(resp)
			c.Writer.Flush()
		}
	}
}

// handleNonStreamingClaude handles non-streaming Claude responses.
func (s *Server) handleNonStreamingClaude(c *gin.Context, modelName string, payload []byte, creds *auth.Credentials) {
	resp, err := s.executor.Execute(c.Request.Context(), creds, executor.Request{
		Model:   modelName,
		Payload: payload,
		Stream:  false,
	})
	if err != nil {
		log.Errorf("Non-streaming request failed: %v", err)
		statusCode := http.StatusInternalServerError
		if resp != nil {
			statusCode = resp.StatusCode
		}
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "api_error",
			},
		})
		return
	}

	converted := translator.ConvertAntigravityResponseToClaudeNonStream(modelName, resp.Body)
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, converted)
}