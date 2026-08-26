package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/service"
)

type FriendshipHandler struct {
	friendshipService *service.FriendshipService
}

func NewFriendshipHandler(friendshipService *service.FriendshipService) *FriendshipHandler {
	return &FriendshipHandler{
		friendshipService: friendshipService,
	}
}

func (h *FriendshipHandler) SendFriendRequest(c *gin.Context) {

	receiverId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(400, gin.H{"error": "invalid receiver ID"})
		return
	}

	requesterId, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendship, err := h.friendshipService.SendFriendRequest(c.Request.Context(), requesterId, receiverId)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrCannotFriendYourself):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		case errors.Is(err, repository.ErrFriendshipAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		}
		return
	}
	c.JSON(http.StatusOK, friendship)
}

func (h *FriendshipHandler) AcceptFriendRequest(c *gin.Context) {
	friendshipId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship ID"})
		return
	}

	receiverId, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendship, err := h.friendshipService.AcceptFriendRequest(c.Request.Context(), friendshipId, receiverId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "friend request not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept friend request"})
		return
	}
	c.JSON(http.StatusOK, friendship)
}

func (h *FriendshipHandler) RejectFriendRequest(c *gin.Context) {
	friendshipId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship ID"})
		return
	}

	receiverId, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendship, err := h.friendshipService.RejectFriendRequest(c.Request.Context(), friendshipId, receiverId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "friend request not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject friend request"})
		return
	}
	c.JSON(http.StatusOK, friendship)
}
