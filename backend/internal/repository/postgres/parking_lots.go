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

type rowScanner interface {
	Scan(dest ...any) error
}

const parkingLotColumns = `
	pl.id, pl.name, pl.address, pl.kind, pl.center_lat, pl.center_lon,
	pl.entrance_lat, pl.entrance_lon, pl.radius_m, pl.geojson, pl.created_at`

const parkingLotColumnsBare = `
	id, name, address, kind, center_lat, center_lon,
	entrance_lat, entrance_lon, radius_m, geojson, created_at`

func scanParkingLot(row rowScanner) (*model.ParkingLot, error) {
	var (
		lot model.ParkingLot
		geo []byte
	)
	if err := row.Scan(
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
		return nil, err
	}
	if len(geo) > 0 {
		lot.GeoJSON = json.RawMessage(geo)
	}
	return &lot, nil
}

func (d *DB) ListParkingLots(ctx context.Context) ([]model.ParkingLot, error) {
	const query = `
		SELECT ` + parkingLotColumns + `,
		       COALESCE(s.total, 0), COALESCE(s.free, 0),
		       COALESCE(s.occupied, 0), COALESCE(s.unknown, 0),
		       c.last_frame_at
		FROM parking_lots pl
		LEFT JOIN LATERAL (
			SELECT count(*) AS total,
			       count(*) FILTER (WHERE st.status = 'free') AS free,
			       count(*) FILTER (WHERE st.status = 'occupied') AS occupied,
			       count(*) FILTER (WHERE st.status = 'unknown') AS unknown
			FROM parking_spots ps
			LEFT JOIN spot_status st ON st.spot_id = ps.id
			WHERE ps.lot_id = pl.id
		) s ON true
		LEFT JOIN LATERAL (
			SELECT max(last_frame_at) AS last_frame_at
			FROM cameras
			WHERE lot_id = pl.id
		) c ON true
		ORDER BY pl.id`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query parking lots: %w", err)
	}
	defer rows.Close()

	lots := make([]model.ParkingLot, 0)
	for rows.Next() {
		var (
			lot                        model.ParkingLot
			geo                        []byte
			total, free, occupied, unk int64
		)
		if err := rows.Scan(
			&lot.ID, &lot.Name, &lot.Address, &lot.Kind,
			&lot.CenterLat, &lot.CenterLon, &lot.EntranceLat, &lot.EntranceLon,
			&lot.RadiusM, &geo, &lot.CreatedAt,
			&total, &free, &occupied, &unk,
			&lot.LastFrameAt,
		); err != nil {
			return nil, fmt.Errorf("scan parking lot: %w", err)
		}
		if len(geo) > 0 {
			lot.GeoJSON = json.RawMessage(geo)
		}
		lot.TotalSpots = int(total)
		lot.FreeSpots = int(free)
		lot.OccupiedSpots = int(occupied)
		lot.UnknownSpots = int(unk)
		lots = append(lots, lot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate parking lots: %w", err)
	}
	return lots, nil
}

func (d *DB) GetParkingLot(ctx context.Context, id int64) (*model.ParkingLot, error) {
	const query = `
		SELECT ` + parkingLotColumns + `,
		       COALESCE(s.total, 0), COALESCE(s.free, 0),
		       COALESCE(s.occupied, 0), COALESCE(s.unknown, 0),
		       c.last_frame_at
		FROM parking_lots pl
		LEFT JOIN LATERAL (
			SELECT count(*) AS total,
			       count(*) FILTER (WHERE st.status = 'free') AS free,
			       count(*) FILTER (WHERE st.status = 'occupied') AS occupied,
			       count(*) FILTER (WHERE st.status = 'unknown') AS unknown
			FROM parking_spots ps
			LEFT JOIN spot_status st ON st.spot_id = ps.id
			WHERE ps.lot_id = pl.id
		) s ON true
		LEFT JOIN LATERAL (
			SELECT max(last_frame_at) AS last_frame_at
			FROM cameras
			WHERE lot_id = pl.id
		) c ON true
		WHERE pl.id = $1`

	var (
		lot                        model.ParkingLot
		geo                        []byte
		total, free, occupied, unk int64
	)
	err := d.pool.QueryRow(ctx, query, id).Scan(
		&lot.ID, &lot.Name, &lot.Address, &lot.Kind,
		&lot.CenterLat, &lot.CenterLon, &lot.EntranceLat, &lot.EntranceLon,
		&lot.RadiusM, &geo, &lot.CreatedAt,
		&total, &free, &occupied, &unk,
		&lot.LastFrameAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get parking lot: %w", err)
	}
	if len(geo) > 0 {
		lot.GeoJSON = json.RawMessage(geo)
	}
	lot.TotalSpots = int(total)
	lot.FreeSpots = int(free)
	lot.OccupiedSpots = int(occupied)
	lot.UnknownSpots = int(unk)
	return &lot, nil
}

func (d *DB) CreateParkingLot(ctx context.Context, lot model.ParkingLot) (*model.ParkingLot, error) {
	const query = `
		INSERT INTO parking_lots
			(name, address, kind, center_lat, center_lon,
			 entrance_lat, entrance_lon, radius_m, geojson)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
		RETURNING ` + parkingLotColumnsBare

	created, err := scanParkingLot(d.pool.QueryRow(ctx, query,
		lot.Name, lot.Address, lot.Kind, lot.CenterLat, lot.CenterLon,
		lot.EntranceLat, lot.EntranceLon, lot.RadiusM, jsonArg(lot.GeoJSON),
	))
	if err != nil {
		return nil, fmt.Errorf("create parking lot: %w", err)
	}
	return created, nil
}

func (d *DB) UpdateParkingLot(ctx context.Context, lot model.ParkingLot) (*model.ParkingLot, error) {
	const query = `
		UPDATE parking_lots
		SET name = $2, address = $3, kind = $4, center_lat = $5, center_lon = $6,
		    entrance_lat = $7, entrance_lon = $8, radius_m = $9, geojson = $10::jsonb
		WHERE id = $1
		RETURNING ` + parkingLotColumnsBare

	updated, err := scanParkingLot(d.pool.QueryRow(ctx, query,
		lot.ID, lot.Name, lot.Address, lot.Kind, lot.CenterLat, lot.CenterLon,
		lot.EntranceLat, lot.EntranceLon, lot.RadiusM, jsonArg(lot.GeoJSON),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update parking lot: %w", err)
	}
	return updated, nil
}

func (d *DB) DeleteParkingLot(ctx context.Context, id int64) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM parking_lots WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete parking lot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}
