package router


import "github.com/gin-gonic/gin"
import "github.com/l4khd4r/GuildChat/internal/handler"


func New(userHandler *handler.UserHandler) *gin.Engine {
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

	return  router
}
