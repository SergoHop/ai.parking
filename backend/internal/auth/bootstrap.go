package auth

import (
	"context"
	"errors"

	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository"
)

// UserStore is the subset of the user repository needed for admin bootstrap.
type UserStore interface {
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUser(ctx context.Context, username, passwordHash, role string) (*model.User, error)
	UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error
}

// EnsureAdmin creates the admin from env on first start and keeps its password
// in sync with ADMIN_PASSWORD. It returns true when a new user was created.
func EnsureAdmin(ctx context.Context, users UserStore, username, password string) (bool, error) {
	if username == "" || password == "" {
		return false, nil
	}

	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}

	user, err := users.GetByUsername(ctx, username)
	if errors.Is(err, repository.ErrNotFound) {
		if _, err := users.CreateUser(ctx, username, hash, "admin"); err != nil {
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}

	if !CheckPassword(user.PasswordHash, password) {
		if err := users.UpdateUserPassword(ctx, user.ID, hash); err != nil {
			return false, err
		}
	}
	return false, nil
}
