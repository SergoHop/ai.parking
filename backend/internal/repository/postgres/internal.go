package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

// UpdateCameraHeartbeat stores camera health reported by the CV worker.
// A nil lastFrameAt leaves the previous timestamp untouched.
func (d *DB) UpdateCameraHeartbeat(
	ctx context.Context,
	id int64,
	status string,
	lastFrameAt *time.Time,
	fps *int,
	resolution *string,
) error {
	const query = `
		UPDATE cameras
		SET status = $2,
		    last_frame_at = CASE
		        WHEN $3::timestamptz IS NULL THEN last_frame_at
		        ELSE GREATEST(COALESCE(last_frame_at, $3::timestamptz), $3::timestamptz)
		    END,
		    fps = COALESCE($4, fps),
		    resolution = COALESCE($5, resolution)
		WHERE id = $1`

	tag, err := d.pool.Exec(ctx, query, id, status, lastFrameAt, fps, resolution)
	if err != nil {
		return fmt.Errorf("update camera heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ApplySpotEvents writes occupancy history and updates current spot states in
// one transaction. It returns the accepted changes with lot and spot code, so
// callers can broadcast them over WebSocket after the commit.
func (d *DB) ApplySpotEvents(
	ctx context.Context,
	cameraID *int64,
	events []model.SpotEvent,
) ([]model.SpotChange, error) {
	if len(events) == 0 {
		return nil, nil
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin events transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	changes := make([]model.SpotChange, 0, len(events))

	for i := range events {
		event := &events[i]

		var eventID int64
		err := tx.QueryRow(ctx, `
			INSERT INTO spot_events
				(spot_id, camera_id, status, confidence, captured_at, snapshot_path)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,
			event.SpotID, cameraID, event.Status, event.Confidence,
			event.CapturedAt, event.SnapshotPath,
		).Scan(&eventID)
		if err != nil {
			return nil, mapEventError(err)
		}

		var change model.SpotChange
		err = tx.QueryRow(ctx, `
			WITH upserted AS (
				INSERT INTO spot_status
					(spot_id, status, confidence, updated_at, last_event_id)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (spot_id) DO UPDATE
				SET status = EXCLUDED.status,
				    confidence = EXCLUDED.confidence,
				    updated_at = EXCLUDED.updated_at,
				    last_event_id = EXCLUDED.last_event_id
				RETURNING spot_id, status, confidence, updated_at
			)
			SELECT u.spot_id, ps.lot_id, ps.code,
			       u.status, u.confidence, u.updated_at
			FROM upserted u
			JOIN parking_spots ps ON ps.id = u.spot_id`,
			event.SpotID, event.Status, event.Confidence, event.CapturedAt, eventID,
		).Scan(
			&change.SpotID, &change.LotID, &change.Code,
			&change.Status, &change.Confidence, &change.At,
		)
		if err != nil {
			return nil, mapEventError(err)
		}
		changes = append(changes, change)
	}

	if cameraID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE cameras
			SET last_frame_at = GREATEST(
			        COALESCE(last_frame_at, $2::timestamptz),
			        $2::timestamptz
			    )
			WHERE id = $1`,
			*cameraID, latestCapturedAt(events),
		); err != nil {
			return nil, mapEventError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit events transaction: %w", err)
	}
	return changes, nil
}

func latestCapturedAt(events []model.SpotEvent) time.Time {
	latest := events[0].CapturedAt
	for _, event := range events[1:] {
		if event.CapturedAt.After(latest) {
			latest = event.CapturedAt
		}
	}
	return latest
}

func mapEventError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return repository.ErrInvalidReference
	}
	return fmt.Errorf("apply spot event: %w", err)
}
