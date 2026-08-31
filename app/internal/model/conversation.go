package model

import "time"

const (
	ErrorConversationNotFound = "conversation not found"
	ErrorCannotCreateDMWithSelf = "cannot create a DM with yourself"
)



type Conversation struct {
	ID        int64    `json:"id"`
	Type      string    `json:"type"`
	Name      *string   `json:"name,omitempty"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
