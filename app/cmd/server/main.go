package main

import (
	"log"
	"time"

	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/config"
	"github.com/l4khd4r/GuildChat/internal/database"
	"github.com/l4khd4r/GuildChat/internal/handler"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/router"
	"github.com/l4khd4r/GuildChat/internal/service"
)

func main() {
	cfg := config.Load()
	dbCfg := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}

	// Bring the schema up to date before serving, so the app never talks to a
	// database it does not match.
	if err := database.MigrateUp(dbCfg); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Database schema is up to date")

	db, err := database.NewPostgresPool(dbCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	privateKey, err := auth.LoadPrivateKey(cfg.JWT.PrivateKeyPath)
	if err != nil {
		log.Fatalf("failed to load private key: %v", err)
	}
	publicKey, err := auth.LoadPublicKey(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatalf("failed to load public key: %v", err)
	}

	jwtManager := auth.NewJWTManager(privateKey, publicKey, "GuildChat", 24*time.Hour)

	log.Println("Postgres connection established")
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	friendshipRepo := repository.NewFriendshipRepository(db)
	friendshipService := service.NewFriendshipService(friendshipRepo)
	friendshipHandler := handler.NewFriendshipHandler(friendshipService)

	authService := service.NewAuthService(userRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authService)
	conversationRepo := repository.NewConversationRepository(db)
	conversationService := service.NewConversationService(conversationRepo)
	conversationHandler := handler.NewConversationHandler(conversationService)
	r := router.New(userHandler, authHandler, friendshipHandler, conversationHandler, jwtManager)
	log.Println("Server is running on port : " + cfg.Port)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

}
