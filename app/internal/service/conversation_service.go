package service

import (
	"context"
	"errors"
	"strings"

	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)

type ConversationService struct {
	conversationRepo *repository.ConversationRepository
}

func NewConversationService(conversationRepo *repository.ConversationRepository) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
	}
}

func (s *ConversationService) GetOrCreateDM(ctx context.Context, userID1 int64, userID2 int64) (*model.Conversation, error) {
	if userID1 == userID2 {
		return nil, errors.New(model.ErrorCannotCreateDMWithSelf)
	}

	conversation, err := s.conversationRepo.GetDM(ctx, userID1, userID2)

	if err == nil {
		return conversation, nil
	}
	// here we create a new DM conversation

	if errors.Is(err, repository.ErrNotFound) {
		return nil, errors.New(model.ErrorConversationNotFound)
	}

	conversation, err = s.conversationRepo.CreateConversation(ctx,
		model.ConversationDM,
		nil, // no name for DM
		userID1)

	if err != nil {
		return nil, err
	}

	err = s.conversationRepo.AddMember(ctx, conversation.ID, userID1, model.MemberOwner)
	if err != nil {
		return nil, err
	}

	err = s.conversationRepo.AddMember(ctx, conversation.ID, userID2, model.MemberMember)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}

// ListUserConversations returns the caller's conversations, newest activity
// first. Membership is enforced by the query itself: a conversation the user
// is not a member of cannot appear in the result, so there is no separate
// authorisation check to forget here.
// The list is intentionally not split or filtered by type: DMs and rooms are
// one stream of conversations to the caller, and each entry carries its own
// Type for a client that wants to tell them apart.
func (s *ConversationService) ListUserConversations(ctx context.Context, userID int64) ([]*model.ConversationListEntry, error) {

	return s.conversationRepo.ListUserConversations(ctx, userID)
}

// GetUserConversation returns one of the caller's conversations. It returns
// repository.ErrConversationNotFound both when no such conversation exists and
// when it exists but the caller is not a member of it.
func (s *ConversationService) GetUserConversation(ctx context.Context, userID int64, conversationID int64) (*model.ConversationListEntry, error) {

	return s.conversationRepo.GetUserConversation(ctx, userID, conversationID)
}

// CreateRoom creates a room owned by the caller.
//
// Unlike GetOrCreateDM this is never idempotent: a room is identified by its
// own id, so two rooms with the same name are two different rooms, and every
// call returns a new one. The creator is the sole member, with the owner role;
// anyone else joins through the member endpoints.
func (s *ConversationService) CreateRoom(ctx context.Context, creatorID int64, name string) (*model.Conversation, error) {

	// A name of only spaces passes the handler's "required" binding but is not
	// a usable label, so it is rejected here rather than stored.
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, repository.ErrRoomNameRequired
	}

	return s.conversationRepo.CreateRoom(ctx, name, creatorID)
}
