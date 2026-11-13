package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

const spotColumns = `
	ps.id, ps.lot_id, ps.camera_id, ps.code, ps.polygon,
	ps.map_lat, ps.map_lon, ps.map_geojson, ps.created_at,
	COALESCE(st.status, 'unknown'), st.confidence, st.updated_at`

func scanSpot(row rowScanner) (*model.ParkingSpot, error) {
	var (
		spot   model.ParkingSpot
		poly   []byte
		mapGeo []byte
	)
	if err := row.Scan(
		&spot.ID, &spot.LotID, &spot.CameraID, &spot.Code, &poly,
		&spot.MapLat, &spot.MapLon, &mapGeo, &spot.CreatedAt,
		&spot.Status, &spot.Confidence, &spot.StatusUpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(poly) > 0 {
		spot.Polygon = json.RawMessage(poly)
	}
	if len(mapGeo) > 0 {
		spot.MapGeoJSON = json.RawMessage(mapGeo)
	}
	return &spot, nil
}

func (d *DB) ListSpotsByLot(ctx context.Context, lotID int64) ([]model.ParkingSpot, error) {
	const query = `
		SELECT ` + spotColumns + `
		FROM parking_spots ps
		LEFT JOIN spot_status st ON st.spot_id = ps.id
		WHERE ps.lot_id = $1
		ORDER BY ps.code`

	rows, err := d.pool.Query(ctx, query, lotID)
	if err != nil {
		return nil, fmt.Errorf("query spots: %w", err)
	}
	defer rows.Close()

	spots := make([]model.ParkingSpot, 0)
	for rows.Next() {
		spot, err := scanSpot(rows)
		if err != nil {
			return nil, fmt.Errorf("scan spot: %w", err)
		}
		spots = append(spots, *spot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spots: %w", err)
	}
	return spots, nil
}

func (d *DB) GetSpot(ctx context.Context, id int64) (*model.ParkingSpot, error) {
	const query = `
		SELECT ` + spotColumns + `
		FROM parking_spots ps
		LEFT JOIN spot_status st ON st.spot_id = ps.id
		WHERE ps.id = $1`

	spot, err := scanSpot(d.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get spot: %w", err)
	}
	return spot, nil
}

func (d *DB) CreateSpot(ctx context.Context, spot model.ParkingSpot) (*model.ParkingSpot, error) {
	const query = `
		INSERT INTO parking_spots
			(lot_id, camera_id, code, polygon, map_lat, map_lon, map_geojson)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7::jsonb)
		RETURNING id`

	var id int64
	err := d.pool.QueryRow(ctx, query,
		spot.LotID, spot.CameraID, spot.Code, jsonArg(spot.Polygon),
		spot.MapLat, spot.MapLon, jsonArg(spot.MapGeoJSON),
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create spot: %w", err)
	}
	return d.GetSpot(ctx, id)
}

func (d *DB) UpdateSpot(ctx context.Context, spot model.ParkingSpot) (*model.ParkingSpot, error) {
	const query = `
		UPDATE parking_spots
		SET camera_id = $2, code = $3, polygon = $4::jsonb,
		    map_lat = $5, map_lon = $6, map_geojson = $7::jsonb
		WHERE id = $1`

	tag, err := d.pool.Exec(ctx, query,
		spot.ID, spot.CameraID, spot.Code, jsonArg(spot.Polygon),
		spot.MapLat, spot.MapLon, jsonArg(spot.MapGeoJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("update spot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, repository.ErrNotFound
	}
	return d.GetSpot(ctx, spot.ID)
}

func (d *DB) DeleteSpot(ctx context.Context, id int64) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM parking_spots WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete spot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}
