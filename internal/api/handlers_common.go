package api

import (
	"net/http"
	"slices"

	"github.com/anthropics/antigravity-wrapper/internal/models"
	"github.com/gin-gonic/gin"
)

// healthHandler returns server health status.
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// modelsHandler returns available models.
// If the API key has allowed_models restrictions, only those models are returned.
func (s *Server) modelsHandler(c *gin.Context) {
	registry := models.GetGlobalRegistry()
	modelList := registry.ListModels()

	// Check if the API key has model restrictions
	var allowedModels []string
	apiKey := extractAPIKey(c)

	// Config-based API keys have no restrictions
	if !slices.Contains(s.cfg.APIKeys, apiKey) && s.keyStore != nil {
		if keyData := s.keyStore.Get(apiKey); keyData != nil {
			allowedModels = keyData.AllowedModels
		}
	}

	data := make([]gin.H, 0, len(modelList))
	for _, m := range modelList {
		// If allowed models is set, filter the list
		if len(allowedModels) > 0 && !slices.Contains(allowedModels, m.ID) {
			continue
		}

		data = append(data, gin.H{
			"id":       m.ID,
			"object":   m.Object,
			"created":  m.Created,
			"owned_by": m.OwnedBy,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}
