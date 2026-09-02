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

	// Both sides of a DM are plain members. "Owner" is a room concept: giving
	// it to whoever happened to open the conversation would mean, once roles
	// have teeth, that one participant could administer a conversation between
	// equals.
	err = s.conversationRepo.AddMember(ctx, conversation.ID, userID1, model.MemberMember)
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

// ListMembers returns the roster of a conversation the caller belongs to.
//
// The GetMemberRole call is the authorisation: it fails with
// ErrConversationNotFound for a stranger, which is the same answer a
// non-existent id gives. Any member may read the roster -- knowing who else is
// in a room you are already in is not privileged.
func (s *ConversationService) ListMembers(ctx context.Context, userID int64, conversationID int64) ([]*model.ConversationMemberEntry, error) {

	if _, err := s.conversationRepo.GetConversationAccess(ctx, conversationID, userID); err != nil {
		return nil, err
	}

	return s.conversationRepo.ListMembers(ctx, conversationID)
}

// AddMember puts a user into a room.
//
// The checks run in this order, and the order is the point -- each one is only
// safe to ask once the previous has passed:
//
//  1. Does the caller have access at all? A stranger cannot get past this, and
//     the error they get is the same 404 a made-up id gives, so the endpoint
//     leaks nothing about which conversations exist.
//  2. Is this a room? A DM's membership is its two participants, for life;
//     adding a third would turn it into something the DM queries cannot
//     describe (GetDM and the other-user lookup both assume exactly two).
//  3. May the caller manage members? Only now that they are known to be *in*
//     the conversation is it safe to say "your role is too low" -- to anyone
//     else that sentence would confirm the conversation exists.
//
// The new member always joins as a plain member. Promoting to admin is a
// separate operation with its own rule (only an owner should be able to), and
// folding it in here would mean this endpoint could hand out its own
// permission.
func (s *ConversationService) AddMember(ctx context.Context, callerID int64, conversationID int64, newMemberID int64) error {

	access, err := s.conversationRepo.GetConversationAccess(ctx, conversationID, callerID)
	if err != nil {
		return err
	}

	if access.ConversationType != model.ConversationRoom {
		return repository.ErrNotARoom
	}

	if !model.CanManageMembers(access.Role) {
		return repository.ErrForbidden
	}

	return s.conversationRepo.AddMember(ctx, conversationID, newMemberID, model.MemberMember)
}

func (s *ConversationService) RemoveMember(ctx context.Context, callerID int64, conversationID int64, memberID int64) error {
	access, err := s.conversationRepo.GetConversationAccess(ctx, conversationID, callerID) // just for not making it the pain in the ass , anyway i will check for the access later

	if err != nil {
		return err
	}

	if access.ConversationType != model.ConversationRoom {
		return repository.ErrNotARoom
	}

	if !model.CanManageMembers(access.Role) {
		return repository.ErrForbidden
	}

	err = s.conversationRepo.RemoveMember(ctx, conversationID, memberID)

	if err != nil {
		return err
	}
	return nil
}
