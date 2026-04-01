package model

import "time"

// Message represents a single chat message persisted to DB.
type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"index;not null" json:"conversation_id"`
	Role           string    `gorm:"size:20" json:"role"`    // user / assistant
	Type           string    `gorm:"size:30" json:"type"`    // text/step/info/outline/action/similarity/complete/error
	Content        string    `gorm:"type:text" json:"content,omitempty"`
	DataJSON       string    `gorm:"column:data;type:text" json:"-"`    // JSON for outline/similarity
	OptionsJSON    string    `gorm:"column:options;type:text" json:"-"` // JSON array for action
	Step           int       `json:"step,omitempty"`
	Name           string    `gorm:"size:200" json:"name,omitempty"`
	StageID        string    `gorm:"size:64;index" json:"stage_id,omitempty"`
	WorkerName     string    `gorm:"size:64" json:"worker_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
