package model

import "time"

type Conversation struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Name      *string   `json:"name,omitempty"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ConversationListEntry is one row of a user's conversation list. It is the
// conversation itself plus the extra facts needed to render a sidebar row,
// which differ by type:
//
//   - dm   -> OtherUser is the person on the other side. A DM has no name, so
//     without this the row has nothing to display.
//   - room -> OtherUser is nil; the room's Name is the label, and MemberCount
//     is all the client needs beyond it. The full roster is a separate
//     request, so this list does not grow with the size of a room.
//
// MemberCount is filled for both types; for a DM it is always 2.
type ConversationListEntry struct {
	Conversation *Conversation
	OtherUser    *User
	MemberCount  int
}
