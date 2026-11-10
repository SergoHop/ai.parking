package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "degraded",
			"db":     "down",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"db":     "ok",
	})
}
