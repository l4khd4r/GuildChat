package service

import (
	"context"
	"errors"

	"github.com/l4khd4r/GuildChat/internal/crypto"
	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

func (s *AuthService) Login(ctx context.Context, email string,
	password string) (*model.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
	}
	valid, err := crypto.VerifyPassword(password, user.PasswordHash)
	if !valid || err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}
