package http

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/service"
)

const internalTokenHeader = "X-Internal-Token"

// requireInternalToken protects the CV -> Go endpoints with a shared secret.
func (s *Server) requireInternalToken(c *gin.Context) {
	if s.cfg.InternalToken == "" {
		writeError(c, http.StatusServiceUnavailable, "internal_api_disabled", "internal API is not configured")
		c.Abort()
		return
	}

	got := c.GetHeader(internalTokenHeader)
	if subtle.ConstantTimeCompare([]byte(got), []byte(s.cfg.InternalToken)) != 1 {
		writeError(c, http.StatusUnauthorized, "unauthorized", "invalid internal token")
		c.Abort()
		return
	}

	c.Next()
}

func (s *Server) cameraHeartbeat(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var in model.CameraHeartbeatInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateCameraHeartbeat(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if err := s.internal.UpdateCameraHeartbeat(
		c.Request.Context(), id, in.Status, in.LastFrameAt, in.FPS, in.Resolution,
	); err != nil {
		writeStoreError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) spotEvents(c *gin.Context) {
	var in model.SpotEventsInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateSpotEvents(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if in.CameraID != nil {
		if _, err := s.internal.GetCamera(c.Request.Context(), *in.CameraID); err != nil {
			writeStoreError(c, err)
			return
		}
	}

	events := make([]model.SpotEvent, 0, len(in.Events))
	for _, e := range in.Events {
		events = append(events, model.SpotEvent{
			SpotID:       e.SpotID,
			CameraID:     in.CameraID,
			Status:       e.Status,
			Confidence:   e.Confidence,
			CapturedAt:   e.CapturedAt,
			SnapshotPath: e.SnapshotPath,
		})
	}

	accepted, err := s.internal.ApplySpotEvents(c.Request.Context(), in.CameraID, events)
	if err != nil {
		writeStoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"accepted": accepted})
}
