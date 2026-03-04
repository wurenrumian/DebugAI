package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"time"

	"backend-go/config"
	"backend-go/logger"
	"backend-go/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EmailServiceIface defines the interface for email service operations
type EmailServiceIface interface {
	SendVerificationEmail(user *models.User) error
	GenerateVerificationToken() (string, error)
	VerifyToken(token string) (uint, error)
	IsEmailConfigured() bool
}

// EmailService handles email operations
type EmailService struct {
	DB *gorm.DB
}

// NewEmailService creates a new EmailService
func NewEmailService(db *gorm.DB) *EmailService {
	return &EmailService{
		DB: db,
	}
}

// GenerateVerificationToken generates a secure random token for email verification
func (s *EmailService) GenerateVerificationToken() (string, error) {
	// Generate 32 bytes (256 bits) of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate verification token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// SendVerificationEmail sends a verification email to the user
func (s *EmailService) SendVerificationEmail(user *models.User) error {
	// Generate token and expiration
	token, err := s.GenerateVerificationToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(24 * time.Hour)

	// Update user with verification token and timestamps
	now := time.Now()
	user.EmailVerificationToken = token
	user.EmailVerificationSentAt = &now
	user.EmailVerificationExpiresAt = &expiresAt

	if err := s.DB.Save(user).Error; err != nil {
		return fmt.Errorf("failed to save verification token: %w", err)
	}

	// Build verification link
	verificationLink := fmt.Sprintf("/auth/verify-email?token=%s", token)
	if config.Global.AppEnv == "production" {
		// TODO: Configure production frontend URL
		// verificationLink = fmt.Sprintf("%s/auth/verify-email?token=%s", config.Global.FrontendURL, token)
	}

	return s.sendEmail(user.Email, "邮箱验证", verificationLink, user.Username)
}

// SendVerificationEmailWithLink sends a verification email with a custom verification link
func (s *EmailService) SendVerificationEmailWithLink(user *models.User, verificationLink string) error {
	// Check if SMTP is configured
	if !s.IsEmailConfigured() {
		return fmt.Errorf("SMTP is not configured")
	}

	// Email content
	subject := "邮箱验证"
	body := fmt.Sprintf(`
尊敬的 %s，

请点击以下链接完成邮箱验证：

%s

该链接将在24小时后失效。

如果这不是您本人的操作，请忽略此邮件。

此致，
DebugAI 团队
`, user.Username, verificationLink)

	// Send email via SMTP
	addr := fmt.Sprintf("%s:%d", config.Global.SMTPHost, config.Global.SMTPPort)
	auth := smtp.PlainAuth("", config.Global.SMTPUsername, config.Global.SMTPPassword, config.Global.SMTPHost)

	to := []string{user.Email}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", user.Email, subject, body))

	if err := smtp.SendMail(addr, auth, config.Global.SMTPFrom, to, msg); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

// sendEmail helper method to send email
func (s *EmailService) sendEmail(recipient, subject, verificationLink, username string) error {
	// Check if SMTP is configured
	if !s.IsEmailConfigured() {
		return fmt.Errorf("SMTP is not configured")
	}

	body := fmt.Sprintf(`
尊敬的 %s，

请点击以下链接完成邮箱验证：

%s

该链接将在24小时后失效。

如果这不是您本人的操作，请忽略此邮件。

此致，
DebugAI 团队
`, username, verificationLink)

	addr := fmt.Sprintf("%s:%d", config.Global.SMTPHost, config.Global.SMTPPort)
	auth := smtp.PlainAuth("", config.Global.SMTPUsername, config.Global.SMTPPassword, config.Global.SMTPHost)

	to := []string{recipient}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", recipient, subject, body))

	return smtp.SendMail(addr, auth, config.Global.SMTPFrom, to, msg)
}

// VerifyToken verifies the verification token and returns the user ID if valid
func (s *EmailService) VerifyToken(token string) (uint, error) {
	var user models.User
	if err := s.DB.Where("email_verification_token = ? AND email_verification_expires_at > ?", token, time.Now()).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, fmt.Errorf("invalid or expired verification token")
		}
		return 0, err
	}

	return user.ID, nil
}

// IsEmailConfigured checks if SMTP email is properly configured
func (s *EmailService) IsEmailConfigured() bool {
	configured := config.Global.SMTPHost != "" &&
		config.Global.SMTPUsername != "" &&
		config.Global.SMTPPassword != "" &&
		config.Global.SMTPFrom != ""

	if !configured {
		logger.Logger.Warn("SMTP配置检查失败",
			zap.String("SMTPHost", config.Global.SMTPHost),
			zap.String("SMTPUsername", config.Global.SMTPUsername),
			zap.Bool("SMTPPasswordSet", config.Global.SMTPPassword != ""),
			zap.String("SMTPFrom", config.Global.SMTPFrom),
		)
	}

	return configured
}
