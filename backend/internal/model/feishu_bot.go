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