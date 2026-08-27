package dto

import "time"

// This file holds the response shapes for the friendship endpoints.
//
// Why these exist at all: the repository fills model.Friendship / model.User
// from the database, one field per column. Those are internal types. If a
// handler serialised them directly, the JSON contract would be pinned to the
// table layout, and every new column added later would be published to clients
// by accident. The DTOs below list their fields explicitly, so a response can
// only ever contain what is written here.
//
// Direction of the dependency: dto knows nothing (no imports beyond time), the
// handler maps model -> dto. See toFriendshipResponse and friends in
// internal/handler/mapper.go.

// FriendshipResponse is the friendship row itself, returned by the endpoints
// that create or change one:
//
//	POST /users/:id/friend-request
//	POST /friend-request/:id/accept
//	POST /friend-request/:id/reject
//
// ID is the friendship id, which is what the accept and reject routes take as
// their :id parameter. Status is one of "pending" / "accepted" / "rejected".
type FriendshipResponse struct {
	ID          int64     `json:"id"`
	RequesterID int64     `json:"requester_id"`
	ReceiverID  int64     `json:"receiver_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FriendRequestResponse is one entry in the pending / sent request lists:
//
//	GET /me/friend-requests       -> requests I received, User = who sent it
//	GET /me/friend-requests/sent  -> requests I sent,     User = who I sent it to
//
// ID is the friendship id, so the client can POST it straight to
// /friend-request/:id/accept. Without it the lists would be unusable, since a
// user id is not enough to answer a request. CreatedAt is when the request was
// sent, not when the user signed up (that one lives inside User).
type FriendRequestResponse struct {
	ID        int64        `json:"id"`
	User      UserResponse `json:"user"`
	Status    string       `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
}
