package repository

import "errors"

var (
	// ErrUserAlreadyExists = errors.New("user already exists")
	ErrUsernameAlreadyExists   = errors.New("username already exists")
	ErrEmailAlreadyExists      = errors.New("email already exists")
	ErrUserNotFound            = errors.New("user not found")
	ErrInvalidUserID           = errors.New("invalid user ID")
	ErrCannotFriendYourself    = errors.New("cannot send friend request to yourself")
	ErrFriendshipNotFound      = errors.New("friendship not found")
	ErrFriendshipAlreadyExists = errors.New("friendship already exists")

	ErrNotFound = errors.New("not found")
)
