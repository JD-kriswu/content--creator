package model

import "time"

type Script struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index;not null" json:"user_id"`
	Title           string    `gorm:"size:256" json:"title"`
	SourceURL       string    `gorm:"size:1024" json:"source_url"`
	Platform        string    `gorm:"size:32" json:"platform"` // douyin/xiaohongshu/bilibili
	ContentPath     string    `gorm:"size:1024" json:"content_path"` // local path or OSS URL
	SimilarityScore float64   `json:"similarity_score"`
	ViralScore      float64   `json:"viral_score"`
	Tags            string    `gorm:"type:text" json:"tags"` // JSON array
	CreatedAt       time.Time `json:"created_at"`
}
