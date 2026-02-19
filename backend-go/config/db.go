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
	DB.AutoMigrate(
		&models.User{},
		&models.AIRecord{},
		&models.WeakPoint{},
		&models.UserWeakPoint{},
		&models.Conversation{},
		&models.Class{},
		&models.ClassMember{},
	)

	// 创建索引优化查询性能
	createIndexes(DB)
}

func createIndexes(db *gorm.DB) {
	// 为 class_members 表创建复合索引，优化 GetMyClasses 和权限查询
	db.Exec("CREATE INDEX IF NOT EXISTS idx_class_members_user_class ON class_members(user_id, class_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_class_members_class_role ON class_members(class_id, member_role)")
}
