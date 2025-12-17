package api

import (
	"net/http"

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
func (s *Server) modelsHandler(c *gin.Context) {
	registry := models.GetGlobalRegistry()
	modelList := registry.ListModels()

	data := make([]gin.H, 0, len(modelList))
	for _, m := range modelList {
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