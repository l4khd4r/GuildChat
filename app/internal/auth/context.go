package auth

import "github.com/gin-gonic/gin"



func GetUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := value.(int64)
	if !ok {
		return 0, false
	}

	return userID, true
}
