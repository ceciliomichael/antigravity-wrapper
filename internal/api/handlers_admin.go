package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type generateKeyRequest struct {
	Note string `json:"note"`
}

type generateKeyResponse struct {
	Key       string `json:"key"`
	CreatedAt string `json:"created_at"`
	Note      string `json:"note,omitempty"`
}

// generateKeyHandler handles the generation of new API keys.
// Requires Authorization: Bearer <MasterSecret>
func (s *Server) generateKeyHandler(c *gin.Context) {
	// Check if master secret is configured
	if s.cfg.MasterSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "Master secret not configured",
				"type":    "configuration_error",
			},
		})
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
		return
	}

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
	apiKey, err := s.keyStore.Generate(req.Note)
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
		Key:       apiKey.Key,
		CreatedAt: apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Note:      apiKey.Note,
	})
}
