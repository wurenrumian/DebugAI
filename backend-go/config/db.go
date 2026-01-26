package config

import (
	"backend-go/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	// 连接 SQLite，会在本地生成 data.db 文件
	DB, err = gorm.Open(sqlite.Open("data.db"), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}

	// 自动同步表结构
	DB.AutoMigrate(&models.User{}, &models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})
}
