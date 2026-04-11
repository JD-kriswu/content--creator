package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateFeishuUser(user *model.FeishuUser) error {
	return db.DB.Create(user).Error
}

func GetFeishuUserByOpenID(openID string) (*model.FeishuUser, error) {
	var user model.FeishuUser
	err := db.DB.Where("open_id = ?", openID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetOrCreateFeishuUserByOpenID(openID string) (*model.FeishuUser, error) {
	user, err := GetFeishuUserByOpenID(openID)
	if err == nil {
		return user, nil
	}
	newUser := &model.FeishuUser{
		OpenID:     openID,
		FeishuID:   openID,
		BindStatus: model.BindIndependent,
	}
	if err := CreateFeishuUser(newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

func UpdateFeishuUserBind(feishuUserID, webUserID uint) error {
	return db.DB.Model(&model.FeishuUser{}).
		Where("id = ?", feishuUserID).
		Updates(map[string]interface{}{
			"user_id":     webUserID,
			"bind_status": model.BindMerged,
		}).Error
}