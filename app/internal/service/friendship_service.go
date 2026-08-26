package service

import (
	"context"
	"errors"

	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)

type FriendshipService struct {
	friendshipRepo *repository.FriendshipRepository
}

func NewFriendshipService(friendshipRepo *repository.FriendshipRepository) *FriendshipService {
	return &FriendshipService{
		friendshipRepo: friendshipRepo,
	}
}
func (s *FriendshipService) CreateFriendship(requesterId int64, receiverId int64) (*model.Friendship, error) {
	if requesterId == receiverId {
		return nil, repository.ErrCannotFriendYourself
	}

	friendship, err := s.friendshipRepo.Create(nil, requesterId, receiverId)
	if err != nil {
		return nil, err
	}

	return friendship, nil

}

func (s *FriendshipService) SendFriendRequest(ctx context.Context, requesterId int64, receiverId int64) (*model.Friendship, error) {

	if requesterId == receiverId {
		return nil, repository.ErrCannotFriendYourself
	}

	existingFriendship, err := s.friendshipRepo.GetBetweenUsers(ctx, requesterId, receiverId)
	if err == nil && existingFriendship != nil {
		return nil, repository.ErrFriendshipAlreadyExists
	}

	if errors.Is(err, repository.ErrFriendshipNotFound) {
		return nil, err
	}

	return s.friendshipRepo.Create(ctx, requesterId, receiverId)
}


func (s *FriendshipService) AcceptFriendRequest(ctx context.Context, friendshipID int64 , receiverId int64) (*model.Friendship, error) {
	friendship , err := s.friendshipRepo.Accept(ctx, friendshipID , receiverId)

	if err != nil {
		return nil , err
	}

	return friendship , nil
}



func (s *FriendshipService) RejectFriendRequest(ctx context.Context, friendshipID int64 , receiverId int64) (*model.Friendship, error) {
	friendship , err := s.friendshipRepo.Reject(ctx, friendshipID , receiverId)

	if err != nil {
		return nil , err
	}

	return friendship , nil
}
