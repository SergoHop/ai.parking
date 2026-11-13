package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/service"
)

func (s *Server) listParkingLots(c *gin.Context) {
	lots, err := s.lots.ListParkingLots(c.Request.Context())
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, lots)
}

func (s *Server) getParkingLot(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	lot, err := s.lots.GetParkingLot(c.Request.Context(), id)
	if err != nil {
		writeStoreError(c, err)
		return
	}

	spots, err := s.spots.ListSpotsByLot(c.Request.Context(), id)
	if err != nil {
		writeStoreError(c, err)
		return
	}

	cameras, err := s.cameras.ListCamerasByLot(c.Request.Context(), id)
	if err != nil {
		writeStoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"lot":     lot,
		"spots":   spots,
		"cameras": cameras,
	})
}

func (s *Server) createParkingLot(c *gin.Context) {
	var in model.ParkingLotInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateParkingLot(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	lot, err := s.lots.CreateParkingLot(c.Request.Context(), parkingLotFromInput(in))
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusCreated, lot)
}

func (s *Server) updateParkingLot(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var in model.ParkingLotInput
	if !bindJSON(c, &in) {
		return
	}
	if err := service.ValidateParkingLot(in); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	lot := parkingLotFromInput(in)
	lot.ID = id
	updated, err := s.lots.UpdateParkingLot(c.Request.Context(), lot)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) deleteParkingLot(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := s.lots.DeleteParkingLot(c.Request.Context(), id); err != nil {
		writeStoreError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parkingLotFromInput(in model.ParkingLotInput) model.ParkingLot {
	radius := 100
	if in.RadiusM != nil {
		radius = *in.RadiusM
	}

	lot := model.ParkingLot{
		Name:        strings.TrimSpace(in.Name),
		Address:     strings.TrimSpace(in.Address),
		Kind:        in.Kind,
		CenterLat:   *in.CenterLat,
		CenterLon:   *in.CenterLon,
		EntranceLat: in.EntranceLat,
		EntranceLon: in.EntranceLon,
		RadiusM:     radius,
		GeoJSON:     in.GeoJSON,
	}
	return lot
}
