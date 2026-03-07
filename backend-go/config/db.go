package config

import (
	"backend-go/models"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	// 连接 PostgreSQL（通过环境变量配置）
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "debugai"),
		getEnv("DB_PORT", "5432"),
	)

	// 增加重试逻辑，应对容器启动顺序问题
	for i := 0; i < 10; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("数据库连接尝试失败 (%d/10): %v. 2秒后重试...", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		panic("数据库连接最终失败: " + err.Error())
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
		&models.AuditLog{},
	)

	// 创建索引优化查询性能
	createIndexes(DB)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createIndexes(db *gorm.DB) {
	// PostgreSQL 使用 IF NOT EXISTS 语法创建索引
	db.Exec("CREATE INDEX IF NOT EXISTS idx_class_members_user_class ON class_members (user_id, class_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_class_members_class_role ON class_members (class_id, member_role)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_weak_points_student_weakpoint_date ON user_weak_points (student_id, weak_point_id, record_date)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_ai_records_conv_student_created ON ai_records (conversation_id, student_id, created_at)")
}
