package service

import (
	"context"
	"errors"

	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)

// FriendshipService holds the business rules for friendships. The handler
// above it deals only in HTTP, and the repository below it only in SQL; the
// decisions about what is and is not allowed live here.
type FriendshipService struct {
	friendshipRepo *repository.FriendshipRepository
}

func NewFriendshipService(friendshipRepo *repository.FriendshipRepository) *FriendshipService {
	return &FriendshipService{
		friendshipRepo: friendshipRepo,
	}
}

// SendFriendRequest creates a pending friendship from requesterId to
// receiverId.
//
// Two rules are enforced here rather than in the database: you cannot befriend
// yourself, and a pair that already has a friendship row (in either direction,
// in any status) cannot get a second one.
//
// The order of the checks below matters. GetBetweenUsers returns
// ErrFriendshipNotFound when there is no row, which is the *success* path here
// and the only one that continues to Create. Any other error is a real failure
// and is returned as-is, so a database problem is never mistaken for "no
// friendship yet" and silently turned into an insert.
//
// The lookup is still a check-then-act, so two simultaneous requests can both
// pass it. Create closes that hole by mapping the unique index violation to
// ErrFriendshipAlreadyExists.
func (s *FriendshipService) SendFriendRequest(ctx context.Context, requesterId int64, receiverId int64) (*model.Friendship, error) {

	if requesterId == receiverId {
		return nil, repository.ErrCannotFriendYourself
	}

	_, err := s.friendshipRepo.GetBetweenUsers(ctx, requesterId, receiverId)
	switch {
	case err == nil:
		// Any existing row means the pair is already linked, whatever its status.
		return nil, repository.ErrFriendshipAlreadyExists
	case !errors.Is(err, repository.ErrFriendshipNotFound):
		return nil, err
	}

	// No friendship between them yet, so create one.
	return s.friendshipRepo.Create(ctx, requesterId, receiverId)
}

// AcceptFriendRequest marks a pending request as accepted.
//
// receiverId is the authenticated caller and is passed down to the repository,
// where it becomes part of the UPDATE's WHERE clause: only the person a
// request was addressed to can accept it. A request that does not exist, is
// addressed to someone else, or is no longer pending all come back as
// ErrFriendshipNotFound.
func (s *FriendshipService) AcceptFriendRequest(ctx context.Context, friendshipID int64, receiverId int64) (*model.Friendship, error) {
	friendship, err := s.friendshipRepo.Accept(ctx, friendshipID, receiverId)

	if err != nil {
		return nil, err
	}

	return friendship, nil
}

// RejectFriendRequest turns down a pending request.
//
// Same authorisation rule as AcceptFriendRequest. Note this deletes the row
// instead of storing status = "rejected"; see the repository method for the
// reasoning.
func (s *FriendshipService) RejectFriendRequest(ctx context.Context, friendshipID int64, receiverId int64) (*model.Friendship, error) {
	friendship, err := s.friendshipRepo.Reject(ctx, friendshipID, receiverId)

	if err != nil {
		return nil, err
	}

	return friendship, nil
}

// ListFriends returns the users userID has an accepted friendship with,
// regardless of who sent the original request.
func (s *FriendshipService) ListFriends(ctx context.Context, userID int64) ([]*model.User, error) {
	friends, err := s.friendshipRepo.ListFriends(ctx, userID)
	if err != nil {
		return nil, err
	}

	return friends, nil
}

// ListPendingRequests returns the still-unanswered requests userID has
// received, each paired with the user who sent it.
func (s *FriendshipService) ListPendingRequests(ctx context.Context, userID int64) ([]*model.FriendRequest, error) {
	pendingRequests, err := s.friendshipRepo.ListPendingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pendingRequests, nil
}

// ListSentRequests returns the still-unanswered requests userID has sent,
// each paired with the user it was sent to.
func (s *FriendshipService) ListSentRequests(ctx context.Context, userID int64) ([]*model.FriendRequest, error) {
	sentRequests, err := s.friendshipRepo.ListSentRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	return sentRequests, nil
}

// DeleteFriendRequest removes a pending request.
//
// Accept and reject are the receiver's answers; this one is open to both
// parties - the sender cancels, the receiver dismisses. The authorisation rule
// lives in the repository's WHERE clause, as it does for the other two: a
// caller who is neither side, an id that does not exist, and a friendship that
// is no longer pending all come back as ErrFriendshipNotFound.
//
// The error is returned as-is. Wrapping it in a fresh errors.New would flatten
// it to a string and cost the handler its errors.Is check, turning every 404
// into a 500.
func (s *FriendshipService) DeleteFriendRequest(ctx context.Context, friendshipID int64, userID int64) error {
	return s.friendshipRepo.Delete(ctx, friendshipID, userID)
}

func (s *FriendshipService) DeleteFriend(ctx context.Context, friendshipID int64, userID int64) error {
	return s.friendshipRepo.DeleteAccepted(ctx, friendshipID, userID)
}
