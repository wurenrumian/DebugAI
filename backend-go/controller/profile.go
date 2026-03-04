package controller

import (
	"backend-go/config"
	"backend-go/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	studentID, _ := c.Get("student_id")

	var user models.User
	if err := config.DB.Where("student_id = ?", studentID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"student_id":     user.StudentID,
			"username":       user.Username,
			"user_type":      user.UserType,
			"email":          user.Email,
			"email_verified": user.EmailVerified,
		},
	})
}
