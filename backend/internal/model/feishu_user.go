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