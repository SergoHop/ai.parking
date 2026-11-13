package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

func parseID(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Param(name))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid "+name)
		return 0, false
	}
	return id, true
}

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return false
	}
	return true
}

func writeStoreError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, repository.ErrConflict):
		writeError(c, http.StatusConflict, "conflict", "resource already exists")
	default:
		slog.Error("storage error", "error", err)
		writeError(c, http.StatusInternalServerError, "internal_error", "storage error")
	}
}
