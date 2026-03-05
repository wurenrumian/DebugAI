package utils

import "fmt"

// ==================== 轮次配置缓存键 ====================

// RoundInfoKey 轮次配置缓存键
func RoundInfoKey(round int) string {
	return fmt.Sprintf("round_info:%d", round)
}

// ==================== 班级相关缓存键 ====================

// ClassListKey 班级列表缓存键
func ClassListKey() string {
	return "classes:list"
}

// ClassBasicKey 班级基本信息缓存键
func ClassBasicKey(classID uint) string {
	return fmt.Sprintf("class:basic:%d", classID)
}

// ClassMembersKey 班级成员列表缓存键
func ClassMembersKey(classID uint) string {
	return fmt.Sprintf("class:members:%d", classID)
}

// ClassMembersPageKey 班级成员分页缓存键
func ClassMembersPageKey(classID uint, page int) string {
	return fmt.Sprintf("class:members:%d:page:%d", classID, page)
}

// ClassDetailKey 班级完整信息缓存键（包含基本信息和成员）
func ClassDetailKey(classID uint) string {
	return fmt.Sprintf("class:detail:%d", classID)
}

// ==================== 薄弱点相关缓存键 ====================

// WeakPointsUserKey 用户薄弱点缓存键
func WeakPointsUserKey(studentID, startDate, endDate string) string {
	return fmt.Sprintf("weak_points:user:%s:%s:%s", studentID, startDate, endDate)
}

// WeakPointsUserTopKey 用户Top-N薄弱点缓存键
func WeakPointsUserTopKey(studentID string, limit int, startDate, endDate string) string {
	return fmt.Sprintf("weak_points:user:top:%s:%d:%s:%s", studentID, limit, startDate, endDate)
}

// WeakPointsClassKey 班级薄弱点缓存键
func WeakPointsClassKey(classID uint, startDate, endDate string) string {
	return fmt.Sprintf("weak_points:class:%d:%s:%s", classID, startDate, endDate)
}

// WeakPointsUserPattern 用户薄弱点缓存键模式（用于批量删除）
func WeakPointsUserPattern(studentID string) string {
	return fmt.Sprintf("weak_points:user:%s:*", studentID)
}

// WeakPointsUserTopPattern 用户Top-N薄弱点缓存键模式
func WeakPointsUserTopPattern(studentID string) string {
	return fmt.Sprintf("weak_points:user:top:%s:*", studentID)
}

// WeakPointsClassPattern 班级薄弱点缓存键模式
func WeakPointsClassPattern(classID uint) string {
	return fmt.Sprintf("weak_points:class:%d:*", classID)
}

// ==================== 缓存键前缀（用于版本控制） ====================

const (
	// CacheVersionPrefix 缓存版本前缀
	CacheVersionPrefix = "v1"
)

// WithVersion 添加版本前缀
func WithVersion(key string) string {
	return fmt.Sprintf("%s:%s", CacheVersionPrefix, key)
}
