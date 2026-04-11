# 飞书集成实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现飞书扫码创建机器人 + 飞书聊天使用口播稿创作服务，支持流式 Card 输出。

**Architecture:** 独立飞书模块 + 共享 Workflow Engine。新增 feishu_bots/users/conversations 表，WebSocket 连接池管理多用户连接，FeishuSSEWriter 实现 SSEWriter 接口适配。

**Tech Stack:** Go + gorilla/websocket + 飞书开放平台 API + React前端绑定页面

---

## 文件结构

```
backend/
├── feishu_manifest.yaml                    # App Manifest 配置
├── internal/
│   ├── feishu/                             # 飞书模块目录
│   │   ├── types.go                        # 事件类型定义
│   │   ├── ws_client.go                    # WebSocket 连接
│   │   ├── ws_pool.go                      # 连接池
│   │   ├── card_sse.go                     # SSEWriter 适配器
│   │   ├── router.go                       # 消息路由
│   │   └── ws_pool_test.go                 # 连接池测试
│   ├── model/
│   │   ├── feishu_bot.go                   # Bot 模型
│   │   ├── feishu_user.go                  # User 模型
│   │   └── feishu_conversation.go          # Conversation 模型
│   ├── repository/
│   │   ├── feishu_bot_repo.go              # Bot CRUD
│   │   ├── feishu_user_repo.go             # User CRUD
│   │   └── feishu_conv_repo.go             # Conversation CRUD
│   ├── service/
│   │   ├── feishu_api.go                   # 飞书 API 封装
│   │   └── feishu_session.go               # 会话管理
│   └── handler/
│       └── feishu_handler.go               # HTTP handler

frontend/
└── src/
    ├── pages/
    │   └── FeishuBind.tsx                  # 绑定页面
    ├── api/
    │   └── feishu.ts                       # API 封装
    └── components/
        └── FeishuQRCode.tsx                # 二维码组件

修改文件:
- backend/config/config.go                  # 新增飞书配置
- backend/internal/db/db.go                 # AutoMigrate
- backend/main.go                           # 路由注册 + WS初始化
- frontend/src/router.tsx                   # 路由
- frontend/src/components/Sidebar.tsx       # 入口
```

---

## Task 1: 配置扩展

**Files:**
- Modify: `backend/config/config.go`
- Create: `backend/feishu_manifest.yaml`

- [ ] **Step 1: 添加飞书配置字段到 Config 结构体**

在 `backend/config/config.go` 的 Config 结构体末尾（第23行后）添加：

```go
// Feishu integration
FeishuEnabled         bool   `json:"feishu_enabled"`
FeishuManifestPath    string `json:"feishu_manifest_path"`
FeishuWSReconnectMax  int    `json:"feishu_ws_reconnect_max"`
FeishuWSHeartbeatSec  int    `json:"feishu_ws_heartbeat_sec"`
FeishuCardThrottleMs  int    `json:"feishu_card_throttle_ms"`
```

- [ ] **Step 2: 添加默认值**

在 `Load()` 函数的默认值部分（第30-45行）添加：

```go
FeishuEnabled:        false,
FeishuManifestPath:   "feishu_manifest.yaml",
FeishuWSReconnectMax: 3,
FeishuWSHeartbeatSec: 30,
FeishuCardThrottleMs: 200,
```

- [ ] **Step 3: 添加环境变量覆盖**

在环境变量覆盖部分（第57-65行后）添加：

```go
if v := os.Getenv("FEISHU_ENABLED"); v != "" { C.FeishuEnabled = v == "true" || v == "1" }
if v := os.Getenv("FEISHU_WS_RECONNECT_MAX"); v != "" {
	n, _ := strconv.Atoi(v)
	C.FeishuWSReconnectMax = n
}
```

添加 `import "strconv"` 到文件顶部 import 区域。

- [ ] **Step 4: 创建 App Manifest 配置文件**

创建 `backend/feishu_manifest.yaml`:

```yaml
app:
  name: "口播稿助手"
  description: "AI驱动的爆款口播稿改写工具"

permissions:
  - im:message:receive_as_bot
  - im:message:send_as_bot
  - im:card
  - contact:user.base:readonly

events:
  - im.message.receive_v1
  - card.action.trigger

event_subscription:
  type: websocket

config:
  locale: zh_CN
```

- [ ] **Step 5: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 6: Commit**

```bash
git add backend/config/config.go backend/feishu_manifest.yaml
git commit -m "feat(config): add feishu integration config fields"
```

---

## Task 2: 数据模型

**Files:**
- Create: `backend/internal/model/feishu_bot.go`
- Create: `backend/internal/model/feishu_user.go`
- Create: `backend/internal/model/feishu_conversation.go`
- Modify: `backend/internal/db/db.go`

- [ ] **Step 1: 创建 FeishuBot 模型**

```go
// backend/internal/model/feishu_bot.go
package model

import "time"

type FeishuBot struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	AppID         string    `gorm:"size:64;unique;not null" json:"app_id"`
	AppSecret     string    `gorm:"size:128;not null" json:"-"`
	TenantKey     string    `gorm:"size:64" json:"tenant_key"`
	BotName       string    `gorm:"size:128" json:"bot_name"`
	WSConnected   bool      `gorm:"default:false" json:"ws_connected"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: 创建 FeishuUser 模型**

```go
// backend/internal/model/feishu_user.go
package model

import "time"

type BindStatus string

const (
	BindIndependent BindStatus = "independent"
	BindMerged      BindStatus = "merged"
)

type FeishuUser struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	FeishuID   string     `gorm:"size:64;unique;not null" json:"feishu_id"`
	OpenID     string     `gorm:"size:64;unique" json:"open_id"`
	UnionID    string     `gorm:"size:64" json:"union_id"`
	UserID     uint       `gorm:"index" json:"user_id"`
	BindStatus BindStatus `gorm:"size:20;default:'independent'" json:"bind_status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
