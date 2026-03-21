package db

import (
	"database/sql"
	"fmt"

	"content-creator-imm/config"
	"content-creator-imm/internal/model"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() error {
	c := config.C

	// Step 1: Connect without specifying database name, create DB if not exists
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort)
	rawDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer rawDB.Close()

	_, err = rawDB.Exec(fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		c.DBName,
	))
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	// Step 2: Connect with database name
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}

	// Step 3: AutoMigrate tables
	return DB.AutoMigrate(
		&model.User{},
		&model.UserStyle{},
		&model.Script{},
		&model.Conversation{},
		&model.Message{},
	)
}
