package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
)

// corsMiddleware returns middleware that handles CORS (Cross-Origin Resource Sharing).
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Allow all origins
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// requestLogger returns middleware for logging requests.
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.WithFields(log.Fields{
			"status":  status,
			"method":  c.Request.Method,
			"path":    path,
			"latency": latency,
			"ip":      c.ClientIP(),
		}).Info("Request completed")
	}
}

// apiKeyAuth returns middleware that validates API keys if configured.
func (s *Server) apiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth if no API keys configured and no dynamic keystore active
		if len(s.cfg.APIKeys) == 0 && s.keyStore == nil {
			c.Next()
			return
		}

		apiKey := extractAPIKey(c)

		// Validate API key
		valid := slices.Contains(s.cfg.APIKeys, apiKey)

		if !valid && s.keyStore != nil {
			valid = s.keyStore.Validate(apiKey)
		}

		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key",
					"type":    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// rateLimitMiddleware returns middleware that performs rate limiting.
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := s.cfg.RateLimit
		key := extractAPIKey(c)

		// Check for per-key rate limit
		if key != "" && s.keyStore != nil {
			if apiKey := s.keyStore.Get(key); apiKey != nil && apiKey.RateLimit > 0 {
				limit = apiKey.RateLimit
			}
		}

		if limit <= 0 {
			c.Next()
			return
		}

		if key == "" {
			key = c.ClientIP()
		}

		// Get or create limiter for this key
		// limiters is a sync.Map storing *rate.Limiter
		val, _ := s.limiters.LoadOrStore(key, rate.NewLimiter(rate.Every(time.Minute/time.Duration(limit)), limit))
		limiter := val.(*rate.Limiter)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "Rate limit exceeded. Please try again later.",
					"type":    "rate_limit_error",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// modelAccessMiddleware returns middleware that validates model access for API keys.
func (s *Server) modelAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check POST requests with a body
		if c.Request.Method != "POST" {
			c.Next()
			return
		}

		apiKey := extractAPIKey(c)

		// Skip model check for config-based API keys (they have unrestricted access)
		if slices.Contains(s.cfg.APIKeys, apiKey) {
			c.Next()
			return
		}

		// Get the API key from keystore
		if s.keyStore == nil {
			c.Next()
			return
		}

		keyData := s.keyStore.Get(apiKey)
		if keyData == nil {
			c.Next()
			return
		}

		// If no allowed models configured, allow all
		if len(keyData.AllowedModels) == 0 {
			c.Next()
			return
		}

		// Read request body to extract model
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Failed to read request body",
					"type":    "invalid_request_error",
				},
			})
			c.Abort()
			return
		}

		// Restore the body for downstream handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Extract model from request
		model := gjson.GetBytes(body, "model").String()
		if model == "" {
			// No model specified, use default - allow it
			c.Next()
			return
		}

		// Check if model is in allowed list
		if !slices.Contains(keyData.AllowedModels, model) {
			log.Warnf("API key attempted to use restricted model: %s", model)
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"message": fmt.Sprintf("Model '%s' is not allowed for this API key", model),
					"type":    "permission_error",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractAPIKey extracts the API key from request headers.
func extractAPIKey(c *gin.Context) string {
	// Extract API key from Authorization header
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	// Check x-api-key header
	if apiKey := c.GetHeader("x-api-key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// masterSecretAuth returns middleware that validates the Master Secret.
func (s *Server) masterSecretAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if master secret is configured
		if s.cfg.MasterSecret == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"message": "Master secret not configured",
					"type":    "configuration_error",
				},
			})
			c.Abort()
			return
		}

		// Validate Master Secret
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Missing authorization header",
					"type":    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != s.cfg.MasterSecret {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid master secret",
					"type":    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
