package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationResult 统一验证结果
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// ValidateInput 统一验证入口
func ValidateInput(inputType, value string) ValidationResult {
	switch inputType {
	case "student_id":
		valid, errMsg := ValidateStudentID(value)
		if !valid {
			return ValidationResult{Valid: false, Errors: []string{errMsg}}
		}
	case "username":
		valid, errMsg := ValidateUsername(value)
		if !valid {
			return ValidationResult{Valid: false, Errors: []string{errMsg}}
		}
	case "password":
		valid, errors := ValidatePassword(value)
		if !valid {
			return ValidationResult{Valid: false, Errors: errors}
		}
	}
	return ValidationResult{Valid: true}
}

// PasswordComplexity 密码复杂度要求
type PasswordComplexity struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSpecial bool
	MaxRepeated    int // 最大连续重复字符数
}

// DefaultPasswordComplexity 默认密码复杂度（简化版）
var DefaultPasswordComplexity = PasswordComplexity{
	MinLength:      6,
	RequireUpper:   false,
	RequireLower:   false,
	RequireDigit:   false, // 简化：移除数字强制要求
	RequireSpecial: false,
	MaxRepeated:    0,
}

// ValidatePassword 验证密码复杂度
func ValidatePassword(password string) (bool, []string) {
	var errors []string

	// 长度检查
	if len(password) < DefaultPasswordComplexity.MinLength {
		errors = append(errors, fmt.Sprintf("Password must be at least %d characters", DefaultPasswordComplexity.MinLength))
	}

	// 中文字符检查（禁止）
	if regexp.MustCompile(`[\x{4e00}-\x{9fa5}]`).MatchString(password) {
		errors = append(errors, "Password cannot contain Chinese characters")
	}

	// 空格检查（禁止）
	if strings.Contains(password, " ") {
		errors = append(errors, "Password cannot contain spaces")
	}

	// 数字检查
	if DefaultPasswordComplexity.RequireDigit {
		if !regexp.MustCompile(`[0-9]`).MatchString(password) {
			errors = append(errors, "Must contain at least one digit")
		}
	}

	// 常见弱密码检查
	weakPasswords := []string{
		//临时注释，用于测试，测试后删除注释
		"password", "123456", "qwerty", "admin", "letmein",
		"welcome", "monkey", "dragon", "baseball", "football",
	}
	for _, wp := range weakPasswords {
		if len(password) >= len(wp) &&
			(password == wp || regexp.MustCompile(`(?i)^`+regexp.QuoteMeta(wp)).MatchString(password)) {
			errors = append(errors, "Password is too common, weak passwords are prohibited")
			break
		}
	}

	return len(errors) == 0, errors
}

// SanitizeInput 输入消毒，去除不可见字符和控制字符
func SanitizeInput(input string) string {
	// 移除控制字符（ASCII 0-31 和 127）
	re := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	sanitized := re.ReplaceAllString(input, "")

	// 修剪首尾空白
	return strings.TrimSpace(sanitized)
}

// ValidateStudentID 学号格式验证（简化版）
func ValidateStudentID(studentID string) (bool, string) {
	// 空格检查
	if strings.Contains(studentID, " ") {
		return false, "学号不能包含空格"
	}
	// 简单检查：以2023或2024开头，长度8-12位数字
	if len(studentID) < 8 || len(studentID) > 12 {
		return false, "学号长度应为8-12位"
	}
	for _, c := range studentID {
		if c < '0' || c > '9' {
			return false, "学号只能包含数字"
		}
	}
	return true, ""
}

// ValidateUsername 用户名验证（过滤危险字符）
func ValidateUsername(username string) (bool, string) {
	// 空格检查
	if strings.Contains(username, " ") {
		return false, "用户名不能包含空格"
	}
	// 只允许字母、数字、下划线、中文、短横线
	pattern := `^[a-zA-Z0-9_\p{Han}\-]+$`
	if matched, _ := regexp.MatchString(pattern, username); !matched {
		return false, "用户名只能包含字母、数字、下划线、短横线"
	}

	// 长度检查
	if len(username) < 2 || len(username) > 100 {
		return false, "用户名长度必须在2-100个字符之间"
	}

	return true, ""
}
