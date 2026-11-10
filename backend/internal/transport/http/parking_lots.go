package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) listParkingLots(c *gin.Context) {
	lots, err := s.lots.ListParkingLots(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "failed to list parking lots")
		return
	}

	c.JSON(http.StatusOK, lots)
}
