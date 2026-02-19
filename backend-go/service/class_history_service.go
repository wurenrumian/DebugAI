package service

import (
	"backend-go/config"
	"backend-go/models"
	"time"
)

// ClassRecordResponse 班级历史记录响应
type ClassRecordResponse struct {
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Data     []interface{} `json:"data"`
}

// GetClassStudentIDs 获取班级所有学生的 student_id 列表
func GetClassStudentIDs(classID uint) ([]string, error) {
	var members []models.ClassMember
	err := config.DB.Where("class_id = ? AND member_role = ?", classID, models.MemberRoleStudent).
		Find(&members).Error
	if err != nil {
		return nil, err
	}

	// 获取对应的 user 信息以获取 student_id
	var users []models.User
	userIDs := make([]uint, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}

	if len(userIDs) == 0 {
		return []string{}, nil
	}

	err = config.DB.Where("id IN ?", userIDs).Find(&users).Error
	if err != nil {
		return nil, err
	}

	studentIDs := make([]string, len(users))
	for i, u := range users {
		studentIDs[i] = u.StudentID
	}

	return studentIDs, nil
}

// GetClassDebugRecords 获取班级 debug 历史记录
// studentIDs 为 nil 时查询全班，为空数组时无结果，为有内容的数组时筛选指定学生
func GetClassDebugRecords(classID uint, studentIDs []string, startDate, endDate time.Time, page, pageSize int) (*ClassRecordResponse, error) {
	// 1. 如果未指定 studentIDs，获取班级所有学生 ID
	if studentIDs == nil {
		var err error
		studentIDs, err = GetClassStudentIDs(classID)
		if err != nil {
			return nil, err
		}

		if len(studentIDs) == 0 {
			return &ClassRecordResponse{
				Total:    0,
				Page:     page,
				PageSize: pageSize,
				Data:     []interface{}{},
			}, nil
		}
	} else if len(studentIDs) > 0 {
		// 2. 如果指定了 studentIDs，验证是否在班级中
		classStudentIDs, err := GetClassStudentIDs(classID)
		if err != nil {
			return nil, err
		}

		validStudentIDs := make([]string, 0)
		for _, sid := range studentIDs {
			found := false
			for _, csid := range classStudentIDs {
				if sid == csid {
					found = true
					break
				}
			}
			if found {
				validStudentIDs = append(validStudentIDs, sid)
			}
		}

		if len(validStudentIDs) == 0 {
			return &ClassRecordResponse{
				Total:    0,
				Page:     page,
				PageSize: pageSize,
				Data:     []interface{}{},
			}, nil
		}
		studentIDs = validStudentIDs
	}

	// 3. 构建查询 - 只查询 round_number > 0 的 debug 记录，且 conversation_id 以 'conv_' 或 'dbg_' 开头
	query := config.DB.Model(&models.AIRecord{}).Where("student_id IN ? AND round_number > 0 AND (conversation_id LIKE ? OR conversation_id LIKE ?)", studentIDs, "conv_%", "dbg_%")

	// 时间筛选
	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at <= ?", endDate.Add(24*time.Hour))
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var records []models.AIRecord
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, err
	}

	// 4. 关联查询学生信息
	studentIDSet := make(map[string]bool)
	for _, r := range records {
		studentIDSet[r.StudentID] = true
	}

	var users []models.User
	if len(studentIDSet) > 0 {
		var studentIDList []string
		for sid := range studentIDSet {
			studentIDList = append(studentIDList, sid)
		}
		config.DB.Where("student_id IN ?", studentIDList).Find(&users)
	}

	userMap := make(map[string]models.User)
	for _, u := range users {
		userMap[u.StudentID] = u
	}

	// 5. 转换结果 - 使用驼峰命名，与个人历史记录保持一致
	data := make([]interface{}, len(records))
	for i, r := range records {
		data[i] = map[string]interface{}{
			"ID":               r.ID,
			"CreatedAt":        r.CreatedAt,
			"UpdatedAt":        r.UpdatedAt,
			"DeletedAt":        r.DeletedAt,
			"conversation_id":  r.ConversationID,
			"student_id":       r.StudentID,
			"round_number":     r.RoundNumber,
			"role":             r.Role,
			"request_payload":  r.RequestPayload,
			"response_payload": r.ResponsePayload,
		}
	}

	return &ClassRecordResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Data:     data,
	}, nil
}

