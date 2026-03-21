package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateMessage(m *model.Message) error {
	return db.DB.Create(m).Error
}

func ListMessagesByConvID(convID uint) ([]model.Message, error) {
	var list []model.Message
	err := db.DB.Where("conversation_id = ?", convID).Order("id ASC").Find(&list).Error
	return list, err
}
