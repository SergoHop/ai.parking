package postgres

import (
	"context"
	"fmt"

	"github.com/SergoHop/ai.parking/backend/internal/model"
)

func (d *DB) ListParkingLots(ctx context.Context) ([]model.ParkingLot, error) {
	const query = `
		SELECT id, name, address, kind, center_lat, center_lon,
		       entrance_lat, entrance_lon, radius_m, geojson, created_at
		FROM parking_lots
		ORDER BY id`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query parking lots: %w", err)
	}
	defer rows.Close()

	lots := make([]model.ParkingLot, 0)
	for rows.Next() {
		var (
			lot model.ParkingLot
			geo []byte
		)

		if err := rows.Scan(
			&lot.ID,
			&lot.Name,
			&lot.Address,
			&lot.Kind,
			&lot.CenterLat,
			&lot.CenterLon,
			&lot.EntranceLat,
			&lot.EntranceLon,
			&lot.RadiusM,
			&geo,
			&lot.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan parking lot: %w", err)
		}

		if len(geo) > 0 {
			lot.GeoJSON = geo
		}
		lots = append(lots, lot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate parking lots: %w", err)
	}

	return lots, nil
}
