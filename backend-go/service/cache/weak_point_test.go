package cache

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"backend-go/config"
	"backend-go/logger"
	"backend-go/models"
	"backend-go/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func init() {
	if logger.Logger == nil {
		logger.Logger = zap.NewNop()
	}
}

func setupCacheTest(t *testing.T) (*gorm.DB, *miniredis.Miniredis, *RedisCache) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:cache_%s?mode=memory&cache=shared", t.Name())),
		&gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&models.User{}, &models.Class{}, &models.ClassMember{}, &models.WeakPoint{}, &models.UserWeakPoint{}); err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	prev := config.RedisClient
	config.RedisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		config.RedisClient.Close()
		config.RedisClient = prev
	})
	return db, mr, NewRedisCache()
}

// seedClasses: 班级1 有 A1、A2，班级2 有 B1；每人一个不同关键词的薄弱点
func seedClasses(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Create(&models.User{StudentID: "T1", Username: "teacher", Password: "x", Email: "t@x"})
	db.Omit("Creator").Create(&models.Class{ID: 1, ClassName: "c1", CreatedBy: 1})
	db.Omit("Creator").Create(&models.Class{ID: 2, ClassName: "c2", CreatedBy: 1})
	students := map[string]uint{"A1": 1, "A2": 1, "B1": 2}
	now := time.Now()
	for sid, classID := range students {
		u := models.User{StudentID: sid, Username: "u" + sid, Password: "x", Email: sid + "@x"}
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		db.Omit("User").Create(&models.ClassMember{ClassID: classID, UserID: u.ID, MemberRole: models.MemberRoleStudent})
		wp := models.WeakPoint{Keyword: "kw_" + sid, Category: "逻辑类"}
		db.Create(&wp)
		db.Create(&models.UserWeakPoint{StudentID: sid, WeakPointID: wp.ID, Count: 1, RecordDate: now})
	}
	// 老师也是班级1成员，但不应计入
	db.Omit("User").Create(&models.ClassMember{ClassID: 1, UserID: 1, MemberRole: models.MemberRoleTeacher})
}

func keywords(result []map[string]interface{}) []string {
	var out []string
	for _, r := range result {
		out = append(out, r["keyword"].(string))
	}
	sort.Strings(out)
	return out
}

func TestClassWeakPointsFilteredByClass(t *testing.T) {
	db, _, rc := setupCacheTest(t)
	seedClasses(t, db)
	wpc := NewWeakPointCache(rc, db)
	ctx := context.Background()

	// 不传学生筛选：只能返回班级1学生的数据（修复前会返回全系统数据）
	all, err := wpc.GetClassWeakPoints(ctx, 1, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := keywords(all); fmt.Sprint(got) != "[kw_A1 kw_A2]" {
		t.Fatalf("class 1 unfiltered = %v", got)
	}

	// 带筛选：只返回指定学生；不属于班级的学生被忽略
	filtered, err := wpc.GetClassWeakPoints(ctx, 1, []string{"A2", "B1"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := keywords(filtered); fmt.Sprint(got) != "[kw_A2]" {
		t.Fatalf("class 1 filtered = %v", got)
	}

	// 再次不带筛选查询，不应被带筛选的缓存结果污染
	again, _ := wpc.GetClassWeakPoints(ctx, 1, nil, nil, nil)
	if got := keywords(again); fmt.Sprint(got) != "[kw_A1 kw_A2]" {
		t.Fatalf("cache poisoned: %v", got)
	}

	// 只包含其他班级学生的筛选 -> 空
	none, _ := wpc.GetClassWeakPoints(ctx, 1, []string{"B1"}, nil, nil)
	if len(none) != 0 {
		t.Fatalf("expected empty, got %v", none)
	}
}

func TestWeakPointsClassKeyIncludesFilters(t *testing.T) {
	k1 := utils.WeakPointsClassKey(1, "2026-01-01", "2026-01-31", nil)
	k2 := utils.WeakPointsClassKey(1, "2026-01-01", "2026-01-31", []string{"A1"})
	k3 := utils.WeakPointsClassKey(1, "2026-01-01", "2026-01-31", []string{"A2", "A1"})
	k4 := utils.WeakPointsClassKey(1, "2026-01-01", "2026-01-31", []string{"A1", "A2", "A1"})
	k5 := utils.WeakPointsClassKey(1, "2026-01-02", "2026-01-31", nil)
	if k1 == k2 || k1 == k5 || k2 == k3 {
		t.Fatalf("keys must differ: %s %s %s %s", k1, k2, k3, k5)
	}
	if k3 != k4 {
		t.Fatalf("key must be order/duplicate independent: %s vs %s", k3, k4)
	}
	if utils.StudentFilterHash(nil) != "all" {
		t.Fatal("empty filter should hash to all")
	}
}

func TestInvalidateUserWeakPointsAlsoInvalidatesClassCache(t *testing.T) {
	db, mr, rc := setupCacheTest(t)
	seedClasses(t, db)
	wpc := NewWeakPointCache(rc, db)
	ctx := context.Background()

	if _, err := wpc.GetClassWeakPoints(ctx, 1, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := wpc.GetClassWeakPoints(ctx, 2, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	countClassKeys := func(classID uint) int {
		n := 0
		for _, k := range mr.Keys() {
			if len(k) > 0 && matchPrefix(k, fmt.Sprintf("weak_points:class:%d:", classID)) {
				n++
			}
		}
		return n
	}
	if countClassKeys(1) == 0 || countClassKeys(2) == 0 {
		t.Fatalf("expected cached class keys, got %v", mr.Keys())
	}

	wpc.InvalidateUserWeakPoints("A1")

	deadline := time.Now().Add(3 * time.Second)
	for countClassKeys(1) != 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if countClassKeys(1) != 0 {
		t.Fatalf("class 1 cache should be invalidated, keys=%v", mr.Keys())
	}
	if countClassKeys(2) == 0 {
		t.Fatal("class 2 cache should be kept")
	}
}

func matchPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestInvalidatePatternUsesScan(t *testing.T) {
	_, mr, rc := setupCacheTest(t)
	ctx := context.Background()
	for i := 0; i < 1200; i++ {
		mr.Set(fmt.Sprintf("weak_points:class:7:k%d", i), "v")
	}
	mr.Set("weak_points:class:70:keep", "v")

	if err := rc.InvalidatePattern(ctx, utils.WeakPointsClassPattern(7)); err != nil {
		t.Fatal(err)
	}
	keys := mr.Keys()
	if len(keys) != 1 || keys[0] != "weak_points:class:70:keep" {
		t.Fatalf("unexpected remaining keys: %d", len(keys))
	}
}
