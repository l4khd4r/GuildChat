package auth

import "github.com/gin-gonic/gin"

const userIDKey = "user_id"

func GetUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return 0, false
	}

	userID, ok := value.(int64)
	if !ok {
		return 0, false
	}

	return userID, true
}

func GetUserIDFromContext(c *gin.Context) (int64, bool) {
	return GetUserID(c)
}
