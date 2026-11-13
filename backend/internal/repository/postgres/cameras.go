package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

const cameraColumns = `
	id, lot_id, name, source_type, source, status,
	last_frame_at, fps, resolution, created_at`

func scanCamera(row rowScanner) (*model.Camera, error) {
	var cam model.Camera
	if err := row.Scan(
		&cam.ID, &cam.LotID, &cam.Name, &cam.SourceType, &cam.Source,
		&cam.Status, &cam.LastFrameAt, &cam.FPS, &cam.Resolution, &cam.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &cam, nil
}

func (d *DB) ListCamerasByLot(ctx context.Context, lotID int64) ([]model.Camera, error) {
	const query = `
		SELECT ` + cameraColumns + `
		FROM cameras
		WHERE lot_id = $1
		ORDER BY id`

	rows, err := d.pool.Query(ctx, query, lotID)
	if err != nil {
		return nil, fmt.Errorf("query cameras: %w", err)
	}
	defer rows.Close()

	cameras := make([]model.Camera, 0)
	for rows.Next() {
		cam, err := scanCamera(rows)
		if err != nil {
			return nil, fmt.Errorf("scan camera: %w", err)
		}
		cameras = append(cameras, *cam)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cameras: %w", err)
	}
	return cameras, nil
}

func (d *DB) GetCamera(ctx context.Context, id int64) (*model.Camera, error) {
	const query = `SELECT ` + cameraColumns + ` FROM cameras WHERE id = $1`

	cam, err := scanCamera(d.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get camera: %w", err)
	}
	return cam, nil
}

func (d *DB) CreateCamera(ctx context.Context, cam model.Camera) (*model.Camera, error) {
	const query = `
		INSERT INTO cameras
			(lot_id, name, source_type, source, status, fps, resolution)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	if err := d.pool.QueryRow(ctx, query,
		cam.LotID, cam.Name, cam.SourceType, cam.Source, cam.Status,
		cam.FPS, cam.Resolution,
	).Scan(&id); err != nil {
		return nil, fmt.Errorf("create camera: %w", err)
	}
	return d.GetCamera(ctx, id)
}

func (d *DB) UpdateCamera(ctx context.Context, cam model.Camera) (*model.Camera, error) {
	const query = `
		UPDATE cameras
		SET lot_id = $2, name = $3, source_type = $4, source = $5,
		    status = $6, fps = $7, resolution = $8
		WHERE id = $1`

	tag, err := d.pool.Exec(ctx, query,
		cam.ID, cam.LotID, cam.Name, cam.SourceType, cam.Source,
		cam.Status, cam.FPS, cam.Resolution,
	)
	if err != nil {
		return nil, fmt.Errorf("update camera: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, repository.ErrNotFound
	}
	return d.GetCamera(ctx, cam.ID)
}

func (d *DB) DeleteCamera(ctx context.Context, id int64) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM cameras WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete camera: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}
