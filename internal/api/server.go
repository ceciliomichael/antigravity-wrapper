// Package api provides the HTTP server and route handlers.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/anthropics/antigravity-wrapper/internal/auth"
	"github.com/anthropics/antigravity-wrapper/internal/config"
	"github.com/anthropics/antigravity-wrapper/internal/executor"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Server represents the HTTP API server.
type Server struct {
	cfg            *config.Config
	engine         *gin.Engine
	httpServer     *http.Server
	executor       *executor.Executor
	tokenManager   *auth.TokenManager
	store          *auth.Store
	credentials    *auth.Credentials
	accountManager *auth.AccountManager
}

// NewServer creates a new API server instance.
func NewServer(cfg *config.Config) (*Server, error) {
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(corsMiddleware())
	engine.Use(requestLogger())

	store := auth.NewStore(cfg.CredentialsDir)
	tokenManager := auth.NewTokenManager(store, executor.NewHTTPClient(cfg.ProxyURL, 30*time.Second))
	exec := executor.NewExecutor(cfg.ProxyURL, tokenManager)

	s := &Server{
		cfg:          cfg,
		engine:       engine,
		executor:     exec,
		tokenManager: tokenManager,
		store:        store,
	}

	// Try to load AccountManager for round-robin (priority)
	if accountManager := auth.LoadAccountManager(tokenManager); accountManager != nil {
		s.accountManager = accountManager
		log.Infof("Round-robin mode enabled with %d accounts", accountManager.Count())
	} else {
		// Fall back to single credential mode
		creds, filename, err := store.LoadFirst()
		if err != nil {
			log.Warnf("No credentials found: %v", err)
			log.Info("Run 'antigravity-wrapper login' to authenticate")
		} else {
			s.credentials = creds
			log.Infof("Loaded credentials from %s", filename)
		}
	}

	s.setupRoutes()

	return s, nil
}

// getNextCredentials returns the next credentials to use for a request.
// If AccountManager is available, uses round-robin selection.
// Otherwise, returns the single stored credentials.
func (s *Server) getNextCredentials() *auth.Credentials {
	if s.accountManager != nil {
		creds, err := s.accountManager.Next()
		if err != nil {
			log.Errorf("Failed to get next account: %v", err)
			return s.credentials // Fall back to single credentials if available
		}
		return creds
	}
	return s.credentials
}

// hasCredentials returns true if any credentials are available.
func (s *Server) hasCredentials() bool {
	return s.accountManager != nil || s.credentials != nil
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// API key authentication middleware
	apiAuth := s.apiKeyAuth()

	// Health check
	s.engine.GET("/health", s.healthHandler)

	// OpenAI-compatible endpoints
	v1 := s.engine.Group("/v1")
	v1.Use(apiAuth)
	{
		v1.GET("/models", s.modelsHandler)
		v1.POST("/chat/completions", s.chatCompletionsHandler)
		v1.POST("/responses", s.responsesHandler)
	}

	// Claude/Anthropic-compatible endpoint
	s.engine.POST("/v1/messages", apiAuth, s.messagesHandler)
}

// apiKeyAuth returns middleware that validates API keys if configured.
func (s *Server) apiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth if no API keys configured
		if len(s.cfg.APIKeys) == 0 {
			c.Next()
			return
		}

		// Extract API key from Authorization header
		authHeader := c.GetHeader("Authorization")
		apiKey := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			apiKey = authHeader[7:]
		}

		// Also check x-api-key header
		if apiKey == "" {
			apiKey = c.GetHeader("x-api-key")
		}

		// Validate API key
		valid := false
		for _, key := range s.cfg.APIKeys {
			if key == apiKey {
				valid = true
				break
			}
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

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	log.Infof("Starting server on %s", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

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
