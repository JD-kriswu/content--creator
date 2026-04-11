package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateFeishuConv(conv *model.FeishuConversation) error {
	return db.DB.Create(conv).Error
}

func GetFeishuConvByChatID(chatID string) (*model.FeishuConversation, error) {
	var conv model.FeishuConversation
	err := db.DB.Where("feishu_chat_id = ?", chatID).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetOrCreateFeishuConv creates a feishu conversation mapping, and creates associated Conversation record
func GetOrCreateFeishuConv(botID uint, chatID string, userID uint) (*model.FeishuConversation, uint, error) {
	conv, err := GetFeishuConvByChatID(chatID)
	if err == nil && conv.ConvID > 0 {
		return conv, conv.ConvID, nil
	}
	// Create new conversation record
	webConv := &model.Conversation{
		UserID: userID,
		Title:  "飞书对话",
		State:  0,
	}
	if err := CreateConversation(webConv); err != nil {
		return nil, 0, err
	}
	newConv := &model.FeishuConversation{
		BotID:        botID,
		ConvID:       webConv.ID,
		FeishuChatID: chatID,
	}
	if err := CreateFeishuConv(newConv); err != nil {
		return nil, 0, err
	}
	return newConv, webConv.ID, nil
}

func DeleteFeishuConvsByBotID(botID uint) error {
	return db.DB.Where("bot_id = ?", botID).Delete(&model.FeishuConversation{}).Error
}