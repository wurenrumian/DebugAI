package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	TypeUser  = "user"
	TypeAdmin = "admin"
)

type User struct {
	gorm.Model
	StudentID                  string     `gorm:"unique;not null;size:50" json:"student_id"` // 业务唯一标识
	UserType                   string     `gorm:"default:user;size:20" json:"user_type"`     // 用户类型
	Username                   string     `gorm:"not null;size:100" json:"username"`         // 用户名
	Password                   string     `gorm:"not null;size:255" json:"-"`                // 密码脱敏
	TokenVersion               int        `gorm:"default:0" json:"-"`                        // Token版本号，用于权限变更后使旧token失效
	Email                      string     `gorm:"size:255;not null" json:"email"`            // 邮箱地址（必填）
	EmailVerified              bool       `gorm:"default:false" json:"email_verified"`       // 邮箱是否已验证
	EmailVerificationToken     string     `gorm:"size:64" json:"-"`                          // 邮箱验证token
	EmailVerificationSentAt    *time.Time `json:"email_verification_sent_at"`                // 验证邮件发送时间
	EmailVerificationExpiresAt *time.Time `json:"email_verification_expires_at"`             // 验证链接过期时间
}
