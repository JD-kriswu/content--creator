package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateConversation(c *model.Conversation) error {
	return db.DB.Create(c).Error
}

func UpdateConversation(c *model.Conversation) error {
	return db.DB.Save(c).Error
}

func UpdateConversationTitle(id uint, title string) error {
	return db.DB.Model(&model.Conversation{}).Where("id = ?", id).Update("title", title).Error
}

func UpdateConversationMeta(id uint, updates map[string]interface{}) error {
	return db.DB.Model(&model.Conversation{}).Where("id = ?", id).Updates(updates).Error
}

func ListConversations(userID uint, limit int) ([]model.Conversation, error) {
	var list []model.Conversation
	err := db.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Select("id, user_id, title, script_id, state, created_at, updated_at").
		Find(&list).Error
	return list, err
}

func GetConversation(id, userID uint) (*model.Conversation, error) {
	var c model.Conversation
	err := db.DB.Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}