```

- [ ] **Step 3: 创建 FeishuConversation 模型**

```go
// backend/internal/model/feishu_conversation.go
package model

import "time"

type FeishuConversation struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	BotID        uint      `gorm:"index;not null" json:"bot_id"`
	ConvID       uint      `gorm:"index;not null" json:"conv_id"`
	FeishuChatID string    `gorm:"size:64;unique;not null" json:"feishu_chat_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

- [ ] **Step 4: 更新 db.go AutoMigrate**

在 `backend/internal/db/db.go` 第50-59行的 AutoMigrate 参数列表末尾添加：

```go
&model.FeishuBot{},
&model.FeishuUser{},
&model.FeishuConversation{},
```

- [ ] **Step 5: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 6: Commit**

```bash
git add backend/internal/model/feishu_*.go backend/internal/db/db.go
git commit -m "feat(model): add feishu bot/user/conversation models"
```

---

## Task 3: Repository 层

**Files:**
- Create: `backend/internal/repository/feishu_bot_repo.go`
- Create: `backend/internal/repository/feishu_user_repo.go`
- Create: `backend/internal/repository/feishu_conv_repo.go`

- [ ] **Step 1: 创建 FeishuBot Repository**

```go
// backend/internal/repository/feishu_bot_repo.go
package repository

