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

type ConversationHandler struct {
	conversationService *service.ConversationService
}

func NewConversationHandler(conversationService *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
	}
}

// CreateDM opens the DM between the caller and the user in :id.
//
// It is deliberately its own handler rather than a general "create a
// conversation" endpoint. A DM is identified by who is in it, so this is
// get-or-create: calling it twice with the same pair must return the same
// conversation. A room is identified by its own id and is created fresh every
// time, takes a name and a member list, and gives its creator the owner role.
// The two share no input, no validation and no idempotency rule, so folding
// them together would only produce one handler branching on a type field.
//
// Note that :id here is a *user* id, not a conversation id.
func (h *ConversationHandler) CreateDM(c *gin.Context) {
	userID2, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID1, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conversation, err := h.conversationService.GetOrCreateDM(c.Request.Context(), userID1, userID2)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"conversation": toConversationResponse(conversation)})
}

// CreateRoom creates a room owned by the caller.
//
// Always a create, never a get-or-create: unlike a DM, a room is identified by
// its own id, so two rooms of the same name are two different rooms and every
// call returns a new one. That is why this takes a body and no :id, where
// CreateDM takes a user id and no body.
//
// The room starts with one member -- the creator, as owner. Adding anyone else
// is a separate endpoint.
func (h *ConversationHandler) CreateRoom(c *gin.Context) {
	var request dto.CreateRoomRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required (1-255 characters)"})
		return
	}

	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conversation, err := h.conversationService.CreateRoom(c.Request.Context(), userID, request.Name)

	if err != nil {
		if errors.Is(err, repository.ErrRoomNameRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required (1-255 characters)"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"conversation": toConversationResponse(conversation)})
}

// GetConversation returns one conversation the caller is a member of, in the
// same shape as an entry of GET /me/conversations -- a client that follows a
// link into a conversation gets the object it already knows how to render.
//
// A conversation the caller is not a member of answers 404, not 403: saying
// "forbidden" would confirm that the id exists.
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conversation, err := h.conversationService.GetUserConversation(c.Request.Context(), userID, conversationID)

	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversation": toConversationListItem(conversation)})
}

func (h *ConversationHandler) ListUserConversations(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	conversations, err := h.conversationService.ListUserConversations(c.Request.Context(), userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversations": toConversationListItems(conversations)})
}
