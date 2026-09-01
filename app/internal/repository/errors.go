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

	// ErrConversationNotFound also covers a conversation that exists but does
	// not belong to the caller: the two are answered identically so that ids
	// cannot be probed. See GetUserConversation.
	ErrConversationNotFound = errors.New("conversation not found")
	ErrRoomNameRequired     = errors.New("room name is required")

	ErrNotFound = errors.New("not found")
)