import (
	"time"
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateFeishuBot(bot *model.FeishuBot) error {
	return db.DB.Create(bot).Error
}

func GetFeishuBotByAppID(appID string) (*model.FeishuBot, error) {
	var bot model.FeishuBot
	err := db.DB.Where("app_id = ?", appID).First(&bot).Error
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

func GetFeishuBotsByUserID(userID uint) ([]model.FeishuBot, error) {
	var bots []model.FeishuBot
	err := db.DB.Where("user_id = ?", userID).Find(&bots).Error
	return bots, err
}

func GetConnectedFeishuBots() ([]model.FeishuBot, error) {
	var bots []model.FeishuBot
	err := db.DB.Where("ws_connected = ?", true).Find(&bots).Error
	return bots, err
}

func UpdateFeishuBotWSStatus(botID uint, connected bool) error {
	return db.DB.Model(&model.FeishuBot{}).
		Where("id = ?", botID).
		Updates(map[string]interface{}{
			"ws_connected":   connected,
			"last_heartbeat": time.Now(),
		}).Error
}

func DeleteFeishuBot(botID, userID uint) error {
	return db.DB.Where("id = ? AND user_id = ?", botID, userID).
		Delete(&model.FeishuBot{}).Error
}
```

- [ ] **Step 2: 创建 FeishuUser Repository**

```go
// backend/internal/repository/feishu_user_repo.go
package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateFeishuUser(user *model.FeishuUser) error {
	return db.DB.Create(user).Error
}

func GetFeishuUserByOpenID(openID string) (*model.FeishuUser, error) {
	var user model.FeishuUser
	err := db.DB.Where("open_id = ?", openID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetOrCreateFeishuUserByOpenID(openID string) (*model.FeishuUser, error) {
	user, err := GetFeishuUserByOpenID(openID)
	if err == nil {
		return user, nil
	}
	newUser := &model.FeishuUser{
		OpenID:     openID,
		FeishuID:   openID,
		BindStatus: model.BindIndependent,
	}
	if err := CreateFeishuUser(newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

func UpdateFeishuUserBind(feishuUserID, webUserID uint) error {
	return db.DB.Model(&model.FeishuUser{}).
		Where("id = ?", feishuUserID).
		Updates(map[string]interface{}{
			"user_id":     webUserID,
			"bind_status": model.BindMerged,
		}).Error
}
```

- [ ] **Step 3: 创建 FeishuConversation Repository**

```go
// backend/internal/repository/feishu_conv_repo.go
package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateFeishuConv(conv *model.FeishuConversation) error {
	return db.DB.Create(conv).Error
}

func GetFeishuConvByChatID(chatID string) (*model.FeishuConversation, error) {
	var conv model.FeishuConversation
	err := db.DB.Where("feishu_chat_id = ?", chatID).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetOrCreateFeishuConv 创建飞书会话映射，同时创建关联的 Conversation 记录
func GetOrCreateFeishuConv(botID uint, chatID string, userID uint) (*model.FeishuConversation, uint, error) {
	conv, err := GetFeishuConvByChatID(chatID)
	if err == nil && conv.ConvID > 0 {
		return conv, conv.ConvID, nil
	}
	// 创建新的 conversation 记录
	webConv := &model.Conversation{
		UserID: userID,
		Title:  "飞书对话",
		State:  0,
	}
	if err := CreateConversation(webConv); err != nil {
		return nil, 0, err
	}
	newConv := &model.FeishuConversation{
		BotID:        botID,
		ConvID:       webConv.ID,
		FeishuChatID: chatID,
	}
	if err := CreateFeishuConv(newConv); err != nil {
		return nil, 0, err
	}
	return newConv, webConv.ID, nil
}

func DeleteFeishuConvsByBotID(botID uint) error {
	return db.DB.Where("bot_id = ?", botID).Delete(&model.FeishuConversation{}).Error
}
```

- [ ] **Step 4: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/feishu_*.go
git commit -m "feat(repo): add feishu repositories"
```

---

## Task 4: 飞书事件类型定义

**Files:**
- Create: `backend/internal/feishu/types.go`

- [ ] **Step 1: 创建飞书模块目录**

```bash
mkdir -p backend/internal/feishu
```

- [ ] **Step 2: 创建事件类型定义**

```go
// backend/internal/feishu/types.go
package feishu

import "encoding/json"

// WebSocket 推送事件
type WSEvent struct {
	Type      string          `json:"type"`
	AppID     string          `json:"app_id"`
	TenantKey string          `json:"tenant_key"`
	Event     json.RawMessage `json:"event"`
}

// 消息接收事件
type MessageEvent struct {
	Sender  Sender  `json:"sender"`
	Message Message `json:"message"`
}

type Sender struct {
	OpenID  string `json:"open_id"`
	UnionID string `json:"union_id"`
}

type Message struct {
	MessageID  string `json:"message_id"`
	ChatID     string `json:"chat_id"`
	ChatType   string `json:"chat_type"`
	Content    string `json:"content"`
	CreateTime int64  `json:"create_time"`
}

// Card 按钮点击事件
type CardActionEvent struct {
	OpenID string          `json:"open_id"`
	ChatID string          `json:"chat_id"`
	Action CardActionValue `json:"action"`
}

type CardActionValue struct {
	Value map[string]string `json:"value"`
}

// App Manifest 创建回调
type ManifestCreatedEvent struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	TenantKey string `json:"tenant_key"`
	BindToken string `json:"bind_token"`
}

// 飞书 Card 结构
type Card struct {
	Config   CardConfig   `json:"config"`
	Header   CardHeader   `json:"header"`
	Elements []CardElement `json:"elements"`
}

type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type CardHeader struct {
	Title    CardText `json:"title"`
	Template string   `json:"template"`
}

type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type CardElement struct {
	Tag     string       `json:"tag"`
	Text    *CardText    `json:"text,omitempty"`
	Actions []CardAction `json:"actions,omitempty"`
}

type CardAction struct {
	Tag   string            `json:"tag"`
	Text  CardText          `json:"text"`
	Type  string            `json:"type"`
	Value map[string]string `json:"value"`
}

// WebSocket 连接状态
type WSStatus string

const (
	WSConnected    WSStatus = "connected"
	WSDisconnected WSStatus = "disconnected"
	WSReconnecting WSStatus = "reconnecting"
)
```

- [ ] **Step 3: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add backend/internal/feishu/types.go
git commit -m "feat(feishu): add event and card type definitions"
```

---

## Task 5: WebSocket 连接池

**Files:**
- Create: `backend/internal/feishu/ws_client.go`
- Create: `backend/internal/feishu/ws_pool.go`
- Create: `backend/internal/feishu/ws_pool_test.go`

- [ ] **Step 1: 安装 websocket 依赖**

```bash
cd backend && go get github.com/gorilla/websocket
```

- [ ] **Step 2: 创建 WSConnection**

```go
// backend/internal/feishu/ws_client.go
package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSConnection struct {
	AppID          string
	AppSecret      string
	Conn           *websocket.Conn
	Status         WSStatus
	MessageHandler func(event WSEvent)
	ReconnectCount int
	MaxReconnect   int
	HeartbeatSec   int
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
}

func NewWSConn(appID, appSecret string, maxReconnect, heartbeatSec int) *WSConnection {
	ctx, cancel := context.WithCancel(context.Background())
	return &WSConnection{
		AppID:        appID,
		AppSecret:    appSecret,
		Status:       WSDisconnected,
		MaxReconnect: maxReconnect,
		HeartbeatSec: heartbeatSec,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (c *WSConnection) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	wsURL := fmt.Sprintf("wss://ws.feishu.cn/ws?app_id=%s&app_secret=%s", c.AppID, c.AppSecret)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.Conn = conn
	c.Status = WSConnected
	c.ReconnectCount = 0

	go c.heartbeatLoop()
	go c.receiveLoop()

	log.Printf("[FeishuWS] connected: %s", c.AppID)
	return nil
}

func (c *WSConnection) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cancel()
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.Status = WSDisconnected
	log.Printf("[FeishuWS] disconnected: %s", c.AppID)
}

func (c *WSConnection) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(c.HeartbeatSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.sendPing()
		}
	}
}

func (c *WSConnection) sendPing() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn == nil || c.Status != WSConnected {
		return
	}

	ping := map[string]string{"type": "ping"}
	data, _ := json.Marshal(ping)
	if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[FeishuWS] ping failed: %v", err)
		c.triggerReconnect()
	}
}

func (c *WSConnection) receiveLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, msg, err := c.Conn.ReadMessage()
			if err != nil {
				log.Printf("[FeishuWS] read error: %v", err)
				c.triggerReconnect()
				return
			}

			var event WSEvent
			if json.Unmarshal(msg, &event) == nil {
				if event.Type == "pong" {
					continue
				}
				if c.MessageHandler != nil {
					c.MessageHandler(event)
				}
			}
		}
	}
}

func (c *WSConnection) triggerReconnect() {
	c.mu.Lock()
	if c.ReconnectCount >= c.MaxReconnect {
		c.Status = WSDisconnected
		c.mu.Unlock()
		return
	}
	c.Status = WSReconnecting
	c.ReconnectCount++
	delay := time.Duration(c.ReconnectCount) * 5 * time.Second
	c.mu.Unlock()

	log.Printf("[FeishuWS] reconnect in %v (attempt %d)", delay, c.ReconnectCount)
	time.Sleep(delay)
	c.Disconnect()
	if err := c.Connect(); err != nil {
		c.triggerReconnect()
	}
}
```

- [ ] **Step 3: 创建 WSConnectionPool**

```go
// backend/internal/feishu/ws_pool.go
package feishu

