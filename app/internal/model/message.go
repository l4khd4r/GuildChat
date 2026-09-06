package model

import "time"

type Message struct {
	ID             int64      `json:"id"`
	ConversationID int64      `json:"conversation_id"`
	SenderID       int64      `json:"sender_id"`
	Body           string     `json:"body"`
	ClientMsgID    *string    `json:"client_msg_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
}

// MessageEntry is a message together with the person who sent it.
//
// The sender travels with the message for the same reason the roster embeds
// the whole user: a list of sender ids would force the client into an N+1 just
// to draw names, on the one endpoint it hits hardest.

type MessageEntry struct {
	Message *Message
	Sender  *User
}

/*
	MessagePage is one page of messages plus what the clients needs to ask for
	the next one

	NextCursor is the id to send back as `before`. It is nil when this page
	reaches the beginning of the conversations which is also what HasMore says ;
	carrying both means the client can render "you've reached the top" without
	inspecting the messages themselves.
*/

type MessagePage struct {
	Messages   []*MessageEntry
	NextCursor *int64
	HasMore    bool
}
