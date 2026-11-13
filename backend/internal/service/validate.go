package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SergoHop/ai.parking/backend/internal/model"
)

// ValidateParkingLot checks a create/replace payload for a parking lot.
func ValidateParkingLot(in model.ParkingLotInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if in.Kind != "lot" && in.Kind != "roadside" {
		return fmt.Errorf("kind must be 'lot' or 'roadside'")
	}
	if in.CenterLat == nil || in.CenterLon == nil {
		return fmt.Errorf("center_lat and center_lon are required")
	}
	if err := validateLatLon(*in.CenterLat, *in.CenterLon, "center"); err != nil {
		return err
	}
	if (in.EntranceLat == nil) != (in.EntranceLon == nil) {
		return fmt.Errorf("entrance_lat and entrance_lon must be set together")
	}
	if in.EntranceLat != nil {
		if err := validateLatLon(*in.EntranceLat, *in.EntranceLon, "entrance"); err != nil {
			return err
		}
	}
	if in.RadiusM != nil && *in.RadiusM <= 0 {
		return fmt.Errorf("radius_m must be positive")
	}
	if len(in.GeoJSON) > 0 && !json.Valid(in.GeoJSON) {
		return fmt.Errorf("geojson must be valid JSON")
	}
	return nil
}

// ValidateSpot checks a create/replace payload for a parking spot.
func ValidateSpot(in model.SpotInput) error {
	if strings.TrimSpace(in.Code) == "" {
		return fmt.Errorf("code is required")
	}
	if in.CameraID != nil && *in.CameraID <= 0 {
		return fmt.Errorf("camera_id must be positive")
	}
	if len(in.Polygon) == 0 {
		return fmt.Errorf("polygon is required")
	}
	var points [][]float64
	if err := json.Unmarshal(in.Polygon, &points); err != nil {
		return fmt.Errorf("polygon must be an array of [x, y] points")
	}
	if len(points) < 3 {
		return fmt.Errorf("polygon must contain at least 3 points")
	}
	for i, p := range points {
		if len(p) != 2 {
			return fmt.Errorf("polygon point %d must have exactly 2 coordinates", i)
		}
		if p[0] < 0 || p[0] > 1 || p[1] < 0 || p[1] > 1 {
			return fmt.Errorf("polygon point %d must be normalized to 0..1", i)
		}
	}
	if (in.MapLat == nil) != (in.MapLon == nil) {
		return fmt.Errorf("map_lat and map_lon must be set together")
	}
	if in.MapLat != nil {
		if err := validateLatLon(*in.MapLat, *in.MapLon, "map"); err != nil {
			return err
		}
	}
	if len(in.MapGeoJSON) > 0 && !json.Valid(in.MapGeoJSON) {
		return fmt.Errorf("map_geojson must be valid JSON")
	}
	return nil
}

// ValidateCamera checks a create/replace payload for a camera.
func ValidateCamera(in model.CameraInput) error {
	if in.LotID == nil || *in.LotID <= 0 {
		return fmt.Errorf("lot_id is required and must be positive")
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch in.SourceType {
	case "rtsp", "webcam", "file":
	default:
		return fmt.Errorf("source_type must be 'rtsp', 'webcam' or 'file'")
	}
	if strings.TrimSpace(in.Source) == "" {
		return fmt.Errorf("source is required")
	}
	if in.FPS != nil && *in.FPS <= 0 {
		return fmt.Errorf("fps must be positive")
	}
	switch in.Status {
	case "", "online", "offline", "error":
	default:
		return fmt.Errorf("status must be 'online', 'offline' or 'error'")
	}
	return nil
}

// ValidateDemoLot checks a seed file before it reaches the database.
func ValidateDemoLot(d model.DemoLot) error {
	lot := model.ParkingLotInput{
		Name:        d.Name,
		Address:     d.Address,
		Kind:        d.Kind,
		CenterLat:   &d.CenterLat,
		CenterLon:   &d.CenterLon,
		EntranceLat: d.EntranceLat,
		EntranceLon: d.EntranceLon,
		RadiusM:     &d.RadiusM,
	}
	if err := ValidateParkingLot(lot); err != nil {
		return err
	}
	if strings.TrimSpace(d.Camera.Name) == "" {
		return fmt.Errorf("camera.name is required")
	}
	if d.Camera.SourceType != "rtsp" && d.Camera.SourceType != "webcam" && d.Camera.SourceType != "file" {
		return fmt.Errorf("camera.source_type is invalid")
	}
	if strings.TrimSpace(d.Camera.Source) == "" {
		return fmt.Errorf("camera.source is required")
	}
	if len(d.Spots) == 0 {
		return fmt.Errorf("spots must not be empty")
	}

	seen := make(map[string]bool, len(d.Spots))
	for _, s := range d.Spots {
		spot := model.SpotInput{
			Code:    s.Code,
			Polygon: s.Polygon,
			MapLat:  s.MapLat,
			MapLon:  s.MapLon,
		}
		if err := ValidateSpot(spot); err != nil {
			return fmt.Errorf("spot %q: %w", s.Code, err)
		}
		if seen[s.Code] {
			return fmt.Errorf("duplicate spot code %q", s.Code)
		}
		seen[s.Code] = true
	}
	return nil
}

func validateLatLon(lat, lon float64, prefix string) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("%s_lat must be between -90 and 90", prefix)
	}
	if lon < -180 || lon > 180 {
		return fmt.Errorf("%s_lon must be between -180 and 180", prefix)
	}
	return nil
}
