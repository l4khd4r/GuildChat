package main

import (
	"log"

	"github.com/l4khd4r/GuildChat/internal/config"
	"github.com/l4khd4r/GuildChat/internal/database"
	"github.com/l4khd4r/GuildChat/internal/handler"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/router"
	"github.com/l4khd4r/GuildChat/internal/service"
)

func main() {
	cfg := config.Load()
	db, err := database.NewPostgresPool(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Postgres connection established")
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)
	authService := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authService)
	r := router.New(userHandler, authHandler)
	log.Println("Server is running on port : " + cfg.Port)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

}
