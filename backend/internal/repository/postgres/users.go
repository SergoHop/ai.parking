package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

func (d *DB) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	const query = `
		SELECT id, username, password_hash, role, created_at
		FROM users
		WHERE username = $1`

	var user model.User
	err := d.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &user, nil
}

func (d *DB) CreateUser(ctx context.Context, username, passwordHash, role string) (*model.User, error) {
	const query = `
		INSERT INTO users (username, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, username, password_hash, role, created_at`

	var user model.User
	err := d.pool.QueryRow(ctx, query, username, passwordHash, role).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &user, nil
}

func (d *DB) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := d.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}
