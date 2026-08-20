package router


import "github.com/gin-gonic/gin"



func New() *gin.Engine {
	router := gin.Default()


	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to My World!"
		})
	})


	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	}
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	}


	router.GET("/hello/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.JSON(200, gin.H{
			"message": "Hello, " + name + "!",
		})
	})

	router.GET("/Forbidden", func(c *gin.Context) {
		c.JSON(403, gin.H{
			"message": "Forbidden",
		})
	}
	return  router
}
