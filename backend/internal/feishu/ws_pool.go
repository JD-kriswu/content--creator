package feishu

import (
	"log"
	"sync"

	"content-creator-imm/internal/repository"
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

	// Update ws_connected status in database
	bot, err := repository.GetFeishuBotByAppID(appID)
	if err == nil {
		repository.UpdateFeishuBotWSStatus(bot.ID, true)
	}

	return nil
}

func (p *WSConnectionPool) Disconnect(appID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[appID]; ok {
		conn.Disconnect()
		delete(p.connections, appID)

		// Update ws_connected status in database
		bot, err := repository.GetFeishuBotByAppID(appID)
		if err == nil {
			repository.UpdateFeishuBotWSStatus(bot.ID, false)
		}
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