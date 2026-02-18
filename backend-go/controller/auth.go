package controller

import (
	"backend-go/config"
	"backend-go/models"
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Register 用户注册
func Register(c *gin.Context) {
	var input struct {
		StudentID string `json:"student_id" binding:"required"`
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}

	// 密码哈希
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	user := models.User{
		StudentID: input.StudentID,
		Username:  input.Username,
		Password:  string(hashedPassword),
		UserType:  models.TypeUser,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "学号已存在"})
		return
	}

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

	var user models.User
	// 按学号查询
	if err := config.DB.Where("student_id = ?", input.StudentID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "学号或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "学号或密码错误"})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.StudentID, user.UserType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成Token失败"})
		return
	}

	// Set HttpOnly cookie
	c.SetCookie("auth_token", token, 24*3600, "/", "", false, true)

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
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}