// GetClassEvaluateRecords 获取班级 evaluate 历史记录
// studentIDs 为 nil 时查询全班，为空数组时无结果，为有内容的数组时筛选指定学生
func GetClassEvaluateRecords(classID uint, studentIDs []string, startDate, endDate time.Time, page, pageSize int) (*ClassRecordResponse, error) {
	// 1. 如果未指定 studentIDs，获取班级所有学生 ID
	if studentIDs == nil {
		var err error
		studentIDs, err = GetClassStudentIDs(classID)
		if err != nil {
			return nil, err
		}

		if len(studentIDs) == 0 {
			return &ClassRecordResponse{
				Total:    0,
				Page:     page,
				PageSize: pageSize,
				Data:     []interface{}{},
			}, nil
		}
	} else if len(studentIDs) > 0 {
		// 2. 如果指定了 studentIDs，验证是否在班级中
		classStudentIDs, err := GetClassStudentIDs(classID)
		if err != nil {
			return nil, err
		}

		validStudentIDs := make([]string, 0)
		for _, sid := range studentIDs {
			found := false
			for _, csid := range classStudentIDs {
				if sid == csid {
					found = true
					break
				}
			}
			if found {
				validStudentIDs = append(validStudentIDs, sid)
			}
		}

		if len(validStudentIDs) == 0 {
			return &ClassRecordResponse{
				Total:    0,
				Page:     page,
				PageSize: pageSize,
				Data:     []interface{}{},
			}, nil
		}
		studentIDs = validStudentIDs
	}

	// 3. 构建查询 - 使用 AIRecord 表，通过 conversation_id 过滤 evaluate 类型
	query := config.DB.Model(&models.AIRecord{}).Where("student_id IN ? AND conversation_id LIKE 'eval_%'", studentIDs)

	// 时间筛选
	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at <= ?", endDate.Add(24*time.Hour))
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var records []models.AIRecord
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, err
	}

	// 4. 关联查询学生信息
	studentIDSet := make(map[string]bool)
	for _, r := range records {
		studentIDSet[r.StudentID] = true
	}

	var users []models.User
	if len(studentIDSet) > 0 {
		var studentIDList []string
		for sid := range studentIDSet {
			studentIDList = append(studentIDList, sid)
		}
		config.DB.Where("student_id IN ?", studentIDList).Find(&users)
	}

	userMap := make(map[string]models.User)
	for _, u := range users {
		userMap[u.StudentID] = u
	}

	// 5. 转换结果 - 使用驼峰命名，与个人历史记录保持一致
	data := make([]interface{}, len(records))
	for i, r := range records {
		data[i] = map[string]interface{}{
			"ID":               r.ID,
			"CreatedAt":        r.CreatedAt,
			"UpdatedAt":        r.UpdatedAt,
			"DeletedAt":        r.DeletedAt,
			"conversation_id":  r.ConversationID,
			"student_id":       r.StudentID,
			"round_number":     r.RoundNumber,
			"role":             r.Role,
			"request_payload":  r.RequestPayload,
			"response_payload": r.ResponsePayload,
		}
	}

	return &ClassRecordResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Data:     data,
	}, nil
}

// GetClassRecommendRecords 获取班级 recommend 历史记录
// studentIDs 为 nil 时查询全班，为空数组时无结果，为有内容的数组时筛选指定学生
func GetClassRecommendRecords(classID uint, studentIDs []string, startDate, endDate time.Time, page, pageSize int) (*ClassRecordResponse, error) {
	// 1. 如果未指定 studentIDs，获取班级所有学生 ID
	if studentIDs == nil {
		var err error
		studentIDs, err = GetClassStudentIDs(classID)
		if err != nil {
			return nil, err
		}

		if len(studentIDs) == 0 {
			return &ClassRecordResponse{
				Total:    0,
				Page:     page,
				PageSize: pageSize,
				Data:     []interface{}{},
			}, nil
		}
	} else if len(studentIDs) > 0 {
		// 2. 如果指定了 studentIDs，验证是否在班级中
		classStudentIDs, err := GetClassStudentIDs(classID)
		if err != nil {
			return nil, err
		}

		validStudentIDs := make([]string, 0)
		for _, sid := range studentIDs {
			found := false
			for _, csid := range classStudentIDs {
				if sid == csid {
					found = true
					break
				}
			}
			if found {
				validStudentIDs = append(validStudentIDs, sid)
			}
		}

		if len(validStudentIDs) == 0 {
			return &ClassRecordResponse{
				Total:    0,
				Page:     page,
				PageSize: pageSize,
				Data:     []interface{}{},
			}, nil
		}
		studentIDs = validStudentIDs
	}

	// 3. 构建查询 - 使用 AIRecord 表，通过 conversation_id 过滤 recommend 类型
	query := config.DB.Model(&models.AIRecord{}).Where("student_id IN ? AND conversation_id LIKE 'rec_%'", studentIDs)

	// 时间筛选
	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at <= ?", endDate.Add(24*time.Hour))
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var records []models.AIRecord
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, err
	}

	// 4. 关联查询学生信息
	studentIDSet := make(map[string]bool)
	for _, r := range records {
		studentIDSet[r.StudentID] = true
	}

	var users []models.User
	if len(studentIDSet) > 0 {
		var studentIDList []string
		for sid := range studentIDSet {
			studentIDList = append(studentIDList, sid)
		}
		config.DB.Where("student_id IN ?", studentIDList).Find(&users)
	}

	userMap := make(map[string]models.User)
	for _, u := range users {
		userMap[u.StudentID] = u
	}

	// 5. 组装返回数据
	data := make([]interface{}, len(records))
	for i, r := range records {
		data[i] = map[string]interface{}{
			"ID":               r.ID,
			"CreatedAt":        r.CreatedAt,
			"UpdatedAt":        r.UpdatedAt,
			"DeletedAt":        r.DeletedAt,
			"conversation_id":  r.ConversationID,
			"student_id":       r.StudentID,
			"round_number":     r.RoundNumber,
			"role":             r.Role,
			"request_payload":  r.RequestPayload,
			"response_payload": r.ResponsePayload,
		}
	}

	return &ClassRecordResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Data:     data,
	}, nil
}
