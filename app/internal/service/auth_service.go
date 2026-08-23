package service

import (
	"context"
	"errors"

	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/crypto"
	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
)

type AuthService struct {
	userRepo *repository.UserRepository
	jwtManager *auth.JWTManager
}


var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func NewAuthService(userRepo *repository.UserRepository, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		jwtManager: jwtManager,
	}
}


func (s *AuthService) Login(ctx context.Context, email string,
	password string) (*model.User, string,  error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil,"",ErrInvalidCredentials
		}
	}
	valid, err := crypto.VerifyPassword(password, user.PasswordHash)
	if !valid || err != nil {
		return nil,"", ErrInvalidCredentials
	}

	token , err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil,"",err
	}

	return user,token,nil
}
