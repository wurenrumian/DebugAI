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

// IsClassMember 检查用户是否为班级成员（任意角色）
func IsClassMember(userID uint, classID uint) bool {
	var count int64
	err := config.DB.Model(&models.ClassMember{}).Where("user_id = ? AND class_id = ?", userID, classID).Count(&count).Error
	return err == nil && count > 0
}

// CanChangeMemberRole 判断操作者能否修改某个已有成员的角色（纯函数，便于测试）
//   - 创建者的角色任何人不可修改
//   - 教师的角色只有班级创建者可以修改
//   - 助教只能管理学生，不能修改教师/助教的角色
//   - 将成员提升为教师/助教需要班级创建者或系统管理员
//
// 返回是否允许以及拒绝原因
func CanChangeMemberRole(operatorRole string, operatorIsCreator, operatorIsSysAdmin bool, target models.ClassMember, newRole string) (bool, string) {
	if target.IsCreator {
		return false, "创建者角色不可修改"
	}
	if target.MemberRole == models.MemberRoleTeacher && !operatorIsCreator {
		return false, "只有班级创建者可以修改教师角色"
	}
	if operatorRole == models.MemberRoleTA && !operatorIsCreator && target.MemberRole != models.MemberRoleStudent {
		return false, "助教只能管理学生"
	}
	if newRole != models.MemberRoleStudent && !operatorIsCreator && !operatorIsSysAdmin {
		return false, "只有班级创建者或管理员可以提升成员角色"
	}
	return true, ""
}

// CanRemoveMember 判断操作者能否移除某个成员（纯函数，便于测试）
//   - 创建者不可被移除
//   - 教师只有班级创建者可以移除
//   - 助教只能移除学生
func CanRemoveMember(operatorRole string, operatorIsCreator bool, target models.ClassMember) (bool, string) {
	if target.IsCreator {
		return false, "班级创建者不可移除"
	}
	if target.MemberRole == models.MemberRoleTeacher && !operatorIsCreator {
		return false, "只有班级创建者可以移除教师"
	}
	if operatorRole == models.MemberRoleTA && !operatorIsCreator && target.MemberRole != models.MemberRoleStudent {
		return false, "助教只能移除学生，不能移除教师或助教"
	}
	return true, ""
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
