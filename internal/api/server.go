// Package api provides the HTTP server and route handlers.
package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
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
	keyStore       *auth.KeyStore
	limiters       sync.Map
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

	// Ensure data directory exists
	if err := cfg.EnsureDataDir(); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	// Initialize KeyStore
	var keyStore *auth.KeyStore
	if cfg.DataDir != "" {
		var err error
		keyStore, err = auth.NewKeyStore(cfg.DataDir)
		if err != nil {
			return nil, fmt.Errorf("initialize key store: %w", err)
		}
	}

	store := auth.NewStore(cfg.CredentialsDir)
	tokenManager := auth.NewTokenManager(store, executor.NewHTTPClient(cfg.ProxyURL, 30*time.Second))
	exec := executor.NewExecutor(cfg.ProxyURL, tokenManager)

	s := &Server{
		cfg:          cfg,
		engine:       engine,
		executor:     exec,
		tokenManager: tokenManager,
		store:        store,
		keyStore:     keyStore,
	}

	// Apply global middlewares
	engine.Use(corsMiddleware())
	engine.Use(requestLogger())
	if cfg.RateLimit > 0 {
		engine.Use(s.rateLimitMiddleware())
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

	// Admin endpoints
	s.engine.POST("/admin/keys", s.generateKeyHandler)

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
