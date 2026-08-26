package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (m *JWTManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Authorization header is missing"})
			return
		}

		splits := strings.SplitN(authHeader, " ", 2)

		if len(splits) != 2 || splits[0] != "Bearer" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}
		token := splits[1]

		claims, err := m.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
