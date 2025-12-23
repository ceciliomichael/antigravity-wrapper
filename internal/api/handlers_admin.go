package api

import (
	"net/http"

	"github.com/anthropics/antigravity-wrapper/internal/models"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type generateKeyRequest struct {
	Note          string   `json:"note"`
	RateLimit     int      `json:"rate_limit"`     // RPM limit
	AllowedModels []string `json:"allowed_models"` // Models this key can access
}

type updateKeyRequest struct {
	Note          string   `json:"note"`
	RateLimit     int      `json:"rate_limit"`
	AllowedModels []string `json:"allowed_models"`
}

type generateKeyResponse struct {
	Key           string   `json:"key"`
	CreatedAt     string   `json:"created_at"`
	Note          string   `json:"note,omitempty"`
	RateLimit     int      `json:"rate_limit,omitempty"`
	AllowedModels []string `json:"allowed_models,omitempty"`
}

// generateKeyHandler handles the generation of new API keys.
func (s *Server) generateKeyHandler(c *gin.Context) {
	// Parse request
	var req generateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Generate key
	apiKey, err := s.keyStore.Generate(req.Note, req.RateLimit, req.AllowedModels)
	if err != nil {
		log.Errorf("Failed to generate API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to generate API key",
				"type":    "internal_error",
			},
		})
		return
	}

	log.Infof("Generated new API key with note: %s", req.Note)

	c.JSON(http.StatusCreated, generateKeyResponse{
		Key:           apiKey.Key,
		CreatedAt:     apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Note:          apiKey.Note,
		RateLimit:     apiKey.RateLimit,
		AllowedModels: apiKey.AllowedModels,
	})
}

// updateKeyHandler modifies an existing API key.
func (s *Server) updateKeyHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Key is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	var req updateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	apiKey, err := s.keyStore.Update(key, req.Note, req.RateLimit, req.AllowedModels)
	if err != nil {
		log.Warnf("Failed to update API key: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "Key not found or update failed",
				"type":    "not_found_error",
			},
		})
		return
	}

	log.Infof("Updated API key: %s", key)
	c.JSON(http.StatusOK, apiKey)
}

// revokeKeyHandler removes an API key.
func (s *Server) revokeKeyHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Key is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	if err := s.keyStore.Revoke(key); err != nil {
		log.Warnf("Failed to revoke API key: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "Key not found",
				"type":    "not_found_error",
			},
		})
		return
	}

	log.Infof("Revoked API key: %s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Key revoked successfully"})
}

// listKeysHandler returns all generated API keys.
func (s *Server) listKeysHandler(c *gin.Context) {
	keys := s.keyStore.List()
	c.JSON(http.StatusOK, gin.H{
		"data": keys,
	})
}

// listModelsHandler returns all available models for admin UI.
func (s *Server) listModelsHandler(c *gin.Context) {
	registry := models.GetGlobalRegistry()
	modelList := registry.ListModels()

	// Filter and format models for admin UI
	type adminModel struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}

	result := make([]adminModel, 0, len(modelList))
	for _, m := range modelList {
		// Skip internal models
		if models.ModelName2Alias(m.ID) == "" {
			continue
		}
		displayName := m.DisplayName
		if displayName == "" {
			displayName = m.ID
		}
		result = append(result, adminModel{
			ID:          m.ID,
			DisplayName: displayName,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}
