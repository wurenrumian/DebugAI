package models

import (
	"gorm.io/gorm"
)

// AuditLog 审计日志表
type AuditLog struct {
	gorm.Model
	UserID        uint   `json:"user_id" gorm:"index"`             // 操作用户ID（可为空，如未登录操作）
	StudentID     string `json:"student_id" gorm:"size:50;index"`  // 操作用户学号
	UserType      string `json:"user_type" gorm:"size:20"`         // 用户类型
	Action        string `json:"action" gorm:"size:100;not null"`  // 操作类型：login, logout, register, password_change等
	Resource      string `json:"resource" gorm:"size:100"`         // 操作资源（如user、profile等）
	ResourceID    string `json:"resource_id" gorm:"size:100"`      // 资源ID
	IP            string `json:"ip" gorm:"size:45"`                // 操作IP地址
	UserAgent     string `json:"user_agent" gorm:"size:500"`       // 用户代理
	Method        string `json:"method" gorm:"size:10"`            // HTTP方法
	Path          string `json:"path" gorm:"size:255"`             // 请求路径
	Status        int    `json:"status" gorm:"default:200"`        // HTTP状态码
	Success       bool   `json:"success" gorm:"default:true"`      // 操作是否成功
	FailureReason string `json:"failure_reason" gorm:"size:500"`   // 失败原因
	Duration      int64  `json:"duration" gorm:"comment:操作耗时(ms)"` // 操作耗时（毫秒）
	Extra         string `json:"extra" gorm:"type:text"`           // 额外信息（JSON格式）
}

// AuditLogAction 操作类型常量
const (
	ActionLogin            = "login"
	ActionLogout           = "logout"
	ActionRegister         = "register"
	ActionPasswordChange   = "password_change"
	ActionPasswordReset    = "password_reset"
	ActionProfileUpdate    = "profile_update"
	ActionAuthFailure      = "auth_failure"
	ActionPermissionChange = "permission_change"
)

// CreateAuditLog 创建审计日志
func CreateAuditLog(db *gorm.DB, log *AuditLog) error {
	return db.Create(log).Error
}
