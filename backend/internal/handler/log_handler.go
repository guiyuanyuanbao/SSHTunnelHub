package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sshtunnelhub/internal/logger"
)

type LogHandler struct{}

func NewLogHandler() *LogHandler {
	return &LogHandler{}
}

func (h *LogHandler) List(c *gin.Context) {
	level := c.Query("level")
	tag := c.Query("tag")
	keyword := c.Query("keyword")

	var tunnelID uint
	if tidStr := c.Query("tunnel_id"); tidStr != "" {
		if tid, err := strconv.ParseUint(tidStr, 10, 64); err == nil {
			tunnelID = uint(tid)
		}
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs := logger.GetLogger().Query(level, tag, tunnelID, keyword, limit)
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

func (h *LogHandler) Clear(c *gin.Context) {
	logger.GetLogger().Clear()
	c.JSON(http.StatusOK, gin.H{"message": "logs cleared successfully"})
}
