package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BindStatus represents the status of a pending bind operation
type BindStatus struct {
	UserID    uint
	Status    string // "pending" / "success" / "error"
	AppID     string
	BotName   string
	CreatedAt time.Time
}

// bindTokenStore holds pending bind operations
var bindTokenStore = struct {
	tokens map[string]*BindStatus
	mu     sync.RWMutex
}{tokens: make(map[string]*BindStatus)}

// GetFeishuBots 获取用户绑定的飞书机器人列表
func GetFeishuBots(c *gin.Context) {
	userID := c.GetUint("userID")

	bots, err := repository.GetFeishuBotsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bots": bots})
}

// UnbindFeishuBot 解绑飞书机器人
func UnbindFeishuBot(c *gin.Context) {
	userID := c.GetUint("userID")
	botIDStr := c.Param("id")

	botID, err := strconv.ParseUint(botIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}

	if err := repository.DeleteFeishuBot(uint(botID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑失败"})
		return
	}

	// 同时删除关联的飞书会话
	repository.DeleteFeishuConvsByBotID(uint(botID))

	c.JSON(http.StatusOK, gin.H{"message": "解绑成功"})
}

// GetBindQRCode generates a QR code URL and bind token for App Manifest scan-to-create
// GET /api/feishu/bind-qrcode
func GetBindQRCode(c *gin.Context) {
	userID := c.GetUint("userID")

	// Generate bind token (UUID)
	bindToken := uuid.New().String()

	// Store pending status
	bindTokenStore.mu.Lock()
	bindTokenStore.tokens[bindToken] = &BindStatus{
		UserID:    userID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	bindTokenStore.mu.Unlock()

	// Generate QR code URL
	// Feishu App Manifest create URL format:
	// https://open.feishu.cn/app-manifest/create?token=<bind_token>
	qrcodeURL := fmt.Sprintf("https://open.feishu.cn/app-manifest/create?token=%s", bindToken)

	c.JSON(http.StatusOK, gin.H{
		"qrcode_url": qrcodeURL,
		"bind_token": bindToken,
	})
}

// GetBindStatus checks the status of a binding operation by token
// GET /api/feishu/bind-status/:token
func GetBindStatus(c *gin.Context) {
	bindToken := c.Param("token")

	bindTokenStore.mu.RLock()
	status, ok := bindTokenStore.tokens[bindToken]
	bindTokenStore.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "绑定请求不存在或已过期"})
		return
	}

	// Check if token expired (10 minutes)
	if time.Since(status.CreatedAt) > 10*time.Minute {
		bindTokenStore.mu.Lock()
		delete(bindTokenStore.tokens, bindToken)
		bindTokenStore.mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "绑定请求已过期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   status.Status,
		"app_id":   status.AppID,
		"bot_name": status.BotName,
	})
}

// HandleBindCallback is called when app_manifest.created WebSocket event is received
func HandleBindCallback(bindToken, appID, appSecret, tenantKey string) error {
	bindTokenStore.mu.Lock()
	status, ok := bindTokenStore.tokens[bindToken]
	if !ok {
		bindTokenStore.mu.Unlock()
		return fmt.Errorf("bind token not found: %s", bindToken)
	}
	bindTokenStore.mu.Unlock()

	// Create FeishuBot record
	bot := &model.FeishuBot{
		UserID:    status.UserID,
		AppID:     appID,
		AppSecret: appSecret,
		TenantKey: tenantKey,
		BotName:   "口播稿助手",
	}
	if err := repository.CreateFeishuBot(bot); err != nil {
		bindTokenStore.mu.Lock()
		status.Status = "error"
		bindTokenStore.mu.Unlock()
		return err
	}

	// Update status
	bindTokenStore.mu.Lock()
	status.Status = "success"
	status.AppID = appID
	bindTokenStore.mu.Unlock()

	return nil
}