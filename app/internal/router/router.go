package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/handler"
)

func New(userHandler *handler.UserHandler, authHandler *handler.AuthHandler, friendshipHandler *handler.FriendshipHandler , conversationHandler *handler.ConversationHandler, jwtManager *auth.JWTManager) *gin.Engine {
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

	protected.POST("/conversations/:id" , conversationHandler.CreateConversation) // this is for creating a conversation with a friend ( if the conversation already exists it will return the existing one)


	return router
}
