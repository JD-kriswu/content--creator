package main

import (
	"log"
	"net/http"
	"strings"

	"content-creator-imm/config"
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/feishu"
	"content-creator-imm/internal/handler"
	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"
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

	// Initialize Feishu WebSocket if enabled
	if config.C.FeishuEnabled {
		feishuRouter := feishu.NewRouter(wfLoader)
		handler.SetFeishuRouter(feishuRouter)
		feishuPool := feishu.GetWSPool(config.C.FeishuWSReconnectMax, config.C.FeishuWSHeartbeatSec)

		// Connect all existing bots
		bots, _ := repository.GetConnectedFeishuBots()
		connectedCount := 0
		for _, bot := range bots {
			if err := feishuPool.Connect(bot.AppID, bot.AppSecret, feishuRouter.HandleEvent); err != nil {
				log.Printf("[Feishu] failed to connect bot %s: %v", bot.AppID, err)
			} else {
				connectedCount++
			}
		}
		log.Printf("[Feishu] initialized %d/%d WS connections", connectedCount, len(bots))
	}

	// Initialize web search service (optional)
	if config.C.WebSearchProvider != "" && config.C.WebSearchAPIKey != "" {
		service.InitWebSearchService(service.WebSearchConfig{
			Provider: config.C.WebSearchProvider,
			APIKey:   config.C.WebSearchAPIKey,
		})
		log.Printf("web search service initialized (provider: %s)", config.C.WebSearchProvider)
	} else {
		log.Println("web search service not configured (set web_search_provider and web_search_api_key in config.json)")
	}

	if config.C.AnthropicKey == "" {
		log.Println("ANTHROPIC_API_KEY not configured! Please set anthropic_api_key in config.json")
	}

	r := gin.Default()

	// CORS - allow configured origins (dev: localhost:5173, prod: same-origin via nginx)
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
		api.GET("/user/style/doc", handler.GetStyleDoc)
		api.PUT("/user/style", handler.UpdateStyle)

		api.GET("/chat/session", handler.GetSession)
		api.POST("/chat/reset", handler.ResetSession)
		api.POST("/chat/message", handler.SendMessage)

		api.GET("/scripts", handler.GetScripts)
		api.GET("/scripts/:id", handler.GetScript)

		api.GET("/conversations", handler.GetConversations)
		api.GET("/conversations/:id", handler.GetConversationDetail)
		api.DELETE("/conversations/:id", handler.DeleteConversation)

		// Prompt management (for editing YAML prompts)
		api.GET("/prompts", handler.GetPrompts)
		api.PUT("/prompts", handler.UpdatePrompt)

		// Feishu routes
		feishuAPI := api.Group("/feishu")
		{
			feishuAPI.GET("/bots", handler.GetFeishuBots)
			feishuAPI.DELETE("/bots/:id", handler.UnbindFeishuBot)
			feishuAPI.GET("/bind-stream", handler.StartBindFlow)        // SSE stream for bind flow
			feishuAPI.GET("/bind-status/:token", handler.GetBindStatus) // Polling fallback
			feishuAPI.DELETE("/bind/:token", handler.CancelBind)        // Cancel bind
		}
	}

	addr := ":" + config.C.Port
	log.Printf("server starting on %s (base: %q)", addr, base)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}