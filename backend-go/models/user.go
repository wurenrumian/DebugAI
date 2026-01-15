package models

import "gorm.io/gorm"

const (
	TypeStudent = "student"
	TypeAdmin   = "admin"
)

type User struct {
	gorm.Model           
	StudentID string `gorm:"unique;not null" json:"student_id"` // 业务唯一标识
	UserType  string `gorm:"default:student" json:"user_type"`  // 账户类型
	Username  string `gorm:"not null" json:"username"`          // 用户名
	Password  string `gorm:"not null" json:"-"`                 // 密码脱敏
}