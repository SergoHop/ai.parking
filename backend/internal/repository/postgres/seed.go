package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/SergoHop/ai.parking/backend/internal/model"
)

// SeedDemoLot inserts (or rebuilds) a demo parking lot with one camera and spots.
// If a lot with the same name exists and force is false, nothing is changed.
func (d *DB) SeedDemoLot(ctx context.Context, demo model.DemoLot, force bool) (int64, bool, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var existingID int64
	err = tx.QueryRow(ctx, `SELECT id FROM parking_lots WHERE name = $1 ORDER BY id LIMIT 1`, demo.Name).Scan(&existingID)
	switch {
	case err == nil:
		if !force {
			return existingID, false, nil
		}
		if _, err := tx.Exec(ctx, `DELETE FROM parking_lots WHERE id = $1`, existingID); err != nil {
			return 0, false, fmt.Errorf("delete existing demo lot: %w", err)
		}
	case errors.Is(err, pgx.ErrNoRows):
		existingID = 0
	default:
		return 0, false, fmt.Errorf("find demo lot: %w", err)
	}

	radius := demo.RadiusM
	if radius <= 0 {
		radius = 100
	}

	var lotID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO parking_lots
			(name, address, kind, center_lat, center_lon,
			 entrance_lat, entrance_lon, radius_m)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		demo.Name, demo.Address, demo.Kind, demo.CenterLat, demo.CenterLon,
		demo.EntranceLat, demo.EntranceLon, radius,
	).Scan(&lotID)
	if err != nil {
		return 0, false, fmt.Errorf("insert demo lot: %w", err)
	}

	cameraStatus := "offline"
	var cameraID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO cameras (lot_id, name, source_type, source, status, fps, resolution)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		lotID, demo.Camera.Name, demo.Camera.SourceType, demo.Camera.Source,
		cameraStatus, demo.Camera.FPS, demo.Camera.Resolution,
	).Scan(&cameraID)
	if err != nil {
		return 0, false, fmt.Errorf("insert demo camera: %w", err)
	}

	for _, spot := range demo.Spots {
		if _, err := tx.Exec(ctx, `
			INSERT INTO parking_spots
				(lot_id, camera_id, code, polygon, map_lat, map_lon)
			VALUES ($1, $2, $3, $4::jsonb, $5, $6)`,
			lotID, cameraID, spot.Code, jsonArg(spot.Polygon), spot.MapLat, spot.MapLon,
		); err != nil {
			return 0, false, fmt.Errorf("insert demo spot %s: %w", spot.Code, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, false, fmt.Errorf("commit seed transaction: %w", err)
	}
	return lotID, true, nil
}
