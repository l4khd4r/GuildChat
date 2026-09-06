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

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

// SendMessage posts a message to a conversation the caller is a member of.
//
// 404 covers both a conversation that does not exist and one the caller is not
// in, so the endpoint cannot be used to probe ids. There is no 403 here at all:
// membership is the only permission, and a member who is refused would have
// nothing to be refused for.
//
// Sending the same client_msg_id twice returns the original message rather than
// creating a second one, so a client retrying after a dropped connection is
// safe to do so.

func (h *MessageHandler) SendMessage(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	request := dto.SendMessageRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	message, err := h.messageService.SendMessage(c.Request.Context(), conversationID, userID, request.Body, request.ClientMsgID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		case errors.Is(err, repository.ErrMessageBodyRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "message body is required"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": toMessageResponse(message)})
}

/*
	ListMessages returns once page of a conversation the caller is a member of,
	newest first.

	Same 404 rule as SendMEssage: a conversation that does not exist and one the
	caller is not in are answered identically

	A malformed `before` or `limit` is a 400 rather than a slient fallback to the
	defaults. A client that sent a cursor and got the newest page back instead
	would loop forever without ever seeing an error.
*/

func (h *MessageHandler) ListMessages(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	query := dto.ListMessagesQuery{}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}
	userID, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, err := h.messageService.ListMessages(c.Request.Context(), conversationID, userID, query.Before, query.Limit)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": toListMessagesResponse(page)})
}
