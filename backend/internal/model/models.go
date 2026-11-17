package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type ParkingLot struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Address     string          `json:"address"`
	Kind        string          `json:"kind"`
	CenterLat   float64         `json:"center_lat"`
	CenterLon   float64         `json:"center_lon"`
	EntranceLat *float64        `json:"entrance_lat,omitempty"`
	EntranceLon *float64        `json:"entrance_lon,omitempty"`
	RadiusM     int             `json:"radius_m"`
	GeoJSON     json.RawMessage `json:"geojson,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`

	// Агрегаты для публичного API, заполняются только при чтении.
	TotalSpots    int        `json:"total_spots"`
	FreeSpots     int        `json:"free_spots"`
	OccupiedSpots int        `json:"occupied_spots"`
	UnknownSpots  int        `json:"unknown_spots"`
	LastFrameAt   *time.Time `json:"last_frame_at,omitempty"`
	Fresh         bool       `json:"fresh"`

	// Spots заполняется только при include=spots в списке парковок.
	Spots []ParkingSpot `json:"spots,omitempty"`
}

type Camera struct {
	ID          int64      `json:"id"`
	LotID       int64      `json:"lot_id"`
	Name        string     `json:"name"`
	SourceType  string     `json:"source_type"`
	Source      string     `json:"source"`
	Status      string     `json:"status"`
	LastFrameAt *time.Time `json:"last_frame_at,omitempty"`
	FPS         *int       `json:"fps,omitempty"`
	Resolution  *string    `json:"resolution,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Fresh       bool       `json:"fresh"`
}

type ParkingSpot struct {
	ID         int64           `json:"id"`
	LotID      int64           `json:"lot_id"`
	CameraID   *int64          `json:"camera_id,omitempty"`
	Code       string          `json:"code"`
	Polygon    json.RawMessage `json:"polygon"`
	MapLat     *float64        `json:"map_lat,omitempty"`
	MapLon     *float64        `json:"map_lon,omitempty"`
	MapGeoJSON json.RawMessage `json:"map_geojson,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`

	// Текущий статус, заполняется при чтении через JOIN spot_status.
	// Status — эффективный статус: устаревшие данные превращаются в unknown.
	Status          string     `json:"status"`
	Confidence      *float32   `json:"confidence,omitempty"`
	StatusUpdatedAt *time.Time `json:"status_updated_at,omitempty"`
	Fresh           bool       `json:"fresh"`
}

type SpotStatus struct {
	SpotID      int64     `json:"spot_id"`
	Status      string    `json:"status"`
	Confidence  *float32  `json:"confidence,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastEventID *int64    `json:"last_event_id,omitempty"`
}

type SpotEvent struct {
	ID           int64     `json:"id"`
	SpotID       int64     `json:"spot_id"`
	CameraID     *int64    `json:"camera_id,omitempty"`
	Status       string    `json:"status"`
	Confidence   *float32  `json:"confidence,omitempty"`
	CapturedAt   time.Time `json:"captured_at"`
	SnapshotPath *string   `json:"snapshot_path,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// SpotChange is an accepted occupancy event enriched with lot and spot code,
// so the WebSocket layer can broadcast it without extra queries.
type SpotChange struct {
	SpotID     int64     `json:"spot_id"`
	LotID      int64     `json:"lot_id"`
	Code       string    `json:"code"`
	Status     string    `json:"status"`
	Confidence *float32  `json:"confidence,omitempty"`
	At         time.Time `json:"at"`
}

// BBox is a geographic viewport in GeoJSON order: minLon,minLat,maxLon,maxLat.
type BBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

// ParkingLotFilter narrows the public parking lot list.
type ParkingLotFilter struct {
	BBox *BBox
	TTL  time.Duration
}
