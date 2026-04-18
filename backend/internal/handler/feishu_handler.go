package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"content-creator-imm/config"
	"content-creator-imm/internal/feishu"
	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BindSession represents an active bind session with lark-op-cli
type BindSession struct {
	UserID      uint
	Token       string
	Status      string // "pending" / "scanning" / "creating" / "success" / "error"
	AppID       string
	AppSecret   string
	TenantKey   string
	BotName     string
	QRCode      string // ASCII QR code content
	Error       string
	CreatedAt   time.Time
	Cmd         *exec.Cmd
	CancelCtx   context.Context
	CancelFunc  context.CancelFunc
	OutputChan  chan string // Channel for CLI output lines
}

// bindSessions holds active bind sessions
var bindSessions = struct {
	sessions map[string]*BindSession
	mu       sync.RWMutex
}{sessions: make(map[string]*BindSession)}

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

// StartBindFlow starts the lark-op-cli create-bot flow and returns SSE stream
// GET /api/feishu/bind-stream
func StartBindFlow(c *gin.Context) {
	userID := c.GetUint("userID")

	// Generate unique bind token
	bindToken := uuid.New().String()

	// Create context with timeout (5 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// Create bind session
	session := &BindSession{
		UserID:     userID,
		Token:      bindToken,
		Status:     "pending",
		CreatedAt:  time.Now(),
		CancelCtx:  ctx,
		CancelFunc: cancel,
		OutputChan: make(chan string, 100),
	}

	bindSessions.mu.Lock()
	bindSessions.sessions[bindToken] = session
	bindSessions.mu.Unlock()

	// Start CLI command in background
	go runLarkCLI(session, userID)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Stream events to client
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SSE not supported"})
		return
	}

	// Send initial event with token
	writeSSE(c, "init", gin.H{"bind_token": bindToken})
	flusher.Flush()

	// Stream CLI output
	timeout := time.After(5 * time.Minute)
	for {
		select {
		case <-timeout:
			writeSSE(c, "error", gin.H{"message": "绑定超时"})
			flusher.Flush()
		 cleanupSession(bindToken)
			return
		case <-c.Request.Context().Done():
			// Client disconnected
		 cleanupSession(bindToken)
			return
		case line, ok := <-session.OutputChan:
			if !ok {
				// Channel closed, CLI finished
				return
			}

			// Parse the output line and send appropriate SSE event
			event := parseCLIOutput(line, session)
			if event != nil {
				eventType := event["type"].(string)
				data := event["data"].(gin.H)
				writeSSE(c, eventType, data)
				flusher.Flush()

				// If success or error, end stream
				if session.Status == "success" || session.Status == "error" {
					return
				}
			}
		}
	}
}

