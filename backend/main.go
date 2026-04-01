package main

import (
	"log"
	"net/http"
	"strings"

	"content-creator-imm/config"
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/handler"
	"content-creator-imm/internal/workflow"
	"content-creator-imm/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()

	if err := db.Init(); err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	log.Println("database connected")

	// Initialize workflow loader (dev mode reloads YAML on every request)
	wfLoader := workflow.NewLoader("workflows", true)
	handler.SetWorkflowLoader(wfLoader)

	if config.C.AnthropicKey == "" {
		log.Println("⚠️  ANTHROPIC_API_KEY 未配置！请在 config.json 中设置 anthropic_api_key")
	}

	r := gin.Default()

	// CORS — allow configured origins (dev: localhost:5173, prod: same-origin via nginx)
	allowedOrigins := strings.Split(config.C.CORSOrigins, ",")
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, allowed := range allowedOrigins {
			if strings.TrimSpace(allowed) == origin || strings.TrimSpace(allowed) == "*" {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	base := strings.TrimRight(config.C.BasePath, "/")

	// Auth routes
	auth := r.Group(base + "/api/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
	}

	// Protected routes
	api := r.Group(base+"/api", middleware.Auth())
	{
		api.GET("/user/profile", handler.GetProfile)
		api.PUT("/user/style", handler.UpdateStyle)

		api.GET("/chat/session", handler.GetSession)
		api.POST("/chat/reset", handler.ResetSession)
		api.POST("/chat/message", handler.SendMessage)

		api.GET("/scripts", handler.GetScripts)
		api.GET("/scripts/:id", handler.GetScript)

		api.GET("/conversations", handler.GetConversations)
		api.GET("/conversations/:id", handler.GetConversationDetail)
	}

	addr := ":" + config.C.Port
	log.Printf("server starting on %s (base: %q)", addr, base)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
