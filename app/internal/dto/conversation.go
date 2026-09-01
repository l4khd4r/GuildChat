package dto

import "time"

// ConversationListItem is one entry of GET /me/conversations.
//
// The shape carries exactly what a client needs to draw a conversation row and
// nothing else, which is why the two conversation types fill different fields:
//
//	dm   -> Name is null and OtherUser is set. The other person is the row's
//	        label; there is no other name to show.
//	room -> Name is the label and OtherUser is null.
//
// MemberCount is present for both (a DM is always 2). The full member list is
// deliberately not here: it would make this response grow with the size of the
// largest room, on an endpoint the client hits every time it opens. Fetch the
// roster from the conversation's own members endpoint when a room is actually
// opened.
type ConversationListItem struct {
	ID          int64         `json:"id"`
	Type        string        `json:"type"`
	Name        *string       `json:"name,omitempty"`
	OtherUser   *UserResponse `json:"other_user,omitempty"`
	MemberCount int           `json:"member_count"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CreateRoomRequest is the body of POST /conversations/room.
//
// A name is required, and that is the whole difference from a DM: a room is
// identified by its own id and needs a label, where a DM is identified by who
// is in it and has none. Max 255 matches the column.
//
// There is no member list here on purpose. Creating a room and populating it
// are separate operations, so that adding a member has one code path (with one
// permission check) rather than two.
type CreateRoomRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

// ConversationResponse is the conversation itself, returned by the endpoints
// that create one:
//
//	POST /conversations/room
//	POST /conversations/dm/:id
//
// It is the plain row, without the list's derived extras -- at creation there
// is nothing to derive yet. Use ConversationListItem for reading.
type ConversationResponse struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Name      *string   `json:"name,omitempty"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
