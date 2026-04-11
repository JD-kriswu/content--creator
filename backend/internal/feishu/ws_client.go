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