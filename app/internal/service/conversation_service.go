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

// GetOrCreateDM returns the DM between two users, opening it if there is not
// one yet. Calling it twice with the same pair gives the same conversation:
// a DM is identified by who is in it, not by an id the caller chose.
//
// The error from GetDM decides which half runs, and only ErrConversationNotFound
// means "create". Every other error is returned as it stands, because a DM that
// exists but could not be read right now is not a DM that needs creating --
// treating a timeout or a dropped connection as absence would open a second
// conversation for a pair that already has one, and neither user would ever see
// the first again.
//
// The creation itself is one repository call. It writes the conversation and
// both membership rows in a single transaction, which is not something this
// layer could arrange out of separate calls.
func (s *ConversationService) GetOrCreateDM(ctx context.Context, userID1 int64, userID2 int64) (*model.Conversation, error) {
	if userID1 == userID2 {
		return nil, repository.ErrCannotDMYourself
	}

	conversation, err := s.conversationRepo.GetDM(ctx, userID1, userID2)

	if err == nil {
		return conversation, nil
	}

	if !errors.Is(err, repository.ErrConversationNotFound) {
		return nil, err
	}

	return s.conversationRepo.CreateDM(ctx, userID1, userID2)
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

// RemoveMember takes a member out of a room.
//
// The rule is model.CanRemove: strictly higher rank wins, so an owner removes
// admins and members, an admin removes members but not a fellow admin and not
// the owner, and a member removes nobody. That subsumes CanManageMembers --
// nothing below admin can outrank a real role -- so the ladder is consulted
// once, here, rather than twice with two different answers.
//
// Leaving is deliberately not handled. CanRemove(role, role) is false, so a
// caller naming themselves is refused rather than quietly allowed. Self-removal
// has its own rule (an owner cannot walk out and leave the room ownerless) and
// belongs in its own endpoint.
func (s *ConversationService) RemoveMember(ctx context.Context, callerID int64, conversationID int64, memberID int64) error {
	access, err := s.conversationRepo.GetConversationAccess(ctx, conversationID, callerID)
	if err != nil {
		return err
	}

	if access.ConversationType != model.ConversationRoom {
		return repository.ErrNotARoom
	}

	// The target's role is what the caller's rank is measured against, so it
	// has to be read, not assumed. A non-member answers ErrConversationNotFound
	// here, but the caller has already proved the room exists and that they are
	// in it, so for them it means "that user is not a member" -- there is
	// nothing left to hide, and the two deserve different status codes.
	target, err := s.conversationRepo.GetConversationAccess(ctx, conversationID, memberID)
	if errors.Is(err, repository.ErrConversationNotFound) {
		return repository.ErrNotMember
	}
	if err != nil {
		return err
	}

	if !model.CanRemove(access.Role, target.Role) {
		return repository.ErrForbidden
	}

	return s.conversationRepo.RemoveMember(ctx, conversationID, memberID)
}
