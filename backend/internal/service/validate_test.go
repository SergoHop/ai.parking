package service

import (
	"encoding/json"
	"testing"

	"github.com/SergoHop/ai.parking/backend/internal/model"
)

func TestValidateSpotPolygon(t *testing.T) {
	valid := model.SpotInput{
		Code:    "A-01",
		Polygon: json.RawMessage(`[[0.1,0.2],[0.2,0.2],[0.2,0.3],[0.1,0.3]]`),
	}
	if err := ValidateSpot(valid); err != nil {
		t.Fatalf("valid polygon rejected: %v", err)
	}

	tooFew := valid
	tooFew.Polygon = json.RawMessage(`[[0.1,0.2],[0.2,0.2]]`)
	if err := ValidateSpot(tooFew); err == nil {
		t.Fatal("expected error for polygon with 2 points")
	}

	outOfRange := valid
	outOfRange.Polygon = json.RawMessage(`[[0.1,0.2],[1.2,0.2],[0.2,0.3]]`)
	if err := ValidateSpot(outOfRange); err == nil {
		t.Fatal("expected error for normalized coordinate above 1")
	}

	badShape := valid
	badShape.Polygon = json.RawMessage(`[[0.1,0.2,0.3],[0.2,0.2],[0.2,0.3]]`)
	if err := ValidateSpot(badShape); err == nil {
		t.Fatal("expected error for point with 3 coordinates")
	}
}

func TestValidateParkingLot(t *testing.T) {
	lat, lon := 55.75, 37.61
	radius := 100

	valid := model.ParkingLotInput{
		Name:      "Карман",
		Kind:      "roadside",
		CenterLat: &lat,
		CenterLon: &lon,
		RadiusM:   &radius,
	}
	if err := ValidateParkingLot(valid); err != nil {
		t.Fatalf("valid lot rejected: %v", err)
	}

	badKind := valid
	badKind.Kind = "garage"
	if err := ValidateParkingLot(badKind); err == nil {
		t.Fatal("expected error for unknown kind")
	}

	badLat := valid
	wrongLat := 120.0
	badLat.CenterLat = &wrongLat
	if err := ValidateParkingLot(badLat); err == nil {
		t.Fatal("expected error for latitude above 90")
	}

	halfEntrance := valid
	halfEntrance.EntranceLat = &lat
	if err := ValidateParkingLot(halfEntrance); err == nil {
		t.Fatal("expected error for entrance without longitude")
	}
}
