package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/services"
)

const (
	// P0 FIX: Maximum code size: 64KB
	MaxCodeSizeBytes = 64 * 1024
	// P0 FIX: Maximum stdin size: 16KB
	MaxStdinSizeBytes = 16 * 1024
)

type ExecuteRequest struct {
	Language string `json:"language" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Stdin    string `json:"stdin"`
}

func ExecuteCode(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// P0 FIX: Enforce code size limits to prevent abuse
	if len(req.Code) > MaxCodeSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Code too large",
			"limit": "64KB maximum",
		})
		return
	}

	if len(req.Stdin) > MaxStdinSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Input too large",
			"limit": "16KB maximum",
		})
		return
	}

	// P0 FIX: Enforce timeout (2s) and memory limits (128MB) - don't use 0 defaults
	result, err := services.ExecuteCode(req.Language, req.Code, req.Stdin, 2.0, 128)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Execution failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