// runLarkCLI executes the lark-op-cli create-bot command
func runLarkCLI(session *BindSession, userID uint) {
	// Find chromium browser
	chromePath := findChromiumPath()
	if chromePath == "" {
		session.OutputChan <- "ERROR:未找到浏览器，请安装 Chromium"
		session.Status = "error"
		session.Error = "未找到浏览器"
		close(session.OutputChan)
		return
	}

	// Build command
	botName := fmt.Sprintf("口播稿助手-%d", time.Now().Unix()%10000)
	cmd := exec.CommandContext(session.CancelCtx,
		"npx", "-y", "lark-op-cli@latest", "create-bot",
		"--name", botName,
		"--desc", "AI驱动的爆款口播稿改写工具",
		"--timeout", "300",
		"--browser-args", "--headless,--no-sandbox,--disable-gpu",
	)

	// Set environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("CHROME_PATH=%s", chromePath))

	session.Cmd = cmd
	session.BotName = botName

	// Capture output
	var outputBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(&outputBuf, &lineWriter{ch: session.OutputChan})
	cmd.Stderr = io.MultiWriter(&outputBuf, &lineWriter{ch: session.OutputChan})

	session.Status = "scanning"
	session.OutputChan <- "STATUS:scanning"

	// Run command
	err := cmd.Run()

	// Parse final output
	fullOutput := outputBuf.String()

	if err != nil && session.CancelCtx.Err() == context.DeadlineExceeded {
		session.Status = "error"
		session.Error = "绑定超时"
		session.OutputChan <- "ERROR:绑定超时"
	} else if err != nil {
		session.Status = "error"
		session.Error = err.Error()
		session.OutputChan <- fmt.Sprintf("ERROR:%s", err.Error())
	} else {
		// Parse success output to extract app_id and app_secret
		appID, appSecret, tenantKey := parseCLISuccessOutput(fullOutput)
		if appID != "" {
			session.Status = "success"
			session.AppID = appID
			session.AppSecret = appSecret
			session.TenantKey = tenantKey

			// Save to database
			bot := &model.FeishuBot{
				UserID:    userID,
				AppID:     appID,
				AppSecret: appSecret,
				TenantKey: tenantKey,
				BotName:   botName,
			}
			if err := repository.CreateFeishuBot(bot); err != nil {
				session.Status = "error"
				session.Error = "保存机器人信息失败"
				session.OutputChan <- "ERROR:保存机器人信息失败"
			} else {
				// Establish WebSocket connection for new bot
				if config.C.FeishuEnabled && feishuRouter != nil {
					pool := feishu.GetWSPool(config.C.FeishuWSReconnectMax, config.C.FeishuWSHeartbeatSec)
					if err := pool.Connect(appID, appSecret, feishuRouter.HandleEvent); err != nil {
						log.Printf("[FeishuBind] failed to establish WS connection: %v", err)
					} else {
						log.Printf("[FeishuBind] WS connection established for app_id=%s", appID)
					}
				}
				session.OutputChan <- fmt.Sprintf("SUCCESS:%s|%s", appID, botName)
			}
		} else {
			session.Status = "error"
			session.Error = "无法解析创建结果"
			session.OutputChan <- "ERROR:无法解析创建结果"
		}
	}

	close(session.OutputChan)

	// Clean up session after 1 minute
	time.AfterFunc(1*time.Minute, func() {
	 cleanupSession(session.Token)
	})
}

// lineWriter wraps a channel to send lines
type lineWriter struct {
	ch   chan string
	buf  []byte
}

func (w *lineWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx == -1 {
			break
		}
		line := string(w.buf[:idx])
		if strings.TrimSpace(line) != "" {
			w.ch <- line
		}
		w.buf = w.buf[idx+1:]
	}
	return len(p), nil
}

// parseCLIOutput parses a line from CLI output and returns SSE event data
func parseCLIOutput(line string, session *BindSession) map[string]interface{} {
	// Check for special prefixes
	if strings.HasPrefix(line, "STATUS:") {
		status := strings.TrimPrefix(line, "STATUS:")
		session.Status = status
		return map[string]interface{}{
			"type": "status",
			"data": gin.H{"status": status},
		}
	}

	if strings.HasPrefix(line, "ERROR:") {
		errMsg := strings.TrimPrefix(line, "ERROR:")
		session.Status = "error"
		session.Error = errMsg
		return map[string]interface{}{
			"type": "error",
			"data": gin.H{"message": errMsg},
		}
	}

	if strings.HasPrefix(line, "SUCCESS:") {
		parts := strings.Split(strings.TrimPrefix(line, "SUCCESS:"), "|")
		if len(parts) >= 2 {
			session.AppID = parts[0]
			session.BotName = parts[1]
		}
		return map[string]interface{}{
			"type": "success",
			"data": gin.H{"app_id": session.AppID, "bot_name": session.BotName},
		}
	}

	// Check for QR code pattern (ANSI art with ▄ and █ characters)
	if isQRCodeLine(line) {
		// Store QR code lines
		session.QRCode += line + "\n"
		return map[string]interface{}{
			"type": "qrcode",
			"data": gin.H{"line": line},
		}
	}

	// Check for status messages
	if strings.Contains(line, "等待扫码") {
		return map[string]interface{}{
			"type": "info",
			"data": gin.H{"message": "请使用飞书 APP 扫描二维码登录"},
		}
	}

	if strings.Contains(line, "正在刷新") {
		return map[string]interface{}{
			"type": "info",
			"data": gin.H{"message": "二维码已刷新，请重新扫描"},
		}
	}

	if strings.Contains(line, "创建成功") || strings.Contains(line, "应用创建成功") {
		session.Status = "creating"
		return map[string]interface{}{
			"type": "info",
			"data": gin.H{"message": "正在创建机器人应用..."},
		}
	}

	return nil
}

