package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/service"
)

func (s *Server) listParkingLots(c *gin.Context) {
	bbox, ok := parseBBox(c)
	if !ok {
		return
	}

	ttl := s.cfg.StatusTTL()
	lots, err := s.lots.ListParkingLots(c.Request.Context(), model.ParkingLotFilter{
		BBox: bbox,
		TTL:  ttl,
	})
	if err != nil {
		writeStoreError(c, err)
		return
	}

	if wantsSpots(c) {
		for i := range lots {
			spots, err := s.spots.ListSpotsByLot(c.Request.Context(), lots[i].ID, ttl)
			if err != nil {
				writeStoreError(c, err)
				return
			}
			lots[i].Spots = spots
		}
	}

	c.JSON(http.StatusOK, lots)
}

func (s *Server) getParkingLot(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	ttl := s.cfg.StatusTTL()
	lot, err := s.lots.GetParkingLot(c.Request.Context(), id, ttl)
	if err != nil {
		writeStoreError(c, err)
		return
	}

	spots, err := s.spots.ListSpotsByLot(c.Request.Context(), id, ttl)
	if err != nil {
		writeStoreError(c, err)
		return
	}

	cameras, err := s.cameras.ListCamerasByLot(c.Request.Context(), id)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	s.markCamerasFresh(cameras)

	c.JSON(http.StatusOK, gin.H{
		"lot":     lot,
		"spots":   spots,
		"cameras": cameras,
	})
}

// parseBBox reads an optional GeoJSON-order viewport: minLon,minLat,maxLon,maxLat.
func parseBBox(c *gin.Context) (*model.BBox, bool) {
	raw := strings.TrimSpace(c.Query("bbox"))
	if raw == "" {
		return nil, true
	}

	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		writeError(c, http.StatusBadRequest, "validation_error",
			"bbox must be minLon,minLat,maxLon,maxLat")
		return nil, false
	}

	values := make([]float64, 4)
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "bbox values must be numbers")
			return nil, false
		}
		values[i] = value
	}

	bbox := &model.BBox{
		MinLon: values[0],
		MinLat: values[1],
		MaxLon: values[2],
		MaxLat: values[3],
	}
	if bbox.MinLon < -180 || bbox.MaxLon > 180 ||
		bbox.MinLat < -90 || bbox.MaxLat > 90 ||
		bbox.MinLon >= bbox.MaxLon || bbox.MinLat >= bbox.MaxLat {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid bbox bounds")
		return nil, false
	}
	return bbox, true
}

func wantsSpots(c *gin.Context) bool {
	include := strings.ToLower(strings.TrimSpace(c.Query("include")))
	return include == "spots" || include == "all"
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
