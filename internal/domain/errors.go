package domain

import "errors"

var (
	ErrNotFound           = errors.New("record not found")
	ErrAlreadyExists      = errors.New("record already exists")
	ErrInvalidInput       = errors.New("invalid input data")
	ErrInternalServer     = errors.New("internal server error")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
)
