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

	// ErrAlreadyMember is the (conversation_id, user_id) primary key rejecting
	// a duplicate, so it means "already in", not "failed to add".
	ErrAlreadyMember = errors.New("user is already a member of this conversation")

	// ErrForbidden is a caller who *is* in the conversation but whose role is
	// too low for what they asked. It is distinct from ErrConversationNotFound
	// on purpose: a member already knows the conversation exists, so there is
	// nothing left to hide from them.
	ErrForbidden = errors.New("insufficient permissions for this conversation")

	// ErrNotARoom guards the operations that only make sense on a room. A DM's
	// membership is fixed at its two participants for life.
	ErrNotARoom = errors.New("this operation is only valid on a room")

	ErrNotFound  = errors.New("not found")
	ErrNotMember = errors.New("not a member of this conversation")
)
