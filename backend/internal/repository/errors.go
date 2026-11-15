package repository

import "errors"

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned on a unique constraint violation.
	ErrConflict = errors.New("conflict")
	// ErrInvalidReference is returned when a foreign key points to a missing row.
	ErrInvalidReference = errors.New("invalid reference")
)
