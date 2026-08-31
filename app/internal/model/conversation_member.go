package model

import "time"

const (
	ConversationDM   = "dm"
	ConversationRoom = "room"

	MemberOwner  = "owner"
	MemberAdmin  = "admin"
	MemberMember = "member"
)

type ConversationMember struct {
	ConversationID int64     `json:"conversation_id"`
	UserID         int64     `json:"user_id"`
	Role           string    `json:"role"`
	JoinedAt       time.Time `json:"joined_at"`
}
