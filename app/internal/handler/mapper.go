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

// toConversationMemberResponse converts one roster entry to its wire form.
func toConversationMemberResponse(member *model.ConversationMemberEntry) dto.ConversationMemberResponse {
	return dto.ConversationMemberResponse{
		User:     toUserResponse(member.User),
		Role:     member.Role,
		JoinedAt: member.JoinedAt,
	}
}

// toConversationMemberResponses maps a roster, preserving order. Empty rather
// than nil, for the same reason as toUserResponses.
func toConversationMemberResponses(members []*model.ConversationMemberEntry) []dto.ConversationMemberResponse {
	responses := make([]dto.ConversationMemberResponse, 0, len(members))
	for _, member := range members {
		responses = append(responses, toConversationMemberResponse(member))
	}
	return responses
}

// toConversationResponse converts a freshly created conversation to its wire
// form. Name stays a pointer: it is null for a DM and set for a room, and
// `omitempty` drops the key entirely rather than emitting a null.
func toConversationResponse(conversation *model.Conversation) dto.ConversationResponse {
	return dto.ConversationResponse{
		ID:        conversation.ID,
		Type:      conversation.Type,
		Name:      conversation.Name,
		CreatedBy: conversation.CreatedBy,
		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}
}

// toConversationListItem converts one conversation-list row to its wire form.
// OtherUser stays nil for a room, which the `omitempty` tag turns into an
// absent key rather than `"other_user": null`.
func toConversationListItem(entry *model.ConversationListEntry) dto.ConversationListItem {
	item := dto.ConversationListItem{
		ID:          entry.Conversation.ID,
		Type:        entry.Conversation.Type,
		Name:        entry.Conversation.Name,
		MemberCount: entry.MemberCount,
		CreatedAt:   entry.Conversation.CreatedAt,
		UpdatedAt:   entry.Conversation.UpdatedAt,
	}
	if entry.OtherUser != nil {
		other := toUserResponse(entry.OtherUser)
		item.OtherUser = &other
	}
	return item
}

// toConversationListItems maps a slice of conversation-list rows, preserving
// order. Empty rather than nil, for the same reason as toUserResponses.
func toConversationListItems(entries []*model.ConversationListEntry) []dto.ConversationListItem {
	items := make([]dto.ConversationListItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toConversationListItem(entry))
	}
	return items
}
