package handler

import (
	"github.com/l4khd4r/GuildChat/internal/dto"
	"github.com/l4khd4r/GuildChat/internal/model"
)

// The mappers below are the single crossing point between the internal domain
// types (model.*, filled by the repository from database rows) and the wire
// types (dto.*, which define the JSON contract).
//
// Keep the translation here and nowhere else. A handler that reaches for a
// model struct in c.JSON has skipped this layer, and whatever gets added to
// that model later ships to clients silently.

// toUserResponse converts a user to its public view. Note what it drops:
// model.User.PasswordHash never reaches the response, because this function
// does not copy it. The `json:"-"` tag on that field is a second seatbelt, not
// the mechanism.
func toUserResponse(user *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// toUserResponses maps a slice of users, preserving order.
//
// It returns an empty (non-nil) slice for an empty input on purpose: a nil
// slice marshals to JSON `null`, an empty one to `[]`. Clients iterating the
// result should not have to special-case null.
func toUserResponses(users []*model.User) []dto.UserResponse {
	responses := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, toUserResponse(user))
	}
	return responses
}

// toFriendshipResponse converts a friendship row to its wire form.
// The Status field is a model.FriendshipStatus (a string type) on the way in
// and a plain string on the way out, so the JSON contract does not depend on
// the domain enum.
func toFriendshipResponse(friendship *model.Friendship) dto.FriendshipResponse {
	return dto.FriendshipResponse{
		ID:          friendship.ID,
		RequesterID: friendship.RequesterID,
		ReceiverID:  friendship.ReceiverID,
		Status:      string(friendship.Status),
		CreatedAt:   friendship.CreatedAt,
		UpdatedAt:   friendship.UpdatedAt,
	}
}

// toFriendRequestResponse converts one entry of a pending / sent request list.
// The embedded user is whoever is on the other side of the request; which side
// that is depends on the query that produced it, not on this function.
func toFriendRequestResponse(request *model.FriendRequest) dto.FriendRequestResponse {
	return dto.FriendRequestResponse{
		ID:        request.ID,
		User:      toUserResponse(request.User),
		Status:    string(request.Status),
		CreatedAt: request.CreatedAt,
	}
}

// toFriendRequestResponses maps a slice of friend requests, preserving order.
// Returns an empty slice rather than nil, for the same reason as
// toUserResponses.
func toFriendRequestResponses(requests []*model.FriendRequest) []dto.FriendRequestResponse {
	responses := make([]dto.FriendRequestResponse, 0, len(requests))
	for _, request := range requests {
		responses = append(responses, toFriendRequestResponse(request))
	}
	return responses
}
