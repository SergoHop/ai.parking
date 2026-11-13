package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/service"
)

func (s *Server) listCameras(c *gin.Context) {
	lotID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := s.lots.GetParkingLot(c.Request.Context(), lotID); err != nil {
		writeStoreError(c, err)
		return
	}

	cameras, err := s.cameras.ListCamerasByLot(c.Request.Context(), lotID)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, cameras)
}

func (s *Server) createCamera(c *gin.Context) {
	var in model.CameraInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateCamera(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if _, err := s.lots.GetParkingLot(c.Request.Context(), *in.LotID); err != nil {
		writeStoreError(c, err)
		return
	}

	camera, err := s.cameras.CreateCamera(c.Request.Context(), cameraFromInput(in))
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusCreated, camera)
}

func (s *Server) updateCamera(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var in model.CameraInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateCamera(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if _, err := s.lots.GetParkingLot(c.Request.Context(), *in.LotID); err != nil {
		writeStoreError(c, err)
		return
	}

	camera := cameraFromInput(in)
	camera.ID = id
	updated, err := s.cameras.UpdateCamera(c.Request.Context(), camera)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) deleteCamera(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := s.cameras.DeleteCamera(c.Request.Context(), id); err != nil {
		writeStoreError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func cameraFromInput(in model.CameraInput) model.Camera {
	status := in.Status
	if status == "" {
		status = "offline"
	}
	return model.Camera{
		LotID:      *in.LotID,
		Name:       strings.TrimSpace(in.Name),
		SourceType: in.SourceType,
		Source:     strings.TrimSpace(in.Source),
		Status:     status,
		FPS:        in.FPS,
		Resolution: in.Resolution,
	}
}
