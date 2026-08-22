package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/dto"
	"github.com/l4khd4r/GuildChat/internal/service"
)


type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler{
	return &AuthHandler{
		authService : authService ,
	}
}


func ValidationError(err error) gin.H {
	return gin.H{
		"error": err.Error(),
	}
}


func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req) ; err != nil {
		c.JSON(http.StatusBadRequest , ValidationError(err))
		return
	}


	user, err := h.authService.Login(c.Request.Context() , req.Email , req.Password)
	if err != nil {

		switch {
			case errors.Is(err , service.ErrInvalidCredentials):
				c.JSON(http.StatusUnauthorized , ValidationError(err))
			default:
				c.JSON(http.StatusInternalServerError , ValidationError(err))
		}
		return
	}

	c.JSON(http.StatusOK , gin.H{
		"id": user.ID,
		"username": user.Username,
		"email": user.Email,
	})

}
