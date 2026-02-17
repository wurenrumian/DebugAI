package service

import (
	"backend-go/config"
	"backend-go/models"

	"gorm.io/gorm"
)

// CanAccessClassData 检查用户是否有权限访问班级数据
// 返回值：是否允许访问，班级角色
func CanAccessClassData(userID uint, classID uint) (bool, string) {
	var member models.ClassMember
	err := config.DB.Where("user_id = ? AND class_id = ?", userID, classID).First(&member).Error

	if err == gorm.ErrRecordNotFound {
		return false, ""
	}

	if err != nil {
		return false, ""
	}

	// teacher 和 ta 可以访问班级数据
	if member.MemberRole == models.MemberRoleTeacher || member.MemberRole == models.MemberRoleTA {
		return true, member.MemberRole
	}

	return false, member.MemberRole
}

// IsClassTeacher 检查用户是否为班级老师
func IsClassTeacher(userID uint, classID uint) bool {
	var member models.ClassMember
	err := config.DB.Where("user_id = ? AND class_id = ? AND member_role = ?", userID, classID, models.MemberRoleTeacher).First(&member).Error
	return err == nil
}

// IsClassAdmin 检查用户是否为班级管理员（teacher 或 ta）
func IsClassAdmin(userID uint, classID uint) bool {
	var member models.ClassMember
	err := config.DB.Where("user_id = ? AND class_id = ? AND (member_role = ? OR member_role = ?)", userID, classID, models.MemberRoleTeacher, models.MemberRoleTA).First(&member).Error
	return err == nil
}

// IsClassCreator 检查用户是否为班级创建者
func IsClassCreator(userID uint, classID uint) bool {
	var member models.ClassMember
	err := config.DB.Where("user_id = ? AND class_id = ? AND is_creator = ?", userID, classID, true).First(&member).Error
	return err == nil
}

// GetUserRoleInClass 获取用户在班级中的角色
func GetUserRoleInClass(userID uint, classID uint) string {
	var member models.ClassMember
	err := config.DB.Where("user_id = ? AND class_id = ?", userID, classID).First(&member).Error
	if err == gorm.ErrRecordNotFound {
		return ""
	}
	return member.MemberRole
}

// IsAdmin 检查用户是否为管理员
func IsAdmin(userType string) bool {
	return userType == models.TypeAdmin
}
