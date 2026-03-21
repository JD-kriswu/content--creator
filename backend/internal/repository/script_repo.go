package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateScript(s *model.Script) error {
	return db.DB.Create(s).Error
}

func ListScripts(userID uint, page, limit int) ([]model.Script, int64, error) {
	var scripts []model.Script
	var total int64
	q := db.DB.Model(&model.Script{}).Where("user_id = ?", userID)
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&scripts).Error
	return scripts, total, err
}

func GetScript(id, userID uint) (*model.Script, error) {
	var s model.Script
	err := db.DB.Where("id = ? AND user_id = ?", id, userID).First(&s).Error
	return &s, err
}
