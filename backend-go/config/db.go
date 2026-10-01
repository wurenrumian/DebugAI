package config

import (
	"backend-go/models"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// DBConfig 数据库连接配置（通过环境变量配置）
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大空闲连接数
	ConnMaxLifetime time.Duration // 连接最大存活时间
	ConnMaxIdleTime time.Duration // 空闲连接最大存活时间
}

// LoadDBConfig 从环境变量加载数据库配置
func LoadDBConfig() DBConfig {
	return DBConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        os.Getenv("DB_PASSWORD"), // 不提供默认密码
		Name:            getEnv("DB_NAME", "debugai"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 20),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 5)) * time.Minute,
	}
}

// Validate 校验数据库配置
func (c DBConfig) Validate(appEnv string) error {
	if appEnv == "production" && c.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required in production environment")
	}
	switch c.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("invalid DB_SSLMODE: %q", c.SSLMode)
	}
	if c.MaxOpenConns <= 0 || c.MaxIdleConns < 0 || c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("invalid DB pool config: DB_MAX_OPEN_CONNS=%d DB_MAX_IDLE_CONNS=%d", c.MaxOpenConns, c.MaxIdleConns)
	}
	return nil
}

// DSN 生成 PostgreSQL key/value 格式连接串（对值做转义，支持包含空格/引号的密码）
func (c DBConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		quoteDSNValue(c.Host),
		quoteDSNValue(c.User),
		quoteDSNValue(c.Password),
		quoteDSNValue(c.Name),
		quoteDSNValue(c.Port),
		quoteDSNValue(c.SSLMode),
	)
}

// quoteDSNValue 按 libpq 规则用单引号包裹值，并转义反斜杠与单引号
func quoteDSNValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

func InitDB() {
	dbCfg := LoadDBConfig()
	appEnv := ""
	if Global != nil {
		appEnv = Global.AppEnv
	}
	if err := dbCfg.Validate(appEnv); err != nil {
		panic("数据库配置错误: " + err.Error())
	}

	var err error
	// 增加重试逻辑，应对容器启动顺序问题
	for i := 0; i < 10; i++ {
		DB, err = gorm.Open(postgres.Open(dbCfg.DSN()), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("数据库连接尝试失败 (%d/10): %v. 2秒后重试...", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		panic("数据库连接最终失败: " + err.Error())
	}

	// 连接池配置
	sqlDB, err := DB.DB()
	if err != nil {
		panic("获取数据库连接池失败: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(dbCfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbCfg.ConnMaxIdleTime)

	// 自动同步表结构（失败则终止启动，避免带着不完整的表结构运行）
	if err := DB.AutoMigrate(
		&models.User{},
		&models.AIRecord{},
		&models.WeakPoint{},
		&models.UserWeakPoint{},
		&models.Conversation{},
		&models.Class{},
		&models.ClassMember{},
		&models.AuditLog{},
	); err != nil {
		panic("数据库表结构迁移失败: " + err.Error())
	}

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
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_class_members_user_class ON class_members (user_id, class_id)",
		"CREATE INDEX IF NOT EXISTS idx_class_members_class_role ON class_members (class_id, member_role)",
		"CREATE INDEX IF NOT EXISTS idx_user_weak_points_student_weakpoint_date ON user_weak_points (student_id, weak_point_id, record_date)",
		"CREATE INDEX IF NOT EXISTS idx_ai_records_conv_student_created ON ai_records (conversation_id, student_id, created_at)",
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			log.Printf("创建索引失败: %v (sql=%s)", err, stmt)
		}
	}

	// 邮箱唯一索引（部分索引：忽略空邮箱与已软删除用户）。
	// 不放在 GORM tag 中，是因为历史数据若存在重复邮箱，AutoMigrate 会直接失败导致服务无法启动；
	// 这里失败只记录告警，清理重复数据后重启即可自动创建。
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users (email) WHERE email <> '' AND deleted_at IS NULL").Error; err != nil {
		log.Printf("[WARN] 创建邮箱唯一索引失败（可能存在重复邮箱，请清理后重启）: %v", err)
	}
}
