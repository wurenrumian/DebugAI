package controller

import (
	"backend-go/config"
	"backend-go/logger"
	"backend-go/models"
	"backend-go/service"
	"backend-go/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Register 用户注册（第一步：发送验证邮件，不创建用户）
func Register(c *gin.Context) {
	var input struct {
		StudentID string `json:"student_id" binding:"required,max=50"`
		Username  string `json:"username" binding:"required,min=2,max=100"`
		Password  string `json:"password" binding:"required,min=6,max=128"`
		Email     string `json:"email" binding:"required,email,max=255"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}

	// 获取客户端IP
	ip := utils.GetIPFromRequest(c.Request)

	// 频率限制检查
	if utils.GlobalSecurity != nil && !utils.GlobalSecurity.CheckRateLimit("register:"+ip, 10) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "注册频率过高，请稍后再试"})
		return
	}

	// 输入消毒
	input.StudentID = utils.SanitizeInput(input.StudentID)
	input.Username = utils.SanitizeInput(input.Username)
	input.Password = utils.SanitizeInput(input.Password)
	input.Email = utils.SanitizeInput(input.Email)

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
		c.JSON(http.StatusConflict, gin.H{"error": "学号不可用"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 检查邮箱是否已被注册
	var existingEmail models.User
	if err := config.DB.Where("email = ?", input.Email).First(&existingEmail).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "邮箱已被注册"})
		return
	}

	// 检查SMTP是否配置
	emailService := service.NewEmailService(config.DB)
	if !emailService.IsEmailConfigured() {
		logger.Logger.Error("SMTP未配置，无法发送验证邮件",
			zap.String("student_id", input.StudentID),
			zap.String("email", input.Email),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器邮件服务未配置，请联系管理员"})
		return
	}

	// 生成验证token（用于注册流程）
	verificationToken, err := emailService.GenerateVerificationToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证token失败"})
		return
	}

	// 计算过期时间（24小时）
	expiresAt := time.Now().Add(24 * time.Hour)

	// 将注册信息暂存到Redis（使用token作为key）
	ctx := c.Request.Context()
	registerKey := fmt.Sprintf("register_verify:%s", verificationToken)
	registerData := map[string]interface{}{
		"student_id": input.StudentID,
		"username":   input.Username,
		"password":   input.Password,
		"email":      input.Email,
		"expires_at": expiresAt.Format(time.RFC3339),
	}

	// 序列化数据为JSON存储
	jsonData, err := json.Marshal(registerData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部错误"})
		return
	}

	// 存储到Redis，设置24小时过期
	if err := config.RedisClient.Set(ctx, registerKey, jsonData, 24*time.Hour).Err(); err != nil {
		logger.Logger.Error("Redis存储注册信息失败",
			zap.Error(err),
			zap.String("key", registerKey),
			zap.String("student_id", input.StudentID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "存储注册信息失败"})
		return
	}
	logger.Logger.Info("Redis存储注册信息成功",
		zap.String("key", registerKey),
		zap.String("student_id", input.StudentID),
		zap.Duration("expires_in", 24*time.Hour),
	)

	// 发送验证邮件（验证链接包含token）
	// 必须配置 FRONTEND_URL，否则无法生成有效链接
	if config.Global.FrontendURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器配置错误：未设置前端URL"})
		return
	}
	verificationLink := fmt.Sprintf("%s/auth/verify-email?token=%s", config.Global.FrontendURL, verificationToken)

	if err := emailService.SendVerificationEmailWithLink(&models.User{
		Email: input.Email,
	}, verificationLink); err != nil {
		logger.Logger.Error("发送验证邮件失败",
			zap.Error(err),
			zap.String("email", input.Email),
			zap.String("link", verificationLink),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送验证邮件失败"})
		return
	}
	logger.Logger.Info("发送验证邮件成功",
		zap.String("email", input.Email),
		zap.String("link", verificationLink),
	)

	// 记录注册尝试审计日志
	go func() {
		log := &models.AuditLog{
			StudentID: input.StudentID,
			Action:    models.ActionRegister,
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    http.StatusOK,
			Success:   true,
			Extra:     "注册邮箱验证邮件已发送",
			Duration:  0,
		}
		config.DB.Create(log)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "注册验证邮件已发送，请查收邮件完成验证",
		"data": gin.H{
			"email": input.Email,
		},
	})
}

// VerifyEmail 邮箱验证（完成注册）
func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证token不能为空"})
		return
	}

	ctx := c.Request.Context()
	registerKey := fmt.Sprintf("register_verify:%s", token)
	jsonData, err := config.RedisClient.Get(ctx, registerKey).Bytes()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "验证链接无效或已过期"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	var registerData map[string]interface{}
	if err := json.Unmarshal(jsonData, &registerData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册信息损坏"})
		return
	}

	// 检查是否过期
	expiresAtStr, ok := registerData["expires_at"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册信息损坏"})
		return
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil || time.Now().After(expiresAt) {
		config.RedisClient.Del(ctx, registerKey)
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证链接已过期，请重新注册"})
		return
	}

	// 提取注册信息
	studentID, _ := registerData["student_id"].(string)
	username, _ := registerData["username"].(string)
	password, _ := registerData["password"].(string)
	email, _ := registerData["email"].(string)

	// 再次检查学号是否已被注册
	var existingUser models.User
	if err := config.DB.Where("student_id = ?", studentID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "学号已被注册"})
		return
	}

	// 再次检查邮箱是否已被注册
	if err := config.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "邮箱已被注册"})
		return
	}

	// 密码哈希
	cost := config.Global.BCryptCost
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 创建用户（邮箱已验证）
	user := models.User{
		StudentID:              studentID,
		Username:               username,
		Password:               string(hashedPassword),
		UserType:               models.TypeUser,
		Email:                  email,
		EmailVerified:          true,
		EmailVerificationToken: "",
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 删除Redis中的注册信息
	config.RedisClient.Del(ctx, registerKey)

	// 记录注册成功审计日志
	go recordRegisterSuccess(c, user.ID, user.StudentID)

	c.JSON(http.StatusOK, gin.H{
		"message": "邮箱验证成功，账号已创建",
		"data": gin.H{
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// Login 用户登录（移除邮箱验证检查）
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
		utils.GlobalSecurity.ClearCaptchaRequirement(input.StudentID)
	}

	var user models.User
	if err := config.DB.Where("student_id = ?", input.StudentID).First(&user).Error; err != nil {
		utils.GlobalSecurity.RecordLoginFailure(input.StudentID, ip)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "学号或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
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

	// Set HttpOnly cookie
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
	userID, _ := c.Get("user_id")
	studentID, _ := c.Get("student_id")

	c.SetCookie("auth_token", "", -1, "/", "", false, true)

	go recordLogout(c, getUint(userID), getString(studentID))

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// ResendVerificationEmail 重新发送验证邮件（注册流程）
func ResendVerificationEmail(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱不能为空"})
		return
	}

	// 检查邮箱是否已被注册（如果已注册则不能重新发送）
	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已被注册，无需验证"})
		return
	}

	// 检查SMTP是否配置
	emailService := service.NewEmailService(config.DB)
	if !emailService.IsEmailConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器邮件服务未配置"})
		return
	}

	// 必须配置 FRONTEND_URL
	if config.Global.FrontendURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器配置错误：未设置前端URL"})
		return
	}

	// 生成新的验证token
	verificationToken, err := emailService.GenerateVerificationToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证token失败"})
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	ctx := c.Request.Context()
	registerKey := fmt.Sprintf("register_verify:%s", verificationToken)
	registerData := map[string]interface{}{
		"email":      input.Email,
		"expires_at": expiresAt.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(registerData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部错误"})
		return
	}

	if err := config.RedisClient.Set(ctx, registerKey, jsonData, 24*time.Hour).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "存储注册信息失败"})
		return
	}

	verificationLink := fmt.Sprintf("/auth/verify-email?token=%s", verificationToken)
	if config.Global.FrontendURL != "" {
		verificationLink = fmt.Sprintf("%s/auth/verify-email?token=%s", config.Global.FrontendURL, verificationToken)
	}

	if err := emailService.SendVerificationEmailWithLink(&models.User{
		Email: input.Email,
	}, verificationLink); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送验证邮件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证邮件已发送"})
}

// 辅助函数保持不变
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