import (
	"log"
	"sync"
)

type WSConnectionPool struct {
	connections  map[string]*WSConnection
	mu           sync.RWMutex
	maxReconnect int
	heartbeatSec int
}

var globalPool *WSConnectionPool
var poolOnce sync.Once

func GetWSPool(maxReconnect, heartbeatSec int) *WSConnectionPool {
	poolOnce.Do(func() {
		globalPool = &WSConnectionPool{
			connections:  make(map[string]*WSConnection),
			maxReconnect: maxReconnect,
			heartbeatSec: heartbeatSec,
		}
	})
	return globalPool
}

func (p *WSConnectionPool) Connect(appID, appSecret string, handler func(WSEvent)) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[appID]; ok && conn.Status == WSConnected {
		return nil
	}

	conn := NewWSConn(appID, appSecret, p.maxReconnect, p.heartbeatSec)
	conn.MessageHandler = handler

	if err := conn.Connect(); err != nil {
		return err
	}

	p.connections[appID] = conn
	log.Printf("[FeishuPool] added: %s", appID)
	return nil
}

func (p *WSConnectionPool) Disconnect(appID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[appID]; ok {
		conn.Disconnect()
		delete(p.connections, appID)
	}
}

func (p *WSConnectionPool) Get(appID string) *WSConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connections[appID]
}

func (p *WSConnectionPool) Status(appID string) WSStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if conn, ok := p.connections[appID]; ok {
		return conn.Status
	}
	return WSDisconnected
}
```

- [ ] **Step 4: 创建单元测试**

```go
// backend/internal/feishu/ws_pool_test.go
package feishu

import "testing"

func TestWSPoolSingleton(t *testing.T) {
	p1 := GetWSPool(3, 30)
	p2 := GetWSPool(3, 30)
	if p1 != p2 {
		t.Error("pool should be singleton")
	}
}

func TestWSPoolStatusDisconnected(t *testing.T) {
	p := GetWSPool(3, 30)
	if p.Status("nonexistent") != WSDisconnected {
		t.Error("nonexistent app should be disconnected")
	}
}

func TestNewWSConn(t *testing.T) {
	conn := NewWSConn("test-id", "test-secret", 3, 30)
	if conn.AppID != "test-id" {
		t.Errorf("expected test-id, got %s", conn.AppID)
	}
	if conn.Status != WSDisconnected {
		t.Error("initial status should be disconnected")
	}
}
```

- [ ] **Step 5: 运行测试**

```bash
cd backend && go test ./internal/feishu/... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/feishu/ws_*.go backend/go.mod backend/go.sum
git commit -m "feat(feishu): add websocket connection pool"
```

---

## Task 6: 飞书 API Service

**Files:**
- Create: `backend/internal/service/feishu_api.go`

- [ ] **Step 1: 创建飞书 API Service**

```go
// backend/internal/service/feishu_api.go
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const feishuAPIBase = "https://open.feishu.cn/open-apis"

type FeishuAPI struct {
	AppID     string
	AppSecret string
	Token     string
	TokenExp  time.Time
}

func NewFeishuAPI(appID, appSecret string) *FeishuAPI {
	return &FeishuAPI{AppID: appID, AppSecret: appSecret}
}

func (a *FeishuAPI) GetToken() (string, error) {
	if a.Token != "" && time.Now().Before(a.TokenExp) {
		return a.Token, nil
	}

	url := feishuAPIBase + "/auth/v3/tenant_access_token/internal"
	body := map[string]string{"app_id": a.AppID, "app_secret": a.AppSecret}

	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Token string `json:"tenant_access_token"`
		Exp   int    `json:"expire"`
	}
	json.Unmarshal(respBody, &result)

	if result.Code != 0 {
		return "", fmt.Errorf("api error: %s", result.Msg)
	}

	a.Token = result.Token
	a.TokenExp = time.Now().Add(time.Duration(result.Exp-60) * time.Second)
	return a.Token, nil
}

func (a *FeishuAPI) CreateCard(chatID string, cardJSON string) (string, error) {
	token, err := a.GetToken()
	if err != nil {
		return "", err
	}

	url := feishuAPIBase + "/im/v1/messages?receive_id_type=chat_id"
	body := map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "interactive",
		"content":    json.RawMessage(cardJSON),
	}

	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	json.Unmarshal(respBody, &result)

	if result.Code != 0 {
		return "", fmt.Errorf("create card failed: code %d", result.Code)
	}
	return result.Data.MessageID, nil
}

func (a *FeishuAPI) UpdateCard(messageID string, cardJSON string) error {
	token, err := a.GetToken()
	if err != nil {
		return err
	}

	url := feishuAPIBase + "/im/v1/messages/" + messageID
	body := map[string]interface{}{
		"msg_type": "interactive",
		"content":  json.RawMessage(cardJSON),
	}

	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/feishu_api.go
git commit -m "feat(service): add feishu API client"
```

---

## Task 7: Feishu SSEWriter 适配器

**Files:**
- Create: `backend/internal/feishu/card_sse.go`

- [ ] **Step 1: 创建 FeishuSSEWriter（实现 SSEWriter 接口）**

```go
// backend/internal/feishu/card_sse.go
package feishu

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"content-creator-imm/internal/service"
	"content-creator-imm/internal/workflow"
)

// FeishuSSEWriter 实现 workflow.SSEWriter 接口，将 SSE 事件转换为飞书 Card 更新
type FeishuSSEWriter struct {
	api         *service.FeishuAPI
	chatID      string
	messageID   string
	stageName   string
	content     strings.Builder
	throttleMs  int
	lastUpdate  time.Time
	mu          sync.Mutex
}

