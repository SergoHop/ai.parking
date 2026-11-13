package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/service"
)

func (s *Server) listSpots(c *gin.Context) {
	lotID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := s.lots.GetParkingLot(c.Request.Context(), lotID); err != nil {
		writeStoreError(c, err)
		return
	}

	spots, err := s.spots.ListSpotsByLot(c.Request.Context(), lotID)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, spots)
}

func (s *Server) createSpot(c *gin.Context) {
	lotID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := s.lots.GetParkingLot(c.Request.Context(), lotID); err != nil {
		writeStoreError(c, err)
		return
	}

	var in model.SpotInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateSpot(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if !s.ensureCameraBelongsToLot(c, in.CameraID, lotID) {
		return
	}

	spot := spotFromInput(in)
	spot.LotID = lotID
	created, err := s.spots.CreateSpot(c.Request.Context(), spot)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (s *Server) updateSpot(c *gin.Context) {
	spotID, ok := parseID(c, "id")
	if !ok {
		return
	}

	existing, err := s.spots.GetSpot(c.Request.Context(), spotID)
	if err != nil {
		writeStoreError(c, err)
		return
	}

	var in model.SpotInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateSpot(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if !s.ensureCameraBelongsToLot(c, in.CameraID, existing.LotID) {
		return
	}

	spot := spotFromInput(in)
	spot.ID = spotID
	spot.LotID = existing.LotID
	updated, err := s.spots.UpdateSpot(c.Request.Context(), spot)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) deleteSpot(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := s.spots.DeleteSpot(c.Request.Context(), id); err != nil {
		writeStoreError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) ensureCameraBelongsToLot(c *gin.Context, cameraID *int64, lotID int64) bool {
	if cameraID == nil {
		return true
	}
	camera, err := s.cameras.GetCamera(c.Request.Context(), *cameraID)
	if err != nil {
		writeStoreError(c, err)
		return false
	}
	if camera.LotID != lotID {
		writeError(c, http.StatusBadRequest, "validation_error", "camera belongs to another parking lot")
		return false
	}
	return true
}

func spotFromInput(in model.SpotInput) model.ParkingSpot {
	return model.ParkingSpot{
		Code:       strings.TrimSpace(in.Code),
		CameraID:   in.CameraID,
		Polygon:    in.Polygon,
		MapLat:     in.MapLat,
		MapLon:     in.MapLon,
		MapGeoJSON: in.MapGeoJSON,
	}
}
