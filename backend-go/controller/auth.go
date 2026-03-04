package controller

import (
	"backend-go/config"
	"backend-go/models"
	"backend-go/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Register 用户注册
func Register(c *gin.Context) {
	var input struct {
		StudentID string `json:"student_id" binding:"required,max=50"`
		Username  string `json:"username" binding:"required,min=2,max=100"`
		Password  string `json:"password" binding:"required,min=6,max=128"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}

	// 获取客户端IP
	ip := utils.GetIPFromRequest(c.Request)

	// 频率限制检查（已添加nil检查，不会panic）
	if utils.GlobalSecurity != nil && !utils.GlobalSecurity.CheckRateLimit("register:"+ip, 10) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "注册频率过高，请稍后再试"})
		return
	}

	// 输入消毒
	input.StudentID = utils.SanitizeInput(input.StudentID)
	input.Username = utils.SanitizeInput(input.Username)
	input.Password = utils.SanitizeInput(input.Password)

	// 统一验证
	if result := utils.ValidateInput("student_id", input.StudentID); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Errors[0]})
		return
	}
	if result := utils.ValidateInput("username", input.Username); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Errors[0]})
		return
	}
	if result := utils.ValidateInput("password", input.Password); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "密码不符合安全要求",
			"details": result.Errors,
		})
		return
	}

	// 检查学号是否已存在
	var existingUser models.User
	if err := config.DB.Where("student_id = ?", input.StudentID).First(&existingUser).Error; err == nil {
		// 找到记录，学号已被占用
		c.JSON(http.StatusConflict, gin.H{"error": "学号不可用"})
		return
	} else if err != gorm.ErrRecordNotFound {
		// 数据库查询错误（非记录不存在的情况）
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 密码哈希（使用配置的bcrypt成本）
	cost := config.Global.BCryptCost
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), cost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user := models.User{
		StudentID: input.StudentID,
		Username:  input.Username,
		Password:  string(hashedPassword),
		UserType:  models.TypeUser,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 记录注册成功审计日志
	go recordRegisterSuccess(c, user.ID, user.StudentID)

	c.JSON(http.StatusOK, gin.H{"message": "注册成功"})
}

// Login 用户登录
func Login(c *gin.Context) {
	var input struct {
		StudentID string `json:"student_id" binding:"required"`
		Password  string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入学号和密码"})
		return
	}

	// 获取客户端IP
	ip := utils.GetIPFromRequest(c.Request)

	// 检查IP是否被锁定
	if locked, unlockTime := utils.GlobalSecurity.CheckIPLockout(ip); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "IP已被临时锁定",
			"unlock_time": unlockTime.Format(time.RFC3339),
		})
		return
	}

	// 检查用户是否被锁定
	if locked, unlockTime := utils.GlobalSecurity.CheckLockout(input.StudentID); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "账户已被临时锁定",
			"unlock_time": unlockTime.Format(time.RFC3339),
		})
		return
	}

	// 检查是否需要验证码
	if utils.GlobalSecurity.IsCaptchaRequired(input.StudentID) {
		var captchaInput struct {
			Captcha string `json:"captcha" binding:"required"`
		}
		if err := c.ShouldBindJSON(&captchaInput); err != nil || !utils.VerifyCaptcha(captchaInput.Captcha, "") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "需要验证码或验证码错误"})
			return
		}
		// 验证成功后清除验证码要求
		utils.GlobalSecurity.ClearCaptchaRequirement(input.StudentID)
	}

	var user models.User
	// 按学号查询
	if err := config.DB.Where("student_id = ?", input.StudentID).First(&user).Error; err != nil {
		// 记录失败
		utils.GlobalSecurity.RecordLoginFailure(input.StudentID, ip)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "学号或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		// 记录失败
		failCount, requiresCaptcha := utils.GlobalSecurity.RecordLoginFailure(input.StudentID, ip)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":            "学号或密码错误",
			"fail_count":       failCount,
			"requires_captcha": requiresCaptcha,
		})
		return
	}

	// 登录成功，清除失败记录
	utils.GlobalSecurity.RecordLoginSuccess(input.StudentID, ip)

	token, err := utils.GenerateToken(user.ID, user.StudentID, user.UserType, user.TokenVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成Token失败"})
		return
	}

	// Set HttpOnly cookie（使用配置的过期时间，单位：秒）
	expirySeconds := config.Global.JWTExpiry * 3600
	c.SetCookie("auth_token", token, expirySeconds, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"data": gin.H{
			"username":   user.Username,
			"user_type":  user.UserType,
			"student_id": user.StudentID,
			"token":      token,
		},
	})
}

// Logout 用户登出
func Logout(c *gin.Context) {
	// 获取用户信息
	userID, _ := c.Get("user_id")
	studentID, _ := c.Get("student_id")

	// 清除cookie
	c.SetCookie("auth_token", "", -1, "/", "", false, true)

	// 记录登出审计日志
	go recordLogout(c, getUint(userID), getString(studentID))

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// 辅助函数：记录认证失败
func recordAuthFailure(c *gin.Context, studentID, ip, reason string) {
	log := &models.AuditLog{
		StudentID:     studentID,
		Action:        models.ActionAuthFailure,
		IP:            ip,
		UserAgent:     c.Request.UserAgent(),
		Method:        c.Request.Method,
		Path:          c.Request.URL.Path,
		Status:        http.StatusUnauthorized,
		Success:       false,
		FailureReason: reason,
		Duration:      0,
	}
	config.DB.Create(log)
}

// 辅助函数：记录注册成功
func recordRegisterSuccess(c *gin.Context, userID uint, studentID string) {
	log := &models.AuditLog{
		UserID:    userID,
		StudentID: studentID,
		Action:    models.ActionRegister,
		IP:        utils.GetIPFromRequest(c.Request),
		UserAgent: c.Request.UserAgent(),
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		Status:    http.StatusOK,
		Success:   true,
		Duration:  0,
	}
	config.DB.Create(log)
}

// 辅助函数：记录登出
func recordLogout(c *gin.Context, userID uint, studentID string) {
	log := &models.AuditLog{
		UserID:    userID,
		StudentID: studentID,
		Action:    models.ActionLogout,
		IP:        utils.GetIPFromRequest(c.Request),
		UserAgent: c.Request.UserAgent(),
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		Status:    http.StatusOK,
		Success:   true,
		Duration:  0,
	}
	config.DB.Create(log)
}

// getUint 安全转换
func getUint(val interface{}) uint {
	if v, ok := val.(uint); ok {
		return v
	}
	return 0
}

func getString(val interface{}) string {
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}
