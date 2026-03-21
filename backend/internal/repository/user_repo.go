package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func GetUserByEmail(email string) (*model.User, error) {
	var u model.User
	err := db.DB.Where("email = ?", email).First(&u).Error
	return &u, err
}

func GetUserByID(id uint) (*model.User, error) {
	var u model.User
	err := db.DB.First(&u, id).Error
	return &u, err
}

func CreateUser(u *model.User) error {
	return db.DB.Create(u).Error
}

func GetStyleByUserID(userID uint) (*model.UserStyle, error) {
	var s model.UserStyle
	err := db.DB.Where("user_id = ?", userID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func UpsertStyle(s *model.UserStyle) error {
	var existing model.UserStyle
	err := db.DB.Where("user_id = ?", s.UserID).First(&existing).Error
	if err != nil {
		return db.DB.Create(s).Error
	}
	return db.DB.Model(&existing).Updates(s).Error
}
