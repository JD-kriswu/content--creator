package model

import "time"

// Conversation represents a chat session between a user and the AI.
type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"size:200" json:"title"`
	Messages  string    `gorm:"type:longtext" json:"-"` // JSON array of StoredMsg
	ScriptID  *uint     `json:"script_id,omitempty"`
	State     int       `json:"state"` // 0=in_progress 1=completed
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
