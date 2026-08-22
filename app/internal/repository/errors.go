package repository

import "errors"

var (
	// ErrUserAlreadyExists = errors.New("user already exists")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidUserID         = errors.New("invalid user ID")
)