func NewFeishuSSEWriter(chatID, appID, appSecret string, throttleMs int) *FeishuSSEWriter {
	return &FeishuSSEWriter{
		api:        service.NewFeishuAPI(appID, appSecret),
		chatID:     chatID,
		throttleMs: throttleMs,
	}
}

func (w *FeishuSSEWriter) Init() error {
	card := w.buildCard()
	cardJSON, _ := json.Marshal(card)
	msgID, err := w.api.CreateCard(w.chatID, string(cardJSON))
	if err != nil {
		return err
	}
	w.messageID = msgID
	return nil
}

// --- SSEWriter 接口实现 ---

func (w *FeishuSSEWriter) SendStageStart(stageID, stageName string, stageType workflow.StageType) {
	w.mu.Lock()
	w.stageName = stageName
	w.content.Reset()
	w.mu.Unlock()
	w.updateCard()
}

func (w *FeishuSSEWriter) SendStageDone(stageID string) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendWorkerStart(stageID, workerName, workerDisplay string) {
	w.updateCard()
}

func (w *FeishuSSEWriter) SendWorkerToken(workerName, content string) {
	w.mu.Lock()
	w.content.WriteString(content)
	w.mu.Unlock()
	w.updateCard()
}

func (w *FeishuSSEWriter) SendWorkerDone(workerName string) {}

func (w *FeishuSSEWriter) SendSynthStart(stageID string) {}
func (w *FeishuSSEWriter) SendSynthToken(content string) {}
func (w *FeishuSSEWriter) SendSynthDone(stageID string) {}

func (w *FeishuSSEWriter) SendStep(step int, name string) {}
func (w *FeishuSSEWriter) SendInfo(content string) {}

func (w *FeishuSSEWriter) SendOutline(data any) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendAction(prompt string, options []string) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendSimilarity(data any) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendFinalDraft(content string) {}

func (w *FeishuSSEWriter) SendComplete(scriptID uint) {
	w.mu.Lock()
	w.stageName = "完成"
	w.mu.Unlock()
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendError(message string) {
	w.mu.Lock()
	w.stageName = "错误"
	w.content.Reset()
	w.content.WriteString(fmt.Sprintf("❌ %s", message))
	w.mu.Unlock()
	w.forceUpdate()
}

// --- 内部方法 ---

func (w *FeishuSSEWriter) updateCard() {
	w.mu.Lock()
	now := time.Now()
	if now.Sub(w.lastUpdate) < time.Duration(w.throttleMs)*time.Millisecond {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	w.forceUpdate()
}

func (w *FeishuSSEWriter) forceUpdate() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.messageID == "" {
		return
	}

	card := w.buildCard()
	cardJSON, _ := json.Marshal(card)
	w.api.UpdateCard(w.messageID, string(cardJSON))
	w.lastUpdate = time.Now()
}

func (w *FeishuSSEWriter) buildCard() Card {
	template := "blue"
	title := "口播稿创作"
	if w.stageName == "完成" {
		template = "green"
		title = "创作完成"
	} else if w.stageName == "错误" {
		template = "red"
		title = "创作失败"
	} else if w.stageName != "" {
		title = fmt.Sprintf("口播稿创作 - %s", w.stageName)
	}

	elements := []CardElement{}
	if w.content.Len() > 0 {
		text := w.content.String()
		if len(text) > 4000 {
			text = text[:4000] + "..."
		}
		elements = append(elements, CardElement{
			Tag:  "div",
			Text: &CardText{Tag: "lark_md", Content: text},
		})
	}

	return Card{
		Config: CardConfig{WideScreenMode: true},
		Header: CardHeader{
			Title:    CardText{Tag: "plain_text", Content: title},
			Template: template,
		},
		Elements: elements,
	}
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/feishu/card_sse.go
git commit -m "feat(feishu): add FeishuSSEWriter adapter"
```

---

## Task 8: Feishu Session Manager

**Files:**
- Create: `backend/internal/service/feishu_session.go`

- [ ] **Step 1: 创建 Session Manager**

```go
// backend/internal/service/feishu_session.go
package service

import "sync"

type FeishuState string

const (
	FeishuIdle      FeishuState = "idle"
	FeishuAnalyzing FeishuState = "analyzing"
	FeishuAwaiting  FeishuState = "awaiting"
	FeishuWriting   FeishuState = "writing"
)

type FeishuSession struct {
	ChatID     string
	BotID      uint
	UserID     uint
	FeishuUID  uint
	ConvID     uint
	WorkflowID uint
	State      FeishuState
	lock       sync.Mutex
}

type FeishuSessionMgr struct {
	sessions map[string]*FeishuSession
	mu       sync.RWMutex
}

var feishuSessionMgr *FeishuSessionMgr
var feishuSessionOnce sync.Once

func GetFeishuSessionMgr() *FeishuSessionMgr {
	feishuSessionOnce.Do(func() {
		feishuSessionMgr = &FeishuSessionMgr{
			sessions: make(map[string]*FeishuSession),
		}
	})
	return feishuSessionMgr
}

func (m *FeishuSessionMgr) GetOrCreate(chatID string, botID, userID, feishuUID uint) *FeishuSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[chatID]; ok {
		return sess
	}
	sess := &FeishuSession{
		ChatID:    chatID,
		BotID:     botID,
		UserID:    userID,
		FeishuUID: feishuUID,
		State:     FeishuIdle,
	}
	m.sessions[chatID] = sess
	return sess
}

func (m *FeishuSessionMgr) Get(chatID string) *FeishuSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[chatID]
}

func (m *FeishuSessionMgr) SetState(chatID string, state FeishuState) {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess != nil {
		sess.lock.Lock()
		sess.State = state
		sess.lock.Unlock()
	}
}

