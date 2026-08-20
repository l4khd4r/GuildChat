package main

import "github.com/gin-gonic/gin"

import "github.com/l4khd4r/GuildChat/internal/router"

func main() {
	router := router.New()
	router.Run(":8080")
}
