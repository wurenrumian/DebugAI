package controller

import (
	"backend-go/config"
	"backend-go/logger"
	"backend-go/models"
	"backend-go/service"
	"backend-go/utils"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 频率限制配置
const (
	registerLimitPerIP      = 30 // 每个 IP 每小时最多注册请求数（机房/NAT 场景需适当放宽）
	registerLimitPerEmail   = 5  // 每个邮箱每小时最多注册请求数（防止向任意邮箱轰炸）
	resendLimitPerIP        = 10 // 每个 IP 每小时最多重发验证邮件次数
	resendLimitPerEmail     = 3  // 每个邮箱每小时最多重发验证邮件次数
	forgotPasswordPerIP     = 10 // 每个 IP 每小时最多申请重置密码次数
	forgotPasswordPerEmail  = 3  // 每个邮箱每小时最多申请重置密码次数
	registerVerifyTTL       = 24 * time.Hour
	passwordResetTTL        = 1 * time.Hour
	bcryptMaxPasswordLength = 72 // bcrypt 只支持 72 字节以内的密码
)

// pendingRegistration 待邮箱验证的注册信息（暂存于 Redis）
// 注意：只保存 bcrypt 哈希，绝不保存明文密码
type pendingRegistration struct {
	StudentID    string `json:"student_id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Email        string `json:"email"`
	ExpiresAt    string `json:"expires_at"`
}

// newPendingRegistration 构造待验证注册信息（在此处完成密码哈希）
func newPendingRegistration(studentID, username, password, email string, cost int, expiresAt time.Time) (*pendingRegistration, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return nil, err
	}
	return &pendingRegistration{
		StudentID:    studentID,
		Username:     username,
		PasswordHash: string(hash),
		Email:        email,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	}, nil
}

// valid 校验注册信息完整性（旧版本遗留或损坏的数据一律视为无效）
func (p *pendingRegistration) valid(now time.Time) bool {
	if p.StudentID == "" || p.Username == "" || p.Email == "" || p.PasswordHash == "" {
		return false
	}
	if _, err := bcrypt.Cost([]byte(p.PasswordHash)); err != nil {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, p.ExpiresAt)
	if err != nil || now.After(expiresAt) {
		return false
	}
	return true
}

func registerVerifyKey(token string) string {
	return fmt.Sprintf("register_verify:%s", token)
}

// registerPendingEmailKey 邮箱 -> 待验证注册 token 的索引，用于重发验证邮件
func registerPendingEmailKey(email string) string {
	return fmt.Sprintf("register_pending_email:%s", email)
}

// requestMeta 在请求 goroutine 中提取的请求信息。
// gin.Context 在请求结束后会被复用，不能在后台 goroutine 中访问。
type requestMeta struct {
	IP        string
	UserAgent string
	Method    string
	Path      string
}

func newRequestMeta(c *gin.Context) requestMeta {
	return requestMeta{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
	}
}

// saveAuditLogAsync 异步写入审计日志（log 需在请求 goroutine 中构造完毕）
func saveAuditLogAsync(log *models.AuditLog) {
	go func() {
		if err := config.DB.Create(log).Error; err != nil {
			logger.Logger.Warn("写入审计日志失败", zap.Error(err), zap.String("action", log.Action))
		}
	}()
}

// newAuditLog 构造成功操作的审计日志
func newAuditLog(meta requestMeta, action string, userID uint, studentID, extra string) *models.AuditLog {
	return &models.AuditLog{
		UserID:    userID,
		StudentID: studentID,
		Action:    action,
		IP:        meta.IP,
		UserAgent: meta.UserAgent,
		Method:    meta.Method,
		Path:      meta.Path,
		Status:    http.StatusOK,
		Success:   true,
		Extra:     extra,
	}
}

// rejectRateLimited 返回 429
func rejectRateLimited(c *gin.Context, msg string, wait time.Duration) {
	if wait > 0 {
		seconds := int(math.Ceil(wait.Seconds()))
		c.Header("Retry-After", strconv.Itoa(seconds))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": msg, "retry_after": seconds})
		return
	}
	c.JSON(http.StatusTooManyRequests, gin.H{"error": msg})
}

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

	meta := newRequestMeta(c)

	// 频率限制检查（按 IP，不能包含学号等客户端可控字段，否则可轻易绕过）
	if ok, wait := utils.GlobalSecurity.Allow("register:ip:"+meta.IP, registerLimitPerIP, time.Hour); !ok {
		rejectRateLimited(c, "请求频率过高，请稍后再试", wait)
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
	if len(input.Password) > bcryptMaxPasswordLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度不能超过72个字符"})
		return
	}

	// 按邮箱限流，防止利用注册接口向任意邮箱发送大量邮件
	if ok, wait := utils.GlobalSecurity.Allow("register:email:"+input.Email, registerLimitPerEmail, time.Hour); !ok {
		rejectRateLimited(c, "该邮箱请求过于频繁，请稍后再试", wait)
		return
	}

	// 统一返回模糊错误，防止账户枚举
	genericConflictError := "注册失败，请稍后重试或使用其他信息"

	// 检查学号是否已存在
	var existingUser models.User
	if err := config.DB.Where("student_id = ?", input.StudentID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": genericConflictError})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Logger.Error("查询学号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 检查邮箱是否已被注册
	var existingEmail models.User
	if err := config.DB.Where("email = ?", input.Email).First(&existingEmail).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": genericConflictError})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Logger.Error("查询邮箱失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 检查SMTP是否配置
	emailService := service.NewEmailService(config.DB)
	if !emailService.IsEmailConfigured() {
		logger.Logger.Error("SMTP未配置，无法发送验证邮件",
			zap.String("student_id", input.StudentID),
			zap.String("email", maskEmail(input.Email)),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器邮件服务未配置，请联系管理员"})
		return
	}

	// 必须配置 FRONTEND_URL，否则无法生成有效链接
	if config.Global.FrontendURL == "" {
		logger.Logger.Error("未配置 FRONTEND_URL，无法生成验证链接")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器配置错误，请联系管理员"})
		return
	}

	// 生成验证token（用于注册流程）
	verificationToken, err := emailService.GenerateVerificationToken()
	if err != nil {
		logger.Logger.Error("生成验证token失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 密码在暂存前即完成 bcrypt 哈希，Redis 中不保存明文密码
	pending, err := newPendingRegistration(input.StudentID, input.Username, input.Password, input.Email,
		config.Global.BCryptCost, time.Now().Add(registerVerifyTTL))
	if err != nil {
		logger.Logger.Error("密码加密失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	jsonData, err := json.Marshal(pending)
	if err != nil {
		logger.Logger.Error("序列化注册信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	ctx := c.Request.Context()
	registerKey := registerVerifyKey(verificationToken)
	emailKey := registerPendingEmailKey(input.Email)

	// 同一邮箱重复注册时，使旧的待验证链接失效
	if oldToken, err := config.RedisClient.Get(ctx, emailKey).Result(); err == nil && oldToken != "" {
		config.RedisClient.Del(ctx, registerVerifyKey(oldToken))
	}

	// 存储注册信息与邮箱索引，设置24小时过期
	_, err = config.RedisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, registerKey, jsonData, registerVerifyTTL)
		pipe.Set(ctx, emailKey, verificationToken, registerVerifyTTL)
		return nil
	})
	if err != nil {
		logger.Logger.Error("Redis存储注册信息失败",
			zap.Error(err),
			zap.String("student_id", input.StudentID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 发送验证邮件（验证链接包含token）
	verificationLink := fmt.Sprintf("%s/auth/verify-email?token=%s", config.Global.FrontendURL, verificationToken)
	if err := emailService.SendVerificationEmailWithLink(&models.User{
		Email:    input.Email,
		Username: input.Username,
	}, verificationLink); err != nil {
		logger.Logger.Error("发送验证邮件失败",
			zap.Error(err),
			zap.String("email", maskEmail(input.Email)),
		)
		// 发送失败则清理暂存数据，保持状态一致
		config.RedisClient.Del(ctx, registerKey, emailKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送验证邮件失败"})
		return
	}
	logger.Logger.Info("发送验证邮件成功",
		zap.String("email", maskEmail(input.Email)),
		zap.String("student_id", input.StudentID),
	)

	// 记录注册尝试审计日志
	saveAuditLogAsync(newAuditLog(meta, models.ActionRegister, 0, input.StudentID, "注册邮箱验证邮件已发送"))

	c.JSON(http.StatusOK, gin.H{
		"message": "注册验证邮件已发送，请查收邮件完成验证",
		"data": gin.H{
			"email": input.Email,
		},
	})
}

// loadPendingRegistration 读取并校验待验证注册信息；数据无效时会清理该 key
func loadPendingRegistration(c *gin.Context, token string) (*pendingRegistration, error) {
	ctx := c.Request.Context()
	jsonData, err := config.RedisClient.Get(ctx, registerVerifyKey(token)).Bytes()
	if err != nil {
		return nil, err
	}
	var p pendingRegistration
	if err := json.Unmarshal(jsonData, &p); err != nil || !p.valid(time.Now()) {
		config.RedisClient.Del(ctx, registerVerifyKey(token))
		return nil, errInvalidPendingRegistration
	}
	return &p, nil
}

var errInvalidPendingRegistration = errors.New("invalid pending registration")

// VerifyEmail 邮箱验证（完成注册）
func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证token不能为空"})
		return
	}

	meta := newRequestMeta(c)
	ctx := c.Request.Context()

	pending, err := loadPendingRegistration(c, token)
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusNotFound, gin.H{"error": "验证链接无效或已过期"})
		return
	} else if errors.Is(err, errInvalidPendingRegistration) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证链接已失效，请重新注册"})
		return
	} else if err != nil {
		logger.Logger.Error("读取注册信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 再次检查学号/邮箱是否已被注册（防止竞态条件）
	var existingUser models.User
	if err := config.DB.Where("student_id = ?", pending.StudentID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "注册失败，该学号已被占用"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Logger.Error("查询学号失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}
	if err := config.DB.Where("email = ?", pending.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "注册失败，该邮箱已被占用"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Logger.Error("查询邮箱失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 创建用户（邮箱已验证），直接使用注册时生成的密码哈希
	user := models.User{
		StudentID:              pending.StudentID,
		Username:               pending.Username,
		Password:               pending.PasswordHash,
		UserType:               models.TypeUser,
		Email:                  pending.Email,
		EmailVerified:          true,
		EmailVerificationToken: "",
	}

	if err := config.DB.Create(&user).Error; err != nil {
		// 并发验证时可能触发唯一约束，此时返回冲突而不是 500
		var count int64
		if cerr := config.DB.Model(&models.User{}).
			Where("student_id = ? OR email = ?", pending.StudentID, pending.Email).
			Count(&count).Error; cerr == nil && count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "注册失败，学号或邮箱已被占用"})
			return
		}
		logger.Logger.Error("创建用户失败", zap.Error(err), zap.String("student_id", pending.StudentID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 删除Redis中的注册信息
	config.RedisClient.Del(ctx, registerVerifyKey(token), registerPendingEmailKey(pending.Email))

	// 记录注册成功审计日志
	saveAuditLogAsync(newAuditLog(meta, models.ActionRegister, user.ID, user.StudentID, ""))

	c.JSON(http.StatusOK, gin.H{
		"message": "邮箱验证成功，账号已创建",
		"data": gin.H{
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// compareDummyPassword 学号不存在时也执行一次 bcrypt 比对，
// 使响应时间与“学号存在但密码错误”一致，防止通过时间差枚举学号
func compareDummyPassword(password string) {
	dummyHashOnce.Do(func() {
		cost := bcrypt.DefaultCost
		if config.Global != nil && config.Global.BCryptCost > 0 {
			cost = config.Global.BCryptCost
		}
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("debugai-dummy-password-for-timing"), cost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

// Login 用户登录
func Login(c *gin.Context) {
	var input struct {
		StudentID string `json:"student_id" binding:"required,max=50"`
		Password  string `json:"password" binding:"required,max=128"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入学号和密码"})
		return
	}

	ip := c.ClientIP()

	// 登录失败退避：按 (IP, 学号) 与 IP 两个维度限制，不会因他人恶意尝试而锁死账号
	if ok, wait := utils.GlobalSecurity.CheckLoginAllowed(ip, input.StudentID); !ok {
		rejectRateLimited(c, "登录失败次数过多，请稍后再试", wait)
		return
	}

	var user models.User
	if err := config.DB.Where("student_id = ?", input.StudentID).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Logger.Error("登录查询用户失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			return
		}
		compareDummyPassword(input.Password)
		utils.GlobalSecurity.RecordLoginFailure(ip, input.StudentID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "学号或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		utils.GlobalSecurity.RecordLoginFailure(ip, input.StudentID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "学号或密码错误"})
		return
	}

	// 登录成功，清除该 IP + 学号 的失败记录
	utils.GlobalSecurity.RecordLoginSuccess(ip, input.StudentID)

	token, err := utils.GenerateToken(user.ID, user.StudentID, user.UserType, user.TokenVersion)
	if err != nil {
		logger.Logger.Error("生成Token失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// Set HttpOnly cookie
	expirySeconds := config.Global.JWTExpiry * 3600
	secureFlag := config.Global.AppEnv == "production"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", token, expirySeconds, "/", "", secureFlag, true)

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

// Logout 用户登出（需认证）
// 递增 token_version 使该用户已签发的 JWT 全部失效（AuthMiddleware 会校验版本号）
func Logout(c *gin.Context) {
	userID, ok := currentUserIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	studentIDVal, _ := c.Get("student_id")
	studentID := getString(studentIDVal)
	meta := newRequestMeta(c)

	// 无论后续是否成功，都清除 Cookie
	secureFlag := config.Global.AppEnv == "production"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", "", -1, "/", "", secureFlag, true)

	if err := config.DB.Model(&models.User{}).Where("id = ?", userID).
		UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		logger.Logger.Error("登出时更新token_version失败", zap.Error(err), zap.Uint("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登出失败，请稍后重试"})
		return
	}

	saveAuditLogAsync(newAuditLog(meta, models.ActionLogout, userID, studentID, ""))

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// ResendVerificationEmail 重新发送验证邮件（注册流程）
// 只能为已存在的待验证注册申请重发，复用原验证链接
func ResendVerificationEmail(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email,max=255"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱不能为空"})
		return
	}
	input.Email = utils.SanitizeInput(input.Email)

	ip := c.ClientIP()

	// 频率限制：按 IP 与邮箱两个维度，防止邮件轰炸
	if ok, wait := utils.GlobalSecurity.Allow("resend_verification:ip:"+ip, resendLimitPerIP, time.Hour); !ok {
		rejectRateLimited(c, "请求频率过高，请稍后再试", wait)
		return
	}
	if ok, wait := utils.GlobalSecurity.Allow("resend_verification:email:"+input.Email, resendLimitPerEmail, time.Hour); !ok {
		rejectRateLimited(c, "该邮箱请求过于频繁，请稍后再试", wait)
		return
	}

	// 检查邮箱是否已被注册（如果已注册则不能重新发送）
	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已被注册，无需验证"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Logger.Error("查询邮箱失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 查找该邮箱的待验证注册申请
	ctx := c.Request.Context()
	notFoundMsg := "未找到待验证的注册申请，请重新注册"
	token, err := config.RedisClient.Get(ctx, registerPendingEmailKey(input.Email)).Result()
	if errors.Is(err, redis.Nil) || (err == nil && token == "") {
		c.JSON(http.StatusNotFound, gin.H{"error": notFoundMsg})
		return
	} else if err != nil {
		logger.Logger.Error("读取待验证注册索引失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	pending, err := loadPendingRegistration(c, token)
	if errors.Is(err, redis.Nil) || errors.Is(err, errInvalidPendingRegistration) || (err == nil && pending.Email != input.Email) {
		config.RedisClient.Del(ctx, registerPendingEmailKey(input.Email))
		c.JSON(http.StatusNotFound, gin.H{"error": notFoundMsg})
		return
	} else if err != nil {
		logger.Logger.Error("读取注册信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
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
		logger.Logger.Error("未配置 FRONTEND_URL，无法生成验证链接")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器配置错误，请联系管理员"})
		return
	}

	// 复用原 token（有效期不变）
	verificationLink := fmt.Sprintf("%s/auth/verify-email?token=%s", config.Global.FrontendURL, token)
	if err := emailService.SendVerificationEmailWithLink(&models.User{
		Email:    pending.Email,
		Username: pending.Username,
	}, verificationLink); err != nil {
		logger.Logger.Error("重发验证邮件失败", zap.Error(err), zap.String("email", maskEmail(input.Email)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送验证邮件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证邮件已发送"})
}

// ForgotPassword 请求重置密码
func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email,max=255"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式错误"})
		return
	}

	meta := newRequestMeta(c)

	// 频率限制检查：同一 IP 每小时最多 10 次，同一邮箱每小时最多 3 次
	if ok, wait := utils.GlobalSecurity.Allow("forgot_password:ip:"+meta.IP, forgotPasswordPerIP, time.Hour); !ok {
		rejectRateLimited(c, "请求频率过高，请稍后再试", wait)
		return
	}
	if ok, wait := utils.GlobalSecurity.Allow("forgot_password:email:"+input.Email, forgotPasswordPerEmail, time.Hour); !ok {
		rejectRateLimited(c, "请求频率过高，请稍后再试", wait)
		return
	}

	// 检查邮箱是否已注册
	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 安全考虑：无论邮箱是否存在，都返回相同提示
			c.JSON(http.StatusOK, gin.H{
				"message": "重置邮件已发送",
				"data": gin.H{
					"email": input.Email,
				},
			})
			return
		}
		logger.Logger.Error("查询邮箱失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 检查SMTP是否配置
	emailService := service.NewEmailService(config.DB)
	if !emailService.IsEmailConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器邮件服务未配置"})
		return
	}

	if config.Global.FrontendURL == "" {
		logger.Logger.Error("未配置 FRONTEND_URL，无法生成重置链接")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器配置错误，请联系管理员"})
		return
	}

	// 生成重置token
	resetToken, err := emailService.GenerateVerificationToken()
	if err != nil {
		logger.Logger.Error("生成重置token失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 将重置信息存入 Redis (1 小时过期，一次性使用)
	ctx := c.Request.Context()
	resetKey := fmt.Sprintf("password_reset:%s", resetToken)
	resetData := map[string]interface{}{
		"student_id": user.StudentID,
		"email":      user.Email,
		"expires_at": time.Now().Add(passwordResetTTL).Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(resetData)
	if err != nil {
		logger.Logger.Error("序列化重置信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	if err := config.RedisClient.Set(ctx, resetKey, jsonData, passwordResetTTL).Err(); err != nil {
		logger.Logger.Error("存储重置信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", config.Global.FrontendURL, resetToken)

	// 构造邮件内容
	subject := "[DebugAI] 密码重置请求"
	body := fmt.Sprintf(`
尊敬的 %s，

我们收到了您的密码重置请求。请点击以下链接设置新密码：

%s

该链接将在 1 小时后失效。

如果这不是您本人的操作，请忽略此邮件并确保您的账户安全。

此致，
DebugAI 团队
`, user.Username, resetLink)

	if err := emailService.SendPlainEmail(user.Email, subject, body); err != nil {
		logger.Logger.Error("发送重置邮件失败", zap.Error(err))
		config.RedisClient.Del(ctx, resetKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送重置邮件失败"})
		return
	}

	// 记录审计日志
	saveAuditLogAsync(newAuditLog(meta, models.ActionPasswordReset, user.ID, user.StudentID, "密码重置邮件已发送"))

	c.JSON(http.StatusOK, gin.H{
		"message": "重置邮件已发送",
		"data": gin.H{
			"email": input.Email,
		},
	})
}

// ResetPassword 执行密码重置
func ResetPassword(c *gin.Context) {
	var input struct {
		Token           string `json:"token" binding:"required,max=128"`
		NewPassword     string `json:"new_password" binding:"required,min=6,max=128"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整或密码过短"})
		return
	}

	if input.NewPassword != input.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "两次输入的密码不一致"})
		return
	}

	// 密码强度校验
	if result := utils.ValidateInput("password", input.NewPassword); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "密码不符合安全要求",
			"details": result.Errors,
		})
		return
	}
	if len(input.NewPassword) > bcryptMaxPasswordLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度不能超过72个字符"})
		return
	}

	meta := newRequestMeta(c)

	// 原子地读取并删除重置 token，保证一次性使用（并发请求只有一个能成功）。
	// 不再校验发起请求时的 IP：token 本身是只发送到邮箱的随机秘密，
	// 用户在其他设备（如手机）打开邮件也应能完成重置。
	ctx := c.Request.Context()
	resetKey := fmt.Sprintf("password_reset:%s", input.Token)
	jsonData, err := config.RedisClient.GetDel(ctx, resetKey).Bytes()
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重置令牌无效或已过期"})
		return
	} else if err != nil {
		logger.Logger.Error("读取重置信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	var resetData map[string]interface{}
	if err := json.Unmarshal(jsonData, &resetData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重置令牌无效或已过期"})
		return
	}

	studentID, _ := resetData["student_id"].(string)
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重置令牌无效或已过期"})
		return
	}

	// 查询用户信息
	var user models.User
	if err := config.DB.Where("student_id = ?", studentID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		logger.Logger.Error("查询用户失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误，请重新申请重置"})
		return
	}

	// 密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), config.Global.BCryptCost)
	if err != nil {
		logger.Logger.Error("密码加密失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误，请重新申请重置"})
		return
	}

	// 更新用户密码并增加 token_version（使已签发的 JWT 全部失效）
	updates := map[string]interface{}{
		"password":      string(hashedPassword),
		"token_version": gorm.Expr("token_version + 1"),
	}

	if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
		logger.Logger.Error("更新密码失败", zap.Error(err), zap.Uint("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密码失败，请重新申请重置"})
		return
	}

	// 记录审计日志
	saveAuditLogAsync(newAuditLog(meta, models.ActionPasswordReset, user.ID, user.StudentID, "密码重置成功"))

	c.JSON(http.StatusOK, gin.H{
		"message": "密码重置成功，请重新登录",
		"data": gin.H{
			"student_id": user.StudentID,
		},
	})
}

func getString(val interface{}) string {
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	if len(parts[0]) <= 2 {
		return strings.Repeat("*", len(parts[0])) + "@" + parts[1]
	}
	masked := parts[0][:2] + strings.Repeat("*", len(parts[0])-2)
	return masked + "@" + parts[1]
}
