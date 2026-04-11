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