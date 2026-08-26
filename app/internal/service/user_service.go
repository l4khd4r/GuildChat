package service

import (
	"context"

	"github.com/l4khd4r/GuildChat/internal/crypto"
	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) CreateUser(
	ctx context.Context,
	username string,
	email string,
	password string,
) (*model.User, error) {
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, err
	}
	return s.userRepo.CreateUser(
		ctx,
		username,
		email,
		passwordHash,
	)
}

func (s *UserService) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) UpdateUser(ctx context.Context, id int64, username string, email string) (*model.User, error) {
	return s.userRepo.Update(ctx, id, username, email)
}

func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	return s.userRepo.Delete(ctx, id)
}
