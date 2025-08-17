package repository

import "errors"

// Common repository errors
var (
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource already exists")
	ErrInvalidInput = errors.New("invalid input")
	ErrForeignKey   = errors.New("foreign key constraint violation")
	ErrUnauthorized = errors.New("unauthorized access")
)
