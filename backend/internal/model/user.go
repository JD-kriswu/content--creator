package model

import (
	"time"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Email        string    `gorm:"uniqueIndex;size:128;not null" json:"email"`
	PasswordHash string    `gorm:"size:256;not null" json:"-"`
	Role         Role      `gorm:"size:16;default:'user'" json:"role"`
	Active       bool      `gorm:"default:true" json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"-"`
}

type UserStyle struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	UserID        uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	LanguageStyle string `gorm:"size:512" json:"language_style"` // 口语化/专业/接地气
	EmotionTone   string `gorm:"size:256" json:"emotion_tone"`   // 理性/感性/幽默
	OpeningStyle  string `gorm:"size:512" json:"opening_style"`
	ClosingStyle  string `gorm:"size:512" json:"closing_style"`
	Catchphrases  string `gorm:"type:text" json:"catchphrases"` // 口头禅，换行分隔
	UpdatedAt     int64  `json:"updated_at"`                    // Unix timestamp
}
