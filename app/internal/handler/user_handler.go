package handler


import (
	"net/http"
	"github.com/l4khd4r/GuildChat/internal/service"
	"github.com/gin-gonic/gin"
)


type UserHandler struct {
	userService *service.UserService
}


func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}


func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest , gin.H {"error": "Invalid request payload"})
		return
	}
	user , err := h.userService.CreateUser(
		c.Request.Context(),
		req.Username,
		req.Email,
		req.Password,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError , gin.H {"error": "Failed to create user"})
		return
	}
	c.JSON(http.StatusCreated , user)
}
