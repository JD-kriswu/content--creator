package repository

import (
	"time"
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateFeishuBot(bot *model.FeishuBot) error {
	return db.DB.Create(bot).Error
}

func GetFeishuBotByAppID(appID string) (*model.FeishuBot, error) {
	var bot model.FeishuBot
	err := db.DB.Where("app_id = ?", appID).First(&bot).Error
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

func GetFeishuBotsByUserID(userID uint) ([]model.FeishuBot, error) {
	var bots []model.FeishuBot
	err := db.DB.Where("user_id = ?", userID).Find(&bots).Error
	return bots, err
}

func GetConnectedFeishuBots() ([]model.FeishuBot, error) {
	var bots []model.FeishuBot
	err := db.DB.Where("ws_connected = ?", true).Find(&bots).Error
	return bots, err
}

func UpdateFeishuBotWSStatus(botID uint, connected bool) error {
	return db.DB.Model(&model.FeishuBot{}).
		Where("id = ?", botID).
		Updates(map[string]interface{}{
			"ws_connected":   connected,
			"last_heartbeat": time.Now(),
		}).Error
}

func DeleteFeishuBot(botID, userID uint) error {
	return db.DB.Where("id = ? AND user_id = ?", botID, userID).
		Delete(&model.FeishuBot{}).Error
}