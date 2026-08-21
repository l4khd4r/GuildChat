package service

import (
	"context"

	"github.com/l4khd4r/GuildChat/internal/crypto"
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
) (*repository.User, error) {
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
