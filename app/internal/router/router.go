package router

import (
	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/handler"
)

func New(userHandler *handler.UserHandler, authHandler *handler.AuthHandler, friendshipHandler *handler.FriendshipHandler, jwtManager *auth.JWTManager) *gin.Engine {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to My World!",
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})
	router.POST("/users", userHandler.CreateUser)
	router.GET("/users/:id", userHandler.GetUserByID)

	router.POST("/auth/login", authHandler.Login)

	// Protected routes

	protected := router.Group("/")
	protected.Use(jwtManager.Middleware())

	protected.GET("/me", userHandler.GetMe)
	protected.PUT("/me", userHandler.UpdateUser)
	protected.DELETE("/me", userHandler.DeleteMe)

	protected.POST("/users/:id/friend-request", friendshipHandler.SendFriendRequest)

	return router
}
