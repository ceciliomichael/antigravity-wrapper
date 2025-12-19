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

// chatCompletionsHandler handles OpenAI Chat Completions requests.
func (s *Server) chatCompletionsHandler(c *gin.Context) {
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

	// Convert OpenAI request to Antigravity format
	payload := translator.ConvertOpenAIRequestToAntigravity(modelName, body, stream)

	// Apply thinking normalization
	payload = models.ApplyDefaultThinkingIfNeeded(modelName, payload)
	payload = models.StripThinkingConfigIfUnsupported(modelName, payload)

	// Get credentials for this request (round-robin if available)
	creds := s.getNextCredentials()

	if stream {
		s.handleStreamingOpenAI(c, modelName, payload, creds)
	} else {
		s.handleNonStreamingOpenAI(c, modelName, payload, creds)
	}
}

// responsesHandler handles OpenAI Responses API requests.
func (s *Server) responsesHandler(c *gin.Context) {
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

	// For Responses API, we use the OpenAI translator (simplified approach)
	payload := translator.ConvertOpenAIRequestToAntigravity(modelName, body, stream)

	// Apply thinking normalization
	payload = models.ApplyDefaultThinkingIfNeeded(modelName, payload)
	payload = models.StripThinkingConfigIfUnsupported(modelName, payload)

	// Get credentials for this request (round-robin if available)
	creds := s.getNextCredentials()

	if stream {
		s.handleStreamingResponses(c, modelName, payload, creds)
	} else {
		s.handleNonStreamingResponses(c, modelName, payload, creds)
	}
}

// handleStreamingOpenAI handles streaming OpenAI responses.
func (s *Server) handleStreamingOpenAI(c *gin.Context, modelName string, payload []byte, creds *auth.Credentials) {
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

	state := &translator.OpenAIStreamState{}

	for chunk := range streamChan {
		if chunk.Err != nil {
			log.Errorf("Stream chunk error: %v", chunk.Err)
			break
		}

		responses := translator.ConvertAntigravityResponseToOpenAI(modelName, chunk.Data, state, &translator.TranslatorOptions{
			ThinkingAsContent: s.cfg.ThinkingAsContent,
		})
		for _, resp := range responses {
			if resp != "" {
				c.Writer.WriteString("data: " + resp + "\n\n")
				c.Writer.Flush()
			}
		}
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()
}

// handleNonStreamingOpenAI handles non-streaming OpenAI responses.
func (s *Server) handleNonStreamingOpenAI(c *gin.Context, modelName string, payload []byte, creds *auth.Credentials) {
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

	converted := translator.ConvertAntigravityResponseToOpenAINonStream(modelName, resp.Body, &translator.TranslatorOptions{
		ThinkingAsContent: s.cfg.ThinkingAsContent,
	})
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, converted)
}

// handleStreamingResponses handles streaming Responses API.
func (s *Server) handleStreamingResponses(c *gin.Context, modelName string, payload []byte, creds *auth.Credentials) {
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

	state := &translator.OpenAIStreamState{}

	for chunk := range streamChan {
		if chunk.Err != nil {
			log.Errorf("Stream chunk error: %v", chunk.Err)
			break
		}

		responses := translator.ConvertAntigravityResponseToOpenAI(modelName, chunk.Data, state, &translator.TranslatorOptions{
			ThinkingAsContent: s.cfg.ThinkingAsContent,
		})
		for _, resp := range responses {
			if resp != "" {
				c.Writer.WriteString("data: " + resp + "\n\n")
				c.Writer.Flush()
			}
		}
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()
}

// handleNonStreamingResponses handles non-streaming Responses API.
func (s *Server) handleNonStreamingResponses(c *gin.Context, modelName string, payload []byte, creds *auth.Credentials) {
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

	converted := translator.ConvertAntigravityResponseToOpenAINonStream(modelName, resp.Body, &translator.TranslatorOptions{
		ThinkingAsContent: s.cfg.ThinkingAsContent,
	})
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, converted)
}
