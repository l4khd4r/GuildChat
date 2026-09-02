package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/handler"
)

func New(userHandler *handler.UserHandler, authHandler *handler.AuthHandler, friendshipHandler *handler.FriendshipHandler, conversationHandler *handler.ConversationHandler, jwtManager *auth.JWTManager) *gin.Engine {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to My World!",
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})
	router.POST("/users", userHandler.CreateUser)
	router.GET("/users/:id", userHandler.GetUserByID)

	router.POST("/auth/login", authHandler.Login)

	protected := router.Group("/")
	protected.Use(jwtManager.Middleware())

	protected.GET("/me", userHandler.GetMe)
	protected.PUT("/me", userHandler.UpdateUser)
	protected.DELETE("/me", userHandler.DeleteMe)

	protected.POST("/users/:id/friend-request", friendshipHandler.SendFriendRequest)
	protected.POST("/friend-request/:id/accept", friendshipHandler.AcceptFriendRequest)

	protected.POST("/friend-request/:id/reject", friendshipHandler.RejectFriendRequest)
	protected.GET("/me/friends", friendshipHandler.ListFriends)
	protected.GET("/me/friend-requests", friendshipHandler.ListPendingFriendRequests)
	protected.GET("/me/friend-requests/sent", friendshipHandler.ListSentFriendRequests)

	protected.DELETE("/friend-request/:id", friendshipHandler.DeleteFriendRequest) // this is removing the row of the pending request
	protected.DELETE("/friends/:id", friendshipHandler.DeleteFriend)               // this is unfriend someone ( accepted )

	protected.POST("/conversations/dm/:id", conversationHandler.CreateDM) // :id is the *other user*; returns the existing DM if there already is one
	protected.POST("/conversations/room", conversationHandler.CreateRoom) // body: {"name": "..."}; always creates a new room, caller becomes owner

	// protected.DELETE("/conversations/:id", conversationHandler.DeleteConversationByID) // :id is the conversation id, not a user id
	protected.GET("/me/conversations", conversationHandler.ListUserConversations) // every conversation the user is a member of, DMs and rooms alike
	protected.GET("/conversations/:id", conversationHandler.GetConversation)      // one of them; 404 if it is not the caller's

	// Room membership. Both are scoped to a conversation the caller belongs to,
	// so someone outside it gets 404 rather than 403 on either.
	protected.GET("/conversations/:id/members", conversationHandler.ListMembers)              // the roster; any member may read it
	protected.POST("/conversations/:id/members", conversationHandler.AddMember)               // body: {"user_id": N}; owner/admin only, rooms only
	protected.DELETE("/conversations/:id/members/:user_id", conversationHandler.RemoveMember) // owner/admin only, rooms only
	return router
}
