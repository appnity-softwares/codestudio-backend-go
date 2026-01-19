package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/services"
)

type ExecuteRequest struct {
	Language string `json:"language" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

func ExecuteCode(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default limits for generic execution (or read from req if needed)
	result, err := services.ExecuteCode(req.Language, req.Code, "", 0, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Execution failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
