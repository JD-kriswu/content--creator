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
	SenderID   SenderID `json:"sender_id"`
	SenderType string   `json:"sender_type"`
	TenantKey  string   `json:"tenant_key"`
}

type SenderID struct {
	OpenID  string `json:"open_id"`
	UnionID string `json:"union_id"`
	UserID  string `json:"user_id"`
}

type Message struct {
	MessageID  string `json:"message_id"`
	ChatID     string `json:"chat_id"`
	ChatType   string `json:"chat_type"`
	Content    string `json:"content"`
	CreateTime string `json:"create_time"` // Feishu uses string timestamp
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
	Config   CardConfig    `json:"config"`
	Header   CardHeader     `json:"header"`
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