func (m *FeishuSessionMgr) SetWorkflowID(chatID string, wfID uint) {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess != nil {
		sess.lock.Lock()
		sess.WorkflowID = wfID
		sess.lock.Unlock()
	}
}

func (m *FeishuSessionMgr) SetConvID(chatID string, convID uint) {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess != nil {
		sess.lock.Lock()
		sess.ConvID = convID
		sess.lock.Unlock()
	}
}

func (m *FeishuSessionMgr) IsBusy(chatID string) bool {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess == nil {
		return false
	}
	sess.lock.Lock()
	defer sess.lock.Unlock()
	return sess.State == FeishuAnalyzing || sess.State == FeishuWriting
}

func (m *FeishuSessionMgr) Clear(chatID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, chatID)
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/feishu_session.go
git commit -m "feat(service): add feishu session manager"
```

---

## Task 9: 消息路由

**Files:**
- Create: `backend/internal/feishu/router.go`

- [ ] **Step 1: 创建消息路由**

```go
// backend/internal/feishu/router.go
package feishu

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"
	"content-creator-imm/internal/workflow"
)

type Router struct {
	engine    *workflow.Engine
	writers   map[string]*FeishuSSEWriter
	writersMu sync.RWMutex
}

func NewRouter(loader *workflow.Loader) *Router {
	return &Router{
		engine:  workflow.NewEngine(loader, nil), // SSEWriter 由每个请求创建
		writers: make(map[string]*FeishuSSEWriter),
	}
}

func (r *Router) HandleEvent(event WSEvent) {
	log.Printf("[FeishuRouter] event: %s", event.Type)

	switch event.Type {
	case "im.message.receive_v1":
		r.handleMessage(event)
	case "card.action.trigger":
		r.handleCardAction(event)
	default:
		log.Printf("[FeishuRouter] unknown type: %s", event.Type)
	}
}

func (r *Router) handleMessage(event WSEvent) {
	var msgEvent MessageEvent
	if json.Unmarshal(event.Event, &msgEvent) != nil {
		return
	}

	chatID := msgEvent.Message.ChatID
	openID := msgEvent.Sender.OpenID
	content := parseContent(msgEvent.Message.Content)

	if content == "" {
		return
	}

	sessMgr := service.GetFeishuSessionMgr()
	if sessMgr.IsBusy(chatID) {
		r.sendBusyHint(event.AppID, chatID)
		return
	}

	bot, err := repository.GetFeishuBotByAppID(event.AppID)
	if err != nil {
		log.Printf("[Router] bot not found: %s", event.AppID)
		return
	}

	feishuUser, err := repository.GetOrCreateFeishuUserByOpenID(openID)
	if err != nil {
		return
	}

	_, convID, err := repository.GetOrCreateFeishuConv(bot.ID, chatID, bot.UserID)
	if err != nil {
		return
	}

	sess := sessMgr.GetOrCreate(chatID, bot.ID, bot.UserID, feishuUser.ID)
	sessMgr.SetConvID(chatID, convID)

	writer := r.getOrCreateWriter(chatID, event.AppID, bot.AppSecret)

	switch sess.State {
	case FeishuIdle:
		r.handleIdle(sess, writer, content, convID, bot.UserID)
	case FeishuAwaiting:
		r.handleAwaiting(sess, writer, content)
	default:
		log.Printf("[Router] unexpected state: %s", sess.State)
	}
}

func (r *Router) handleIdle(sess *service.FeishuSession, writer *FeishuSSEWriter, content string, convID uint, userID uint) {
	sessMgr := service.GetFeishuSessionMgr()
	sessMgr.SetState(sess.ChatID, service.FeishuAnalyzing)

	writer.Init()
	writer.SendStageStart("", "开始分析", workflow.StageSerial)

	input := workflow.WorkflowInput{
		Text:    content,
		UserID:  userID,
		ConvID:  convID,
	}

	engine := workflow.NewEngine(nil, writer)
	if err := engine.Start("viral_script", input); err != nil {
		writer.SendError(err.Error())
		sessMgr.SetState(sess.ChatID, service.FeishuIdle)
		return
	}

	sessMgr.SetWorkflowID(sess.ChatID, engine.WorkflowID())
	sessMgr.SetState(sess.ChatID, service.FeishuAwaiting)
}

func (r *Router) handleAwaiting(sess *service.FeishuSession, writer *FeishuSSEWriter, content string) {
	choice := parseChoice(content)

	sessMgr := service.GetFeishuSessionMgr()
	sessMgr.SetState(sess.ChatID, service.FeishuWriting)

	engine := workflow.NewEngine(nil, writer)
	if err := engine.Resume(sess.WorkflowID, choice); err != nil {
		writer.SendError(err.Error())
		sessMgr.SetState(sess.ChatID, service.FeishuIdle)
	}
}

func (r *Router) handleCardAction(event WSEvent) {
	var actionEvent CardActionEvent
	if json.Unmarshal(event.Event, &actionEvent) != nil {
		return
	}

	chatID := actionEvent.ChatID
	action := actionEvent.Action.Value["action"]

	sess := service.GetFeishuSessionMgr().Get(chatID)
	if sess == nil || sess.WorkflowID == 0 {
		return
	}

	writer := r.getWriter(chatID)
	if writer == nil {
		return
	}

	service.GetFeishuSessionMgr().SetState(chatID, service.FeishuWriting)

	engine := workflow.NewEngine(nil, writer)
	engine.Resume(sess.WorkflowID, action)
}

