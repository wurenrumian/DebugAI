package service

import (
	"fmt"
	"sync/atomic"
	"testing"

	"backend-go/logger"
	"backend-go/models"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var testDBSeq int64

func init() {
	if logger.Logger == nil {
		logger.Logger = zap.NewNop()
	}
}

// newTestDB 创建独立的内存 SQLite 数据库并迁移相关表
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := fmt.Sprintf("file:svc_test_%d?mode=memory&cache=shared", atomic.AddInt64(&testDBSeq, 1))
	db, err := gorm.Open(sqlite.Open(name), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&models.User{},
		&models.Class{},
		&models.ClassMember{},
		&models.AIRecord{},
		&models.Conversation{},
		&models.WeakPoint{},
		&models.UserWeakPoint{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// countRecords 统计指定角色的记录数
func countRecords(t *testing.T, db *gorm.DB, conversationID, role string) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.AIRecord{}).Where("conversation_id = ? AND role = ?", conversationID, role).Count(&n).Error; err != nil {
		t.Fatalf("count records: %v", err)
	}
	return n
}
