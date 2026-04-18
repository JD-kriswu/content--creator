package feishu

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"content-creator-imm/internal/feishu/pb"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
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
	serviceID      int64 // from WebSocket URL query param
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

// getWSTicket calls Feishu API to get WebSocket connection URL
func (c *WSConnection) getWSTicket() (string, string, error) {
	// Call Feishu endpoint API to get WebSocket URL
	endpointURL := "https://open.feishu.cn/callback/ws/endpoint"

	client := &http.Client{Timeout: 30 * time.Second}
	reqBody := map[string]string{"AppID": c.AppID, "AppSecret": c.AppSecret}
	reqBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", endpointURL, bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("locale", "zh")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("call endpoint API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[FeishuWS] endpoint response: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("endpoint API status: %d", resp.StatusCode)
	}

	var endpointResult struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			URL          string `json:"URL"`
			ClientConfig *struct {
				ReconnectNonce int `json:"ReconnectNonce"`
			} `json:"ClientConfig"`
		} `json:"data"`
	}
	json.Unmarshal(respBody, &endpointResult)

	if endpointResult.Code != 0 {
		return "", "", fmt.Errorf("endpoint API error: code=%d msg=%s", endpointResult.Code, endpointResult.Msg)
	}

	if endpointResult.Data.URL == "" {
		return "", "", fmt.Errorf("endpoint API returned empty URL")
	}

	return endpointResult.Data.URL, "", nil
}

func (c *WSConnection) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get WebSocket URL from Feishu endpoint API
	wsURL, _, err := c.getWSTicket()
	if err != nil {
		log.Printf("[FeishuWS] getWSTicket failed: %v", err)
		return fmt.Errorf("get ws url: %w", err)
	}
	log.Printf("[FeishuWS] got URL: %s", wsURL)

	// Parse service_id from URL
	u, err := url.Parse(wsURL)
	if err == nil {
		serviceIDStr := u.Query().Get("service_id")
		if serviceIDStr != "" {
			c.serviceID, _ = strconv.ParseInt(serviceIDStr, 10, 64)
			log.Printf("[FeishuWS] service_id=%d", c.serviceID)
		}
	}

	// Connect to WebSocket
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

	// Create protobuf ping frame
	// FrameType: CONTROL = 0
	seqID := uint64(0)
	logID := uint64(0)
	service := int32(c.serviceID)
	method := int32(0) // CONTROL
	keyType := "type"
	valuePing := "ping"

	frame := &pb.Frame{
		SeqID:   &seqID,
		LogID:   &logID,
		Service: &service,
		Method:  &method,
		Headers: []*pb.Header{
			{Key: &keyType, Value: &valuePing},
		},
	}

	data, err := proto.Marshal(frame)
	if err != nil {
		log.Printf("[FeishuWS] ping marshal error: %v", err)
		return
	}

	if err := c.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
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

			// Parse protobuf Frame
			event := c.parseProtobufFrame(msg)
			if event.Type == "pong" || event.Type == "" {
				continue
			}
			log.Printf("[FeishuWS] received event: type=%s app_id=%s", event.Type, event.AppID)
			if c.MessageHandler != nil {
				c.MessageHandler(event)
			}
		}
	}
}

// parseProtobufFrame parses Feishu WebSocket protobuf message into WSEvent
func (c *WSConnection) parseProtobufFrame(data []byte) WSEvent {
	frame := &pb.Frame{}
	if err := proto.Unmarshal(data, frame); err != nil {
		log.Printf("[FeishuWS] protobuf unmarshal error: %v", err)
		return WSEvent{}
	}

	// Extract headers using getter methods
	headers := make(map[string]string)
	for _, h := range frame.GetHeaders() {
		headers[h.GetKey()] = h.GetValue()
	}

	// Get message type from headers (ping/pong/event)
	msgType := headers["type"]
	if msgType == "pong" {
		return WSEvent{Type: "pong"}
	}

	// Parse payload to get event details
	var rawEvent json.RawMessage
	var appID, tenantKey, eventType string
	payload := frame.GetPayload()

	if len(payload) > 0 {
		// Handle gzip encoding if needed
		if frame.GetPayloadEncoding() == "gzip" {
			// Decompress gzip payload
			decompressed, err := decompressGzip(payload)
			if err != nil {
				log.Printf("[FeishuWS] gzip decompress error: %v", err)
			} else {
				payload = decompressed
			}
		}

		rawEvent = json.RawMessage(payload)

		// Parse event payload
		var eventData map[string]interface{}
		if json.Unmarshal(payload, &eventData) == nil {
			// Extract header fields (event_type, app_id, tenant_key)
			if header, ok := eventData["header"].(map[string]interface{}); ok {
				if etVal, ok := header["event_type"]; ok {
					eventType = fmt.Sprintf("%v", etVal)
				}
				if appID == "" {
					if appIDVal, ok := header["app_id"]; ok {
						appID = fmt.Sprintf("%v", appIDVal)
					}
				}
				if tenantKey == "" {
					if tkVal, ok := header["tenant_key"]; ok {
						tenantKey = fmt.Sprintf("%v", tkVal)
					}
				}
			}
			// Also check root level
			if appIDVal, ok := eventData["app_id"]; ok {
				appID = fmt.Sprintf("%v", appIDVal)
			}
			if tenantKeyVal, ok := eventData["tenant_key"]; ok {
				tenantKey = fmt.Sprintf("%v", tenantKeyVal)
			}
		}
	}

	// Fallback to headers if not found in payload
	if appID == "" {
		appID = headers["app_id"]
	}
	if tenantKey == "" {
		tenantKey = headers["tenant_key"]
	}

	// Use connection's AppID if still empty
	if appID == "" {
		appID = c.AppID
	}

	// Use msgType (from frame headers) as eventType if not found in payload
	if eventType == "" {
		eventType = msgType
	}

	log.Printf("[FeishuWS] frame: SeqID=%d service=%d method=%d msg_type=%s event_type=%s app_id=%s tenant_key=%s payload_len=%d",
		frame.GetSeqID(), frame.GetService(), frame.GetMethod(), msgType, eventType, appID, tenantKey, len(payload))

	return WSEvent{
		Type:      eventType, // Use actual event type like "im.message.receive_v1"
		AppID:     appID,
		TenantKey: tenantKey,
		Event:     rawEvent,
	}
}

func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
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