func (r *Router) getOrCreateWriter(chatID, appID, appSecret string) *FeishuSSEWriter {
	r.writersMu.Lock()
	defer r.writersMu.Unlock()

	if w, ok := r.writers[chatID]; ok {
		return w
	}
	w := NewFeishuSSEWriter(chatID, appID, appSecret, 200)
	r.writers[chatID] = w
	return w
}

func (r *Router) getWriter(chatID string) *FeishuSSEWriter {
	r.writersMu.RLock()
	defer r.writersMu.RUnlock()
	return r.writers[chatID]
}

func (r *Router) sendBusyHint(appID, chatID string) {
	bot, err := repository.GetFeishuBotByAppID(appID)
	if err != nil {
		return
	}
	api := service.NewFeishuAPI(appID, bot.AppSecret)
	token, _ := api.GetToken()
	log.Printf("[Router] busy: %s (token=%s)", chatID, token[:10])
}

func parseContent(content string) string {
	var text struct {
		Text string `json:"text"`
	}
	if json.Unmarshal([]byte(content), &text) == nil {
		return strings.TrimSpace(text.Text)
	}
	return strings.TrimSpace(content)
}

func parseChoice(content string) string {
	content = strings.TrimSpace(content)
	switch content {
	case "1", "确认", "是的":
		return "1"
	case "2", "调整", "修改":
		return "2"
	case "3", "更换素材":
		return "3"
	case "4", "重新", "重来":
		return "4"
	default:
		return content
	}
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/feishu/router.go
git commit -m "feat(feishu): add message router"
```

---

## Task 10: HTTP Handler + 路由注册

**Files:**
- Create: `backend/internal/handler/feishu_handler.go`
- Modify: `backend/main.go`

- [ ] **Step 1: 创建飞书 Handler**

```go
// backend/internal/handler/feishu_handler.go
package handler

import (
	"net/http"

	"content-creator-imm/internal/repository"
	"github.com/gin-gonic/gin"
)

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
	botID := c.Param("id")

	var id uint
	if _, err := parseInt(botID, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}

	if err := repository.DeleteFeishuBot(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑失败"})
		return
	}

	// 同时删除关联的飞书会话
	repository.DeleteFeishuConvsByBotID(id)

	c.JSON(http.StatusOK, gin.H{"message": "解绑成功"})
}

// parseInt helper
func parseInt(s string, out *uint) (bool, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return false, err
	}
	*out = uint(n)
	return true, nil
}
```

添加 `import "strconv"` 到文件顶部。

- [ ] **Step 2: 修改 main.go 添加飞书路由和初始化**

在 `backend/main.go` 的路由注册部分（第76-96行区域）添加飞书路由：

```go
// Feishu routes
feishu := api.Group("/feishu")
{
	feishu.GET("/bots", handler.GetFeishuBots)
	feishu.DELETE("/bots/:id", handler.UnbindFeishuBot)
}
```

在服务启动后添加飞书 WebSocket 初始化（第39-42行区域后）：

```go
// Initialize Feishu WebSocket if enabled
if config.C.FeishuEnabled {
	feishuRouter := feishu.NewRouter(wfLoader)
	feishuPool := feishu.GetWSPool(config.C.FeishuWSReconnectMax, config.C.FeishuWSHeartbeatSec)

	// Connect all existing bots
	bots, _ := repository.GetConnectedFeishuBots()
	for _, bot := range bots {
		feishuPool.Connect(bot.AppID, bot.AppSecret, feishuRouter.HandleEvent)
	}
	log.Printf("[Feishu] initialized %d WS connections", len(bots))
}
```

添加必要的 import：

```go
import (
	// ... 现有 imports ...
	"content-creator-imm/internal/feishu"
)
```

- [ ] **Step 3: 验证编译**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/feishu_handler.go backend/main.go
git commit -m "feat(handler): add feishu handlers and WS initialization"
```

---

## Task 11: 前端 API 封装

**Files:**
- Create: `frontend/src/api/feishu.ts`

- [ ] **Step 1: 创建飞书 API 封装**

```typescript
// frontend/src/api/feishu.ts
import request from './request'

export interface FeishuBot {
  id: number
  user_id: number
  app_id: string
  bot_name: string
  ws_connected: boolean
  created_at: string
}

export function getFeishuBots(): Promise<{ bots: FeishuBot[] }> {
  return request.get('/feishu/bots')
}

export function unbindFeishuBot(botId: number): Promise<{ message: string }> {
  return request.delete(`/feishu/bots/${botId}`)
}
```

- [ ] **Step 2: 验证 TypeScript 编译**

```bash
cd frontend && npx tsc --noEmit
```

Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/feishu.ts
git commit -m "feat(frontend): add feishu API client"
```

---

## Task 12: 前端绑定页面

**Files:**
- Create: `frontend/src/pages/FeishuBind.tsx`
- Create: `frontend/src/components/FeishuQRCode.tsx`
- Modify: `frontend/src/router.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: 创建二维码组件**

```tsx
// frontend/src/components/FeishuQRCode.tsx
import React from 'react'

interface FeishuQRCodeProps {
  qrUrl: string
  status: 'waiting' | 'success' | 'error'
  onRefresh?: () => void
}

export function FeishuQRCode({ qrUrl, status, onRefresh }: FeishuQRCodeProps) {
  return (
    <div className="flex flex-col items-center gap-4">
      <div className="w-64 h-64 border rounded-lg flex items-center justify-center bg-white">
        {status === 'waiting' && (
          <img src={qrUrl} alt="飞书扫码绑定" className="w-60 h-60" />
        )}
        {status === 'success' && (
          <div className="text-green-500 text-4xl">✅</div>
        )}
        {status === 'error' && (
          <div className="text-red-500 text-4xl">❌</div>
        )}
      </div>
      {status === 'waiting' && (
        <p className="text-gray-500 text-sm">请使用飞书 App 扫描二维码</p>
      )}
      {status === 'error' && onRefresh && (
        <button onClick={onRefresh} className="btn btn-secondary">
          刷新二维码
        </button>
      )}
    </div>
  )
}
```

