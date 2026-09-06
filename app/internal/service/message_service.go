package service

import (
	"context"
	"strings"

	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)

// MessageService holds the conversation repository as well as its own, because
// every message operation is gated on the caller's membership and that fact
// lives on the conversation side. Asking it directly keeps the authorisation
// one call rather than a service calling a service.

type MessageService struct {
	conversationRepo *repository.ConversationRepository
	messageRepo      *repository.MessageRepository
}

func NewMessageService(conversationRepo *repository.ConversationRepository, messageRepo *repository.MessageRepository) *MessageService {
	return &MessageService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
	}
}

// SendMessage posts a message to a conversation the caller belongs to.
//
// The access check is the authorisation and it comes first: a stranger gets
// ErrConversationNotFound, the same answer a made-up id gives, so this endpoint
// cannot be used to discover which conversations exist.
//
// No role is consulted. Any member of a conversation may speak in it, in a room
// and a DM alike. Muting is a rule that does not exist yet, and if it ever does
// it belongs here rather than in the repository.

func (s *MessageService) SendMessage(ctx context.Context, conversationID int64, senderID int64, body string, clientMsgID *string) (*model.MessageEntry, error) {

	if _, err := s.conversationRepo.GetConversationAccess(ctx, conversationID, senderID); err != nil {
		return nil, err
	}
	// A body of only spaces passes the binding's "required" but is not a
	// message, so it is rejected here rather than stored.
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, repository.ErrMessageBodyRequired
	}

	return s.messageRepo.CreateMessage(ctx, conversationID, senderID, body, clientMsgID)
}

/*
	Page sizes. DefaultMEssageLimit is what a client that asks for nothgin gets;
	MaxMessageLimit is also defalred in the binding tag on ListMessagesQuery, so change both
	together
*/

const (
	DefaultMessageLimit = 50
	MaxMessageLimit     = 100
)

/*
	ListMessages returns one page of a conversation the caller belongs to

	The access check is the same one SendMessage does, and for the same reason:
	a stanger gets ErrConversationNotFound, so reading is no more useful for probing
	ids than writing is. No role is consultedm because any member may read
	what any memebr may write


	The repository is asked for one row more than the caller wanted. If it comes
	back full, there is at least one older message, and the extra row is dropped before it chips.
	This is why HasMore costs nothing: it is the same read
*/

func (s *MessageService) ListMessages(ctx context.Context, conversationID int64, userID int64, before *int64, limit int) (*model.MessagePage, error) {
	if _, err := s.conversationRepo.GetConversationAccess(ctx, conversationID, userID); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = DefaultMessageLimit
	}

	if limit > MaxMessageLimit {
		limit = MaxMessageLimit
	}

	entries, err := s.messageRepo.ListMessages(ctx, conversationID, before, limit+1)

	if err != nil {
		return nil, err
	}

	page := &model.MessagePage{}

	page.Messages = entries

	if len(entries) > limit {
		page.Messages = entries[:limit]
		page.HasMore = true

		/*
			The cursor is the oldest id on the page being returned, not the id
			of the row that was trimmed. next request asks for what comes
			strictly before it, so the trimmed row is the first thing it finds

		*/
		oldest := entries[len(page.Messages)-1].Message.ID
		page.NextCursor = &oldest
	}
	return page, nil
}
