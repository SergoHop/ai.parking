package model

import "encoding/json"

// ParkingLotInput is the JSON body for creating or replacing a parking lot.
type ParkingLotInput struct {
	Name        string          `json:"name"`
	Address     string          `json:"address"`
	Kind        string          `json:"kind"`
	CenterLat   *float64        `json:"center_lat"`
	CenterLon   *float64        `json:"center_lon"`
	EntranceLat *float64        `json:"entrance_lat"`
	EntranceLon *float64        `json:"entrance_lon"`
	RadiusM     *int            `json:"radius_m"`
	GeoJSON     json.RawMessage `json:"geojson"`
}

// SpotInput is the JSON body for creating or replacing a parking spot.
type SpotInput struct {
	Code       string          `json:"code"`
	CameraID   *int64          `json:"camera_id"`
	Polygon    json.RawMessage `json:"polygon"`
	MapLat     *float64        `json:"map_lat"`
	MapLon     *float64        `json:"map_lon"`
	MapGeoJSON json.RawMessage `json:"map_geojson"`
}

// CameraInput is the JSON body for creating or replacing a camera.
type CameraInput struct {
	LotID      *int64  `json:"lot_id"`
	Name       string  `json:"name"`
	SourceType string  `json:"source_type"`
	Source     string  `json:"source"`
	Status     string  `json:"status"`
	FPS        *int    `json:"fps"`
	Resolution *string `json:"resolution"`
}

// DemoLot is the structure of data/lots/*.json used by cmd/seed.
type DemoLot struct {
	Name        string     `json:"name"`
	Address     string     `json:"address"`
	Kind        string     `json:"kind"`
	CenterLat   float64    `json:"center_lat"`
	CenterLon   float64    `json:"center_lon"`
	EntranceLat *float64   `json:"entrance_lat"`
	EntranceLon *float64   `json:"entrance_lon"`
	RadiusM     int        `json:"radius_m"`
	Camera      DemoCamera `json:"camera"`
	Spots       []DemoSpot `json:"spots"`
}

type DemoCamera struct {
	Name       string  `json:"name"`
	SourceType string  `json:"source_type"`
	Source     string  `json:"source"`
	FPS        *int    `json:"fps"`
	Resolution *string `json:"resolution"`
}

type DemoSpot struct {
	Code    string          `json:"code"`
	Polygon json.RawMessage `json:"polygon"`
	MapLat  *float64        `json:"map_lat"`
	MapLon  *float64        `json:"map_lon"`
}
