package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/dto"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func validationError(err error) gin.H {
	return gin.H{
		"error": err.Error(),
	}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, validationError(err))
		return
	}
	user, err := h.userService.CreateUser(
		c.Request.Context(),
		req.Username,
		req.Email,
		req.Password,
	)
	if err != nil {
		// c.JSON(http.StatusInternalServerError , validationError(err))
		switch {
		case errors.Is(err, repository.ErrUsernameAlreadyExists):
			c.JSON(http.StatusConflict, validationError(err))
		case errors.Is(err, repository.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, validationError(err))
		default:
			c.JSON(http.StatusInternalServerError, validationError(err))
		}
		return
	}

	respone := dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	c.JSON(http.StatusCreated, respone)
}

func (h *UserHandler) GetUserByID(c *gin.Context) {
	idParam := c.Param("id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, validationError(repository.ErrInvalidUserID))
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			c.JSON(http.StatusNotFound, validationError(err))
		default:
			c.JSON(http.StatusInternalServerError, validationError(err))
		}
		return
	}
	response := dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}


func (h *UserHandler) GetMe(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user , err := h.userService.GetUserByID(c.Request.Context() , userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}
	response := dto.UserResponse{
		ID: 	  user.ID,
		Username: user.Username,
		Email:    user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}


func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserRequest

	if err := c.ShouldBindJSON(&req) ; err != nil {
		c.JSON(http.StatusBadRequest , validationError(err))
		return
	}

	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, req.Username, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			c.JSON(http.StatusNotFound, validationError(err))
		case errors.Is(err, repository.ErrUsernameAlreadyExists),
			errors.Is(err, repository.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, validationError(err))
		default:
			c.JSON(http.StatusInternalServerError, validationError(err))
		}
		return
	}

	response := dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}


func (h *UserHandler) DeleteMe(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	err := h.userService.DeleteUser(
		c.Request.Context(),
		userID,
	)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			c.JSON(http.StatusNotFound, validationError(err))

		default:
			c.JSON(http.StatusInternalServerError, validationError(err))
		}

		return
	}

	c.Status(http.StatusNoContent)
}
