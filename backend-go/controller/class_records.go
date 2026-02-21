package controller

import (
	"backend-go/models"
	"backend-go/service"
	"encoding/json"
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

	// 解析查询参数 - 支持多个 student_ids (JSON数组格式)
	var studentIDs []string

	// 优先使用 student_ids (JSON数组格式)
	studentIDsStr := c.Query("student_ids")
	if studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 student_ids 格式"})
			return
		}
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 page 参数"})
		return
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 page_size 参数"})
		return
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
	var result *service.ClassRecordResponse

	// 根据是否有 studentIDs 调用不同的服务方法
	if len(studentIDs) > 0 {
		result, err = service.GetClassDebugRecords(uint(classID), studentIDs, startDate, endDate, page, pageSize)
	} else {
		// 传 nil 表示查询全班
		result, err = service.GetClassDebugRecords(uint(classID), nil, startDate, endDate, page, pageSize)
	}

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

	// 解析查询参数 - 支持多个 student_ids (JSON数组格式)
	var studentIDs []string
	studentIDsStr := c.Query("student_ids")
	if studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 student_ids 格式"})
			return
		}
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 page 参数"})
		return
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 page_size 参数"})
		return
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
	var result *service.ClassRecordResponse
	if len(studentIDs) > 0 {
		result, err = service.GetClassEvaluateRecords(uint(classID), studentIDs, startDate, endDate, page, pageSize)
	} else {
		result, err = service.GetClassEvaluateRecords(uint(classID), nil, startDate, endDate, page, pageSize)
	}
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

	// 解析查询参数 - 支持多个 student_ids (JSON数组格式)
	var studentIDs []string
	studentIDsStr := c.Query("student_ids")
	if studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 student_ids 格式"})
			return
		}
	}

	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 page 参数"})
		return
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 page_size 参数"})
		return
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
	var result *service.ClassRecordResponse
	if len(studentIDs) > 0 {
		result, err = service.GetClassRecommendRecords(uint(classID), studentIDs, startDate, endDate, page, pageSize)
	} else {
		result, err = service.GetClassRecommendRecords(uint(classID), nil, startDate, endDate, page, pageSize)
	}
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

	// 解析查询参数 - 支持多个 student_ids (JSON数组格式)
	var studentIDs []string
	studentIDsStr := c.Query("student_ids")
	if studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 student_ids 格式"})
			return
		}
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
	var result *service.ClassRecordResponse
	if len(studentIDs) > 0 {
		result, err = service.GetClassDebugRecords(uint(classID), studentIDs, startDate, endDate, page, pageSize)
	} else {
		result, err = service.GetClassDebugRecords(uint(classID), nil, startDate, endDate, page, pageSize)
	}
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
		"filters": map[string]interface{}{"class_id": classID, "student_ids": studentIDs, "start_date": startDateStr, "end_date": endDateStr},
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

	// 解析查询参数 - 支持多个 student_ids (JSON数组格式)
	var studentIDs []string
	studentIDsStr := c.Query("student_ids")
	if studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 student_ids 格式"})
			return
		}
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
	var result *service.ClassRecordResponse
	if len(studentIDs) > 0 {
		result, err = service.GetClassEvaluateRecords(uint(classID), studentIDs, startDate, endDate, page, pageSize)
	} else {
		result, err = service.GetClassEvaluateRecords(uint(classID), nil, startDate, endDate, page, pageSize)
	}
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
		"filters": map[string]interface{}{"class_id": classID, "student_ids": studentIDs, "start_date": startDateStr, "end_date": endDateStr},
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

	// 解析查询参数 - 支持多个 student_ids (JSON数组格式)
	var studentIDs []string
	studentIDsStr := c.Query("student_ids")
	if studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 student_ids 格式"})
			return
		}
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
	var result *service.ClassRecordResponse
	if len(studentIDs) > 0 {
		result, err = service.GetClassRecommendRecords(uint(classID), studentIDs, startDate, endDate, page, pageSize)
	} else {
		result, err = service.GetClassRecommendRecords(uint(classID), nil, startDate, endDate, page, pageSize)
	}
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
		"filters": map[string]interface{}{"class_id": classID, "student_ids": studentIDs, "start_date": startDateStr, "end_date": endDateStr},
		"data":    result.Data,
	}

	c.JSON(http.StatusOK, exportData)
}
