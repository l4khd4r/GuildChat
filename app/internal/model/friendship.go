package model

import "time"

type FriendshipStatus string

const (
	FriendshipPending  FriendshipStatus = "pending"
	FriendshipAccepted FriendshipStatus = "accepted"
	FriendshipRejected FriendshipStatus = "rejected"
)

type Friendship struct {
	ID          int64            `json:"id"`
	RequesterID int64            `json:"requester_id"`
	ReceiverID  int64            `json:"receiver_id"`
	Status      FriendshipStatus `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
