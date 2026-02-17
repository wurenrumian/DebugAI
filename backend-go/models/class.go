package models

import "time"

const (
	MemberRoleTeacher = "teacher"
	MemberRoleTA      = "ta"
	MemberRoleStudent = "student"
)

// Class 班级表
type Class struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ClassName string    `gorm:"size:255;not null" json:"class_name"`
	CreatedBy uint      `gorm:"not null" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// ClassMember 班级成员表
type ClassMember struct {
	ID         uint   `gorm:"primarykey" json:"id"`
	ClassID    uint   `gorm:"not null;index" json:"class_id"`
	UserID     uint   `gorm:"not null;index" json:"user_id"`
	MemberRole string `gorm:"size:20;default:student" json:"member_role"`
	IsCreator  bool   `gorm:"default:false" json:"is_creator"` // 是否为班级创建者
}