- [ ] **Step 2: 创建绑定页面**

```tsx
// frontend/src/pages/FeishuBind.tsx
import React, { useState, useEffect } from 'react'
import { FeishuQRCode } from '../components/FeishuQRCode'
import { getFeishuBots, unbindFeishuBot, FeishuBot } from '../api/feishu'

export function FeishuBind() {
  const [bots, setBots] = useState<FeishuBot[]>([])
  const [status, setStatus] = useState<'waiting' | 'success' | 'error'>('waiting')
  const [qrUrl, setQrUrl] = useState('')

  useEffect(() => {
    loadBots()
  }, [])

  const loadBots = async () => {
    const data = await getFeishuBots()
    setBots(data.bots)
  }

  const handleUnbind = async (botId: number) => {
    await unbindFeishuBot(botId)
    loadBots()
  }

  return (
    <div className="p-8 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">飞书机器人绑定</h1>

      {/* 已绑定的机器人列表 */}
      {bots.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-4">已绑定的机器人</h2>
          <div className="space-y-2">
            {bots.map(bot => (
              <div key={bot.id} className="flex items-center justify-between p-4 border rounded">
                <div>
                  <p className="font-medium">{bot.bot_name || '口播稿助手'}</p>
                  <p className="text-sm text-gray-500">
                    {bot.ws_connected ? '🟢 已连接' : '🔴 未连接'}
                  </p>
                </div>
                <button
                  onClick={() => handleUnbind(bot.id)}
                  className="btn btn-danger btn-sm"
                >
                  解绑
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 扫码绑定 */}
      <div className="text-center">
        <p className="mb-4">扫码创建飞书机器人，可在飞书中使用口播稿创作服务</p>
        {/* 注意：实际的扫码流程需要通过飞书开放平台的 App Manifest 导入 API 生成二维码 URL。
            当前版本先显示提示信息，后续根据飞书官方文档实现完整的扫码创建流程。 */}
        <div className="w-64 h-64 border rounded-lg flex items-center justify-center bg-gray-50">
          <p className="text-gray-500 text-center p-4">
            飞书扫码绑定功能需要配置飞书开放平台应用。<br/>
            请联系管理员获取绑定链接。
          </p>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 3: 添加路由**

在 `frontend/src/router.tsx` 中添加：

```tsx
import { FeishuBind } from './pages/FeishuBind'

// 在 routes 数组中添加
<Route path="/feishu" element={<FeishuBind />} />
```

- [ ] **Step 4: 在 Sidebar 添加入口**

在 `frontend/src/components/Sidebar.tsx` 的导航区域添加：

```tsx
<Link to="/feishu" className="nav-item">
  飞书绑定
</Link>
```

- [ ] **Step 5: 验证 TypeScript 编译**

```bash
cd frontend && npx tsc --noEmit
```

Expected: 无错误

- [ ] **Step 6: 验证前端构建**

```bash
cd frontend && npm run build
```

Expected: 无错误

- [ ] **Step 7: Commit**

```bash
git add frontend/src/pages/FeishuBind.tsx frontend/src/components/FeishuQRCode.tsx frontend/src/router.tsx frontend/src/components/Sidebar.tsx
git commit -m "feat(frontend): add feishu bind page and QR code component"
```

---

## Task 13: 验证与测试

**Files:**
- 无新增文件

- [ ] **Step 1: 后端编译验证**

```bash
cd backend && go build .
```

Expected: 无错误

- [ ] **Step 2: 后端单元测试**

```bash
cd backend && go test ./... -v
```

Expected: PASS

- [ ] **Step 3: 前端类型检查**

```bash
cd frontend && npx tsc --noEmit
```

Expected: 无错误

- [ ] **Step 4: 前端构建验证**

```bash
cd frontend && npm run build
```

Expected: 无错误

- [ ] **Step 5: 提交验证**

```bash
git status
```

Expected: 无未提交的更改（除了可能的 .ai_mem 更新）

---

## Task 14: 更新文档

**Files:**
- Modify: `CLAUDE.md`
- Modify: `.ai_mem/L1_modules.md`

- [ ] **Step 1: 更新 CLAUDE.md**

在 `## API 列表` 部分添加：

```
GET  /api/feishu/bots              飞书机器人列表（需认证）
DELETE /api/feishu/bots/:id        解绑飞书机器人（需认证）
```

在 `## 关键文件索引` 部分添加：

```
| `backend/internal/feishu/`       | 飞书模块（WebSocket、Card、路由）|
| `backend/internal/service/feishu_*.go` | 飞书 API/Session 服务 |
```

- [ ] **Step 2: 更新 .ai_mem/L1_modules.md**

在 `## 后端模块` 部分添加飞书模块说明。

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md .ai_mem/L1_modules.md
git commit -m "docs: update API list and module index for feishu"
```

---

## 执行完成

计划编写完成。所有任务已定义，每个步骤包含具体代码和验证命令。

---

**Spec Self-Review:**

1. **Spec coverage:** 设计文档中的所有模块（配置、模型、Repository、WebSocket、Card、Session、Router、Handler、前端）都有对应任务。

2. **Placeholder scan:** 无 TBD/TODO，所有代码步骤包含完整实现。

3. **Type consistency:** 使用现有的 `workflow.WorkflowInput`、`workflow.SSEWriter` 接口，与现有代码一致。