package controller

import (
	"backend-go/models"
	"backend-go/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetClassDebugRecords 获取班级 debug 历史记录
// GET /api/v1/classes/:id/records/debug
func GetClassDebugRecords(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := strconv.ParseUint(classIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 检查权限
	userType, _ := c.Get("user_type")
	isAdmin := userType == models.TypeAdmin
	hasAccess, _ := service.CanAccessClassData(userID.(uint), uint(classID))

	if !isAdmin && !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限访问该班级数据"})
		return
	}

	// 解析查询参数
	studentID := c.Query("student_id")
	var studentIDPtr *string
	if studentID != "" {
		studentIDPtr = &studentID
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 解析时间
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
			return
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
	}

	// 调用服务层
	result, err := service.GetClassDebugRecords(uint(classID), studentIDPtr, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetClassEvaluateRecords 获取班级 evaluate 历史记录
// GET /api/v1/classes/:id/records/evaluate
func GetClassEvaluateRecords(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := strconv.ParseUint(classIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 检查权限
	userType, _ := c.Get("user_type")
	isAdmin := userType == models.TypeAdmin
	hasAccess, _ := service.CanAccessClassData(userID.(uint), uint(classID))

	if !isAdmin && !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限访问该班级数据"})
		return
	}

	// 解析查询参数
	studentID := c.Query("student_id")
	var studentIDPtr *string
	if studentID != "" {
		studentIDPtr = &studentID
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 解析时间
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
			return
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
	}

	// 调用服务层
	result, err := service.GetClassEvaluateRecords(uint(classID), studentIDPtr, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetClassRecommendRecords 获取班级 recommend 历史记录
// GET /api/v1/classes/:id/records/recommend
func GetClassRecommendRecords(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := strconv.ParseUint(classIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 检查权限
	userType, _ := c.Get("user_type")
	isAdmin := userType == models.TypeAdmin
	hasAccess, _ := service.CanAccessClassData(userID.(uint), uint(classID))

	if !isAdmin && !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限访问该班级数据"})
		return
	}

	// 解析查询参数
	studentID := c.Query("student_id")
	var studentIDPtr *string
	if studentID != "" {
		studentIDPtr = &studentID
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 解析时间
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
			return
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
	}

	// 调用服务层
	result, err := service.GetClassRecommendRecords(uint(classID), studentIDPtr, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExportClassDebugRecords 导出班级 debug 历史记录
// GET /api/v1/classes/:id/records/debug/export
func ExportClassDebugRecords(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := strconv.ParseUint(classIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 检查权限
	userType, _ := c.Get("user_type")
	isAdmin := userType == models.TypeAdmin
	hasAccess, _ := service.CanAccessClassData(userID.(uint), uint(classID))

	if !isAdmin && !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限访问该班级数据"})
		return
	}

	// 解析查询参数
	studentID := c.Query("student_id")
	var studentIDPtr *string
	if studentID != "" {
		studentIDPtr = &studentID
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")
	// 导出默认获取更多数据
	page := 1
	pageSize := 10000 // 最大导出10000条

	// 解析时间
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
			return
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
	}

	// 调用服务层
	result, err := service.GetClassDebugRecords(uint(classID), studentIDPtr, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=debug_history.json")

	// 返回 JSON 格式的导出数据
	exportData := map[string]interface{}{
		"total":   result.Total,
		"filters": map[string]interface{}{"class_id": classID, "student_id": studentID, "start_date": startDateStr, "end_date": endDateStr},
		"data":    result.Data,
	}

	c.JSON(http.StatusOK, exportData)
}

// ExportClassEvaluateRecords 导出班级 evaluate 历史记录
// GET /api/v1/classes/:id/records/evaluate/export
func ExportClassEvaluateRecords(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := strconv.ParseUint(classIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 检查权限
	userType, _ := c.Get("user_type")
	isAdmin := userType == models.TypeAdmin
	hasAccess, _ := service.CanAccessClassData(userID.(uint), uint(classID))

	if !isAdmin && !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限访问该班级数据"})
		return
	}

	// 解析查询参数
	studentID := c.Query("student_id")
	var studentIDPtr *string
	if studentID != "" {
		studentIDPtr = &studentID
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")
	// 导出默认获取更多数据
	page := 1
	pageSize := 10000 // 最大导出10000条

	// 解析时间
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
			return
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
	}

	// 调用服务层
	result, err := service.GetClassEvaluateRecords(uint(classID), studentIDPtr, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=evaluate_history.json")

	// 返回 JSON 格式的导出数据
	exportData := map[string]interface{}{
		"total":   result.Total,
		"filters": map[string]interface{}{"class_id": classID, "student_id": studentID, "start_date": startDateStr, "end_date": endDateStr},
		"data":    result.Data,
	}

	c.JSON(http.StatusOK, exportData)
}

// ExportClassRecommendRecords 导出班级 recommend 历史记录
// GET /api/v1/classes/:id/records/recommend/export
func ExportClassRecommendRecords(c *gin.Context) {
	classIDStr := c.Param("id")
	classID, err := strconv.ParseUint(classIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的班级ID"})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 检查权限
	userType, _ := c.Get("user_type")
	isAdmin := userType == models.TypeAdmin
	hasAccess, _ := service.CanAccessClassData(userID.(uint), uint(classID))

	if !isAdmin && !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限访问该班级数据"})
		return
	}

	// 解析查询参数
	studentID := c.Query("student_id")
	var studentIDPtr *string
	if studentID != "" {
		studentIDPtr = &studentID
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")
	// 导出默认获取更多数据
	page := 1
	pageSize := 10000 // 最大导出10000条

	// 解析时间
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
			return
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
	}

	// 调用服务层
	result, err := service.GetClassRecommendRecords(uint(classID), studentIDPtr, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=recommend_history.json")

	// 返回 JSON 格式的导出数据
	exportData := map[string]interface{}{
		"total":   result.Total,
		"filters": map[string]interface{}{"class_id": classID, "student_id": studentID, "start_date": startDateStr, "end_date": endDateStr},
		"data":    result.Data,
	}

	c.JSON(http.StatusOK, exportData)
}