// isQRCodeLine checks if a line is part of ASCII QR code
func isQRCodeLine(line string) bool {
	// QR code lines contain ▄ or █ characters with ANSI color codes
	return strings.Contains(line, "▄") || strings.Contains(line, "█") || strings.Contains(line, "[47m") || strings.Contains(line, "[30m")
}

// parseCLISuccessOutput extracts app_id and app_secret from CLI output
func parseCLISuccessOutput(output string) (appID, appSecret, tenantKey string) {
	// Look for patterns like:
	// "App ID: cli_xxx"
	// "App Secret: xxx"
	// "Tenant Key: xxx"

	// Try multiple patterns
	appIDPattern := regexp.MustCompile(`App\s*ID[:\s]+(cli_[a-zA-Z0-9]+)`)
	appSecretPattern := regexp.MustCompile(`App\s*Secret[:\s]+([a-zA-Z0-9]+)`)
	tenantKeyPattern := regexp.MustCompile(`Tenant\s*Key[:\s]+([a-zA-Z0-9]+)`)

	// Also try JSON output pattern
	jsonPattern := regexp.MustCompile(`\{[^}]*"app_id"[^}]*\}`)

	if match := appIDPattern.FindStringSubmatch(output); len(match) > 1 {
		appID = match[1]
	}
	if match := appSecretPattern.FindStringSubmatch(output); len(match) > 1 {
		appSecret = match[1]
	}
	if match := tenantKeyPattern.FindStringSubmatch(output); len(match) > 1 {
		tenantKey = match[1]
	}

	// Try parsing as JSON if patterns didn't work
	if appID == "" {
		if match := jsonPattern.FindString(output); match != "" {
			// Parse JSON object
			var result map[string]string
			if err := json.Unmarshal([]byte(match), &result); err == nil {
				appID = result["app_id"]
				appSecret = result["app_secret"]
				tenantKey = result["tenant_key"]
			}
		}
	}

	return appID, appSecret, tenantKey
}

// findChromiumPath finds the chromium browser executable
func findChromiumPath() string {
	paths := []string{
		"/usr/bin/ungoogled-chromium",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/chrome",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Check environment variable
	if envPath := os.Getenv("CHROME_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	return ""
}

// cleanupSession removes a bind session
func cleanupSession(token string) {
	bindSessions.mu.Lock()
	if sess, ok := bindSessions.sessions[token]; ok {
		if sess.CancelFunc != nil {
			sess.CancelFunc()
		}
		if sess.Cmd != nil && sess.Cmd.Process != nil {
			sess.Cmd.Process.Kill()
		}
		delete(bindSessions.sessions, token)
	}
	bindSessions.mu.Unlock()
}

// writeSSE writes an SSE event
func writeSSE(c *gin.Context, eventType string, data gin.H) {
	c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", mustMarshal(gin.H{
		"type": eventType,
		"data": data,
	})))
}

// mustMarshal marshals JSON without error (for SSE)
func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// GetBindStatus checks the status of a binding operation by token (for polling fallback)
// GET /api/feishu/bind-status/:token
func GetBindStatus(c *gin.Context) {
	bindToken := c.Param("token")

	bindSessions.mu.RLock()
	session, ok := bindSessions.sessions[bindToken]
	bindSessions.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "绑定请求不存在或已过期"})
		return
	}

	// Check if token expired (5 minutes)
	if time.Since(session.CreatedAt) > 5*time.Minute {
		cleanupSession(bindToken)
		c.JSON(http.StatusNotFound, gin.H{"error": "绑定请求已过期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    session.Status,
		"app_id":    session.AppID,
		"bot_name":  session.BotName,
		"qrcode":    session.QRCode,
		"error":     session.Error,
	})
}

// CancelBind cancels an ongoing bind operation
// DELETE /api/feishu/bind/:token
func CancelBind(c *gin.Context) {
	bindToken := c.Param("token")
	cleanupSession(bindToken)
	c.JSON(http.StatusOK, gin.H{"message": "已取消"})
}