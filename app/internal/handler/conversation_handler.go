package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
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



func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	userID2 , err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID1, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}




	conversation , err := h.conversationService.GetOrCreateDM( c.Request.Context() , userID1, userID2)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"conversation": conversation}) // later we will make our dto for this conversation response
}
