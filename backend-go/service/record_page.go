package service

import (
	"strconv"

	"backend-go/models"
	"backend-go/service/cache"

	"gorm.io/gorm"
)

const (
	// DefaultRecordLimit 未传分页参数时最多返回的最近记录条数（保持旧接口兼容）
	DefaultRecordLimit = cache.DefaultRecordLimit
	// DefaultRecordPageSize 默认每页条数
	DefaultRecordPageSize = 20
	// MaxRecordPageSize 每页最大条数
	MaxRecordPageSize = 100
)

// RecordPage 历史记录分页参数；Page <= 0 表示未分页（返回最近 DefaultRecordLimit 条）
type RecordPage struct {
	Page     int
	PageSize int
}

// Enabled 是否启用分页
func (p RecordPage) Enabled() bool {
	return p.Page > 0
}

// ParseRecordPage 解析 page / page_size 查询参数
// 两者都为空时返回未分页；任一非法时返回 ok=false
func ParseRecordPage(pageStr, pageSizeStr string) (RecordPage, bool) {
	if pageStr == "" && pageSizeStr == "" {
		return RecordPage{}, true
	}
	page := 1
	pageSize := DefaultRecordPageSize
	if pageStr != "" {
		n, err := strconv.Atoi(pageStr)
		if err != nil || n < 1 {
			return RecordPage{}, false
		}
		page = n
	}
	if pageSizeStr != "" {
		n, err := strconv.Atoi(pageSizeStr)
		if err != nil || n < 1 || n > MaxRecordPageSize {
			return RecordPage{}, false
		}
		pageSize = n
	}
	return RecordPage{Page: page, PageSize: pageSize}, true
}

// queryRecordPage 对已带条件的查询执行计数 + 分页
func queryRecordPage(query *gorm.DB, page RecordPage) ([]models.AIRecord, int64, error) {
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	records := make([]models.AIRecord, 0)
	offset := (page.Page - 1) * page.PageSize
	if err := query.Session(&gorm.Session{}).
		Order("created_at DESC").Order("id DESC").
		Offset(offset).Limit(page.PageSize).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// queryRecentRecords 未分页时返回最近 DefaultRecordLimit 条
func queryRecentRecords(query *gorm.DB) ([]models.AIRecord, int64, error) {
	records := make([]models.AIRecord, 0)
	if err := query.Order("created_at DESC").Order("id DESC").Limit(DefaultRecordLimit).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, int64(len(records)), nil
}
