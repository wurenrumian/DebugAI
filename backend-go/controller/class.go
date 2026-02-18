package controller

import (
	"backend-go/config"
	"backend-go/models"
	"backend-go/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateClass 创建班级（仅 admin 可执行）
func CreateClass(c *gin.Context) {
	// 获取用户类型
	userType, exists := c.Get("user_type")
	if !exists || userType != models.TypeAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以创建班级"})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	var input struct {
		ClassName string `json:"class_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供班级名称"})
		return
	}

	// 创建班级
	class := models.Class{
		ClassName: input.ClassName,
		CreatedBy: userID.(uint),
	}

	if err := config.DB.Create(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建班级失败"})
		return
	}

	// 创建者自动成为班级老师，并标记为创建者
	member := models.ClassMember{
		ClassID:    class.ID,
		UserID:     userID.(uint),
		MemberRole: models.MemberRoleTeacher,
		IsCreator:  true, // 标记为班级创建者
	}
	config.DB.Create(&member)

	c.JSON(http.StatusOK, gin.H{
		"message": "班级创建成功",
		"data":    class,
	})
}

// GetClasses 获取班级列表（包含创建者信息）
func GetClasses(c *gin.Context) {
	var classes []models.Class
	// 预加载创建者信息
	if err := config.DB.Preload("Creator").Find(&classes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取班级列表失败"})
		return
	}

	// 转换为前端需要的格式
	type ClassResponse struct {
		ID          uint      `json:"id"`
		Name        string    `json:"name"`
		CreatedBy   uint      `json:"created_by"`
		CreatorName string    `json:"creator_name"`
		CreatedAt   time.Time `json:"created_at"`
	}

	response := make([]ClassResponse, 0, len(classes))
	for _, class := range classes {
		creatorName := ""
		if class.Creator.ID != 0 {
			creatorName = class.Creator.Username
		}
		response = append(response, ClassResponse{
			ID:          class.ID,
			Name:        class.ClassName,
			CreatedBy:   class.CreatedBy,
			CreatorName: creatorName,
			CreatedAt:   class.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// JoinClass 加入班级
func JoinClass(c *gin.Context) {
	classID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	// 检查班级是否存在
	var class models.Class
	if err := config.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}

	// 检查是否已经是成员
	var existingMember models.ClassMember
	if err := config.DB.Where("class_id = ? AND user_id = ?", classID, userID).First(&existingMember).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "已经是班级成员"})
		return
	}

	// 加入班级，默认为 student
	member := models.ClassMember{
		ClassID:    uint(classID),
		UserID:     userID.(uint),
		MemberRole: models.MemberRoleStudent,
	}

	if err := config.DB.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加入班级失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "加入班级成功", "data": member})
}

// GetClassDetail 获取班级详情（包含创建者信息）
func GetClassDetail(c *gin.Context) {
	classID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	var class models.Class
	// 预加载创建者信息
	if err := config.DB.Preload("Creator").First(&class, classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取班级详情失败"})
		}
		return
	}

	// 转换为前端需要的格式
	type ClassResponse struct {
		ID          uint      `json:"id"`
		Name        string    `json:"name"`
		CreatedBy   uint      `json:"created_by"`
		CreatorName string    `json:"creator_name"`
		CreatedAt   time.Time `json:"created_at"`
	}

	creatorName := ""
	if class.Creator.ID != 0 {
		creatorName = class.Creator.Username
	}

	response := ClassResponse{
		ID:          class.ID,
		Name:        class.ClassName,
		CreatedBy:   class.CreatedBy,
		CreatorName: creatorName,
		CreatedAt:   class.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetClassMembers 获取班级成员（包含用户信息）
func GetClassMembers(c *gin.Context) {
	classID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	var members []models.ClassMember
	// 预加载用户信息
	if err := config.DB.Preload("User").Where("class_id = ?", classID).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取成员列表失败"})
		return
	}

	// 转换为前端需要的格式
	type MemberResponse struct {
		ID        uint   `json:"id"`
		ClassID   uint   `json:"class_id"`
		UserID    uint   `json:"user_id"`
		StudentID string `json:"student_id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		IsCreator bool   `json:"is_creator"`
	}

	response := make([]MemberResponse, 0, len(members))
	for _, member := range members {
		response = append(response, MemberResponse{
			ID:        member.ID,
			ClassID:   member.ClassID,
			UserID:    member.UserID,
			StudentID: member.User.StudentID,
			Username:  member.User.Username,
			Role:      member.MemberRole,
			IsCreator: member.IsCreator,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetMyClasses 获取用户加入的班级（包含创建者信息）
func GetMyClasses(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	var members []models.ClassMember
	if err := config.DB.Where("user_id = ?", userID).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取班级失败"})
		return
	}

	// 关联查询班级信息（预加载创建者）
	var classes []models.Class
	for _, m := range members {
		var class models.Class
		if err := config.DB.Preload("Creator").First(&class, m.ClassID).Error; err == nil {
			class.CreatedAt = class.CreatedAt // 保持原样
			classes = append(classes, class)
		}
	}

	// 转换为前端需要的格式（与 GetClasses 一致）
	type ClassResponse struct {
		ID          uint      `json:"id"`
		Name        string    `json:"name"`
		CreatedBy   uint      `json:"created_by"`
		CreatorName string    `json:"creator_name"`
		CreatedAt   time.Time `json:"created_at"`
	}

	response := make([]ClassResponse, 0, len(classes))
	for _, class := range classes {
		creatorName := ""
		if class.Creator.ID != 0 {
			creatorName = class.Creator.Username
		}
		response = append(response, ClassResponse{
			ID:          class.ID,
			Name:        class.ClassName,
			CreatedBy:   class.CreatedBy,
			CreatorName: creatorName,
			CreatedAt:   class.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// AddMembers 添加班级成员（批量）- 仅班级管理员(teacher)可操作
func AddMembers(c *gin.Context) {
	classID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}
	currentUserID := userID.(uint) // 转换为 uint 类型供后续使用

	// 权限检查：只有班级管理员(teacher或ta)才能添加成员
	if !service.IsClassAdmin(currentUserID, uint(classID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有班级管理员可以添加成员"})
		return
	}

	var input struct {
		StudentIDs []string `json:"student_ids" binding:"required"`
		MemberRole string   `json:"member_role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供学生ID列表和角色"})
		return
	}

	// 校验角色合法性
	if input.MemberRole != models.MemberRoleTeacher && input.MemberRole != models.MemberRoleTA && input.MemberRole != models.MemberRoleStudent {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色"})
		return
	}

	// 限制角色分配权限：只有创建者或admin可分配teacher/ta角色，其他管理员只能分配student
	if input.MemberRole != models.MemberRoleStudent {
		// 检查当前用户是否为班级创建者（使用已转换的 currentUserID）
		isCreator := service.IsClassCreator(currentUserID, uint(classID))
		// 检查当前用户是否为系统admin
		userType, _ := c.Get("user_type")
		isAdmin := userType.(string) == models.TypeAdmin

		if !isCreator && !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "只有班级创建者或管理员可以分配教师/助教角色"})
			return
		}
	}

	// 查询所有student_id对应的用户
	var users []models.User
	if len(input.StudentIDs) > 0 {
		if err := config.DB.Where("student_id IN ?", input.StudentIDs).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
			return
		}
	}

	// 构建已存在的用户ID映射
	existUsersMap := make(map[string]uint) // student_id -> user_id
	for _, u := range users {
		existUsersMap[u.StudentID] = u.ID
	}

	// 记录结果
	type Result struct {
		StudentID string `json:"student_id"`
		Status    string `json:"status"` // "success" or "not_found"
		Message   string `json:"message,omitempty"`
	}
	results := make([]Result, 0, len(input.StudentIDs))

	// 批量添加成员
	for _, studentID := range input.StudentIDs {
		if userID, ok := existUsersMap[studentID]; ok {
			// 检查是否已是成员
			var existing models.ClassMember
			if err := config.DB.Where("class_id = ? AND user_id = ?", classID, userID).First(&existing).Error; err == nil {
				// 如果是已存在的成员，检查是否尝试修改创建者角色
				if existing.IsCreator {
					// 创建者角色不可修改
					results = append(results, Result{
						StudentID: studentID,
						Status:    "skipped",
						Message:   "创建者角色不可修改",
					})
					continue
				}
				// 如果角色没有变化，跳过
				if existing.MemberRole == input.MemberRole {
					results = append(results, Result{
						StudentID: studentID,
						Status:    "skipped",
						Message:   "角色未变化",
					})
					continue
				}
				// 角色有变化：如果要提升为 teacher/ta，需要额外权限检查
				if input.MemberRole != models.MemberRoleStudent {
					// 使用函数开头保存的 currentUserID
					isCreator := service.IsClassCreator(currentUserID, uint(classID))
					// 检查当前用户是否为系统admin
					userType, _ := c.Get("user_type")
					isAdmin := userType.(string) == models.TypeAdmin

					if !isCreator && !isAdmin {
						results = append(results, Result{
							StudentID: studentID,
							Status:    "error",
							Message:   "只有班级创建者或管理员可以提升成员角色",
						})
						continue
					}
				}
				// 更新现有成员的角色
				existing.MemberRole = input.MemberRole
				if err := config.DB.Save(&existing).Error; err != nil {
					results = append(results, Result{
						StudentID: studentID,
						Status:    "error",
						Message:   "更新角色失败",
					})
					continue
				}
				results = append(results, Result{
					StudentID: studentID,
					Status:    "success",
					Message:   "角色已更新",
				})
				continue
			}

			// 新增成员：检查是否尝试创建新的 teacher/ta（已在前面权限检查中处理）
			member := models.ClassMember{
				ClassID:    uint(classID),
				UserID:     userID,
				MemberRole: input.MemberRole,
				IsCreator:  false, // 新增成员不可能是创建者
			}
			if err := config.DB.Create(&member).Error; err != nil {
				results = append(results, Result{
					StudentID: studentID,
					Status:    "error",
					Message:   "添加失败",
				})
				continue
			}
			results = append(results, Result{
				StudentID: studentID,
				Status:    "success",
			})
		} else {
			results = append(results, Result{
				StudentID: studentID,
				Status:    "not_found",
				Message:   "用户不存在",
			})
		}
	}

	// 统计
	successCount := 0
	notFoundCount := 0
	skippedCount := 0
	for _, r := range results {
		switch r.Status {
		case "success":
			successCount++
		case "not_found":
			notFoundCount++
		case "skipped":
			skippedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "批量添加完成",
		"summary": gin.H{
			"success":   successCount,
			"not_found": notFoundCount,
			"skipped":   skippedCount,
		},
		"details": results,
	})
}

// RemoveMembers 移除班级成员（批量）- 仅班级管理员(teacher)可操作
func RemoveMembers(c *gin.Context) {
	classID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}
	currentUserID := userID.(uint) // 转换为 uint 类型供后续使用

	// 权限检查：只有班级管理员(teacher或ta)才能移除成员
	if !service.IsClassAdmin(currentUserID, uint(classID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有班级管理员可以移除成员"})
		return
	}

	// 获取当前用户在班级中的角色
	currentUserRole := service.GetUserRoleInClass(currentUserID, uint(classID))

	var input struct {
		StudentIDs []string `json:"student_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供学生ID列表"})
		return
	}

	// 助教权限限制：只能移除学生
	if currentUserRole == models.MemberRoleTA {
		for _, studentID := range input.StudentIDs {
			// 查询该学生的用户ID
			var targetUser models.User
			if err := config.DB.Where("student_id = ?", studentID).First(&targetUser).Error; err != nil {
				continue
			}
			// 查询该学生在班级中的角色
			targetUserRole := service.GetUserRoleInClass(targetUser.ID, uint(classID))
			if targetUserRole != models.MemberRoleStudent {
				c.JSON(http.StatusForbidden, gin.H{"error": "助教只能移除学生，不能移除教师或助教"})
				return
			}
		}
	}

	// 查询所有student_id对应的用户
	var users []models.User
	if len(input.StudentIDs) > 0 {
		if err := config.DB.Where("student_id IN ?", input.StudentIDs).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
			return
		}
	}

	// 构建用户ID映射
	userIDMap := make(map[string]uint)
	for _, u := range users {
		userIDMap[u.StudentID] = u.ID
	}

	// 记录结果
	type Result struct {
		StudentID string `json:"student_id"`
		Status    string `json:"status"` // "success", "not_found", "not_member"
		Message   string `json:"message,omitempty"`
	}
	results := make([]Result, 0, len(input.StudentIDs))

	// 批量移除成员
	for _, studentID := range input.StudentIDs {
		if uid, ok := userIDMap[studentID]; ok {
			// 检查是否是成员
			var member models.ClassMember
			if err := config.DB.Where("class_id = ? AND user_id = ?", classID, uid).First(&member).Error; err != nil {
				results = append(results, Result{
					StudentID: studentID,
					Status:    "not_member",
					Message:   "不是班级成员",
				})
				continue
			}

			// 不允许移除老师自己
			if member.MemberRole == models.MemberRoleTeacher && uid == currentUserID {
				results = append(results, Result{
					StudentID: studentID,
					Status:    "error",
					Message:   "不能移除自己",
				})
				continue
			}

			// 创建者保护：禁止移除班级创建者（即使角色被降级）
			if member.IsCreator {
				results = append(results, Result{
					StudentID: studentID,
					Status:    "error",
					Message:   "班级创建者不可移除",
				})
				continue
			}

			if err := config.DB.Delete(&member).Error; err != nil {
				results = append(results, Result{
					StudentID: studentID,
					Status:    "error",
					Message:   "移除失败",
				})
				continue
			}
			results = append(results, Result{
				StudentID: studentID,
				Status:    "success",
			})
		} else {
			results = append(results, Result{
				StudentID: studentID,
				Status:    "not_found",
				Message:   "用户不存在",
			})
		}
	}

	// 统计
	successCount := 0
	notFoundCount := 0
	notMemberCount := 0
	for _, r := range results {
		switch r.Status {
		case "success":
			successCount++
		case "not_found":
			notFoundCount++
		case "not_member":
			notMemberCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "批量移除完成",
		"summary": gin.H{
			"success":    successCount,
			"not_found":  notFoundCount,
			"not_member": notMemberCount,
		},
		"details": results,
	})
}
