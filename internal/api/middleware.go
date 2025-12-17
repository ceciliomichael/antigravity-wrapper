package api

import (
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
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
		if limit <= 0 {
			c.Next()
			return
		}

		key := extractAPIKey(c)
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
