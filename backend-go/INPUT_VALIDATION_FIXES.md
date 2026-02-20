# 输入验证问题修复方案

> 分析日期: 2025-02-20  
> 分析人员: Roo (软件架构师)

---

## 📋 问题总览

| 序号 | 问题类型         | 严重程度 | 需要修复 |
| ---- | ---------------- | -------- | -------- |
| 1    | 日期参数注入     | MEDIUM   | ✅ 是     |
| 2    | 超长字符串未验证 | MEDIUM   | ✅ 是     |
| 3    | 类型混淆参数     | LOW      | ✅ 是     |
| 4    | 管理员权限缓存   | MEDIUM   | ✅ 是     |

**说明**: "会话所有权绕过" 经代码审查为误报，代码已正确实现权限验证，不包含在本方案中。

---

## 🔍 详细分析与修复

### 1. 日期参数注入

**问题描述**: `GET /api/v1/ai/weak_points/class` 接口接收无效日期格式时，静默忽略错误并返回 200，而非 400。

**受影响文件**: [`backend-go/controller/ai_controller.go`](backend-go/controller/ai_controller.go:363-374)

**当前问题代码**:
```go
var startDate, endDate *time.Time
if startDateStr := c.Query("start_date"); startDateStr != "" {
    if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
        startDate = &t
    }  // 解析失败时静默忽略
}
// ... 类似处理 endDate
```

**修复方案**:
```go
var startDate, endDate time.Time
var hasStartDate, hasEndDate bool

if startDateStr := c.Query("start_date"); startDateStr != "" {
    t, err := time.Parse("2006-01-02", startDateStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式，请使用 YYYY-MM-DD 格式"})
        return
    }
    startDate = t
    hasStartDate = true
}

if endDateStr := c.Query("end_date"); endDateStr != "" {
    t, err := time.Parse("2006-01-02", endDateStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式，请使用 YYYY-MM-DD 格式"})
        return
    }
    endDate = t
    hasEndDate = true
}

// 传递给服务层
var startDatePtr, endDatePtr *time.Time
if hasStartDate {
    startDatePtr = &startDate
}
if hasEndDate {
    endDatePtr = &endDate
}
```

**调用位置**: 第 419 行 `ctrl.AIService.GetClassWeakPoints(classID, studentIDs, startDatePtr, endDatePtr)`

---

### 2. 超长字符串未验证

**问题描述**: 注册接口未限制 username 长度，允许 1000+ 字符存入数据库。

**受影响文件**: [`backend-go/controller/auth.go`](backend-go/controller/auth.go:15-20)

**当前问题代码**:
```go
type input struct {
    StudentID string `json:"student_id" binding:"required"`
    Username  string `json:"username" binding:"required"`  // 无长度限制
    Password  string `json:"password" binding:"required"`
}
```

**修复方案 A** (推荐 - 使用 validator 标签):
```go
type RegisterInput struct {
    StudentID string `json:"student_id" binding:"required,max=50"`
    Username  string `json:"username" binding:"required,min=2,max=100"`
    Password  string `json:"password" binding:"required,min=8,max=128"`
}
```

**修复方案 B** (手动验证):
```go
func Register(c *gin.Context) {
    var input struct {
        StudentID string `json:"student_id" binding:"required"`
        Username  string `json:"username" binding:"required"`
        Password  string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整或格式错误"})
        return
    }

    // 手动验证长度
    if len(input.Username) > 100 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度不能超过100个字符"})
        return
    }
    if len(input.Password) < 8 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少8个字符"})
        return
    }
    // ... 继续
}
```

**数据库层加固** (建议同步执行):
```go
type User struct {
    gorm.Model
    StudentID string `gorm:"unique;not null;size:50" json:"student_id"`
    UserType  string `gorm:"default:user;size:20" json:"user_type"`
    Username  string `gorm:"not null;size:100" json:"username"`  // 添加 size
    Password  string `gorm:"not null;size:255" json:"-"`
}
```

---

### 3. 类型混淆参数

**问题描述**: 分页参数 `page` 和 `page_size` 使用 `strconv.Atoi` 转换失败时静默修正为默认值，而非返回 400 错误。

**受影响文件**:
- [`backend-go/controller/class_records.go`](backend-go/controller/class_records.go:55-63) - GetClassDebugRecords
- [`backend-go/controller/class_records.go`](backend-go/controller/class_records.go:140-148) - GetClassEvaluateRecords
- [`backend-go/controller/class_records.go`](backend-go/controller/class_records.go:200-208) - GetClassRecommendRecords
- [`backend-go/controller/ai_controller.go`](backend-go/controller/ai_controller.go:419) - GetClassWeakPoints

**当前问题代码** (以 class_records.go 为例):
```go
page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

if page < 1 {
    page = 1
}
if pageSize < 1 || pageSize > 100 {
    pageSize = 20
}
```

**修复方案**:
```go
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
```

---

### 4. 管理员权限缓存问题

**问题描述**: 数据库中的 `user_type` 变更后，已发放的 JWT token 不会自动失效，直到过期或用户重新登录。

**受影响模块**:
- [`backend-go/controller/auth.go`](backend-go/controller/auth.go:82) - token 生成
- [`backend-go/middleware/auth.go`](backend-go/middleware/auth.go) - token 验证
- [`backend-go/controller/class.go`](backend-go/controller/class.go:18) - 权限检查
- 所有依赖 `user_type` 的接口

**安全影响**:
- 🔴 管理员权限被撤销后，旧 token 仍可访问管理员接口
- 🔴 普通用户权限提升后，需等待重新登录才能生效

**推荐修复方案**: Token Version 机制

**步骤 1**: 修改 User 模型，添加 `token_version` 字段

```go
// backend-go/models/user.go
type User struct {
    gorm.Model
    StudentID    string `gorm:"unique;not null;size:50" json:"student_id"`
    UserType     string `gorm:"default:user;size:20" json:"user_type"`
    Username     string `gorm:"not null;size:100" json:"username"`
    Password     string `gorm:"not null;size:255" json:"-"`
    TokenVersion int    `gorm:"default:0" json:"-"` // 新增字段
}
```

**步骤 2**: 修改 token 生成，包含版本号

```go
// backend-go/utils/jwt.go
func GenerateToken(userID uint, studentID, userType string, tokenVersion int) (string, error) {
    claims := &Claims{
        ID:           userID,
        StudentID:    studentID,
        UserType:     userType,
        TokenVersion: tokenVersion, // 新增
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
            IssuedAt:  time.Now().Unix(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(GetJWTSecret()))
}
```

```go
// backend-go/controller/auth.go - Login 函数
token, err := utils.GenerateToken(user.ID, user.StudentID, user.UserType, user.TokenVersion)
```

**步骤 3**: 修改中间件，验证 token 版本

```go
// backend-go/middleware/auth.go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... 获取并解析 token
        claims, err := utils.ParseToken(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
            c.Abort()
            return
        }

        // 查询数据库获取最新 token_version
        var user models.User
        if err := config.DB.Where("id = ?", claims.ID).First(&user).Error; err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
            c.Abort()
            return
        }

        // 验证 token 版本
        if claims.TokenVersion != user.TokenVersion {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token已失效，请重新登录"})
            c.Abort()
            return
        }

        // 使用数据库中的最新 user_type
        c.Set("student_id", user.StudentID)
        c.Set("user_type", user.UserType)
        c.Set("user_id", user.ID)

        c.Next()
    }
}
```

**步骤 4**: 修改用户类型更新逻辑，递增版本号

```go
// 在任何修改 user_type 的地方
func UpdateUserType(userID uint, newType string) error {
    return config.DB.Model(&models.User{}).
        Where("id = ?", userID).
        Updates(map[string]interface{}{
            "user_type":     newType,
            "token_version": gorm.Expr("token_version + 1"),
        }).Error
}
```

**步骤 5**: 数据库迁移

```sql
-- 添加 token_version 字段
ALTER TABLE users ADD COLUMN token_version INTEGER DEFAULT 0;

-- 为现有用户初始化版本号
UPDATE users SET token_version = 1 WHERE token_version = 0;
```

---

## 🛠️ 修复优先级

| 优先级 | 问题             | 修复难度 | 影响范围         | 建议完成时间 |
| ------ | ---------------- | -------- | ---------------- | ------------ |
| P0     | 日期参数注入     | 低       | 班级薄弱点查询   | 1 天内       |
| P0     | 超长字符串未验证 | 低       | 用户注册         | 1 天内       |
| P1     | 类型混淆参数     | 低       | 班级历史记录查询 | 2 天内       |
| P2     | 管理员权限缓存   | 中       | 所有管理员接口   | 1 周内       |

---

## 📝 完整修复清单

### 立即修复 (本周内)

- [ ] **修复日期参数验证** - 修改 `ai_controller.go:363-374`
- [ ] **添加 username 长度限制** - 修改 `auth.go` 并执行数据库迁移
- [ ] **严格验证分页参数** - 修改 `class_records.go` 所有相关函数

### 短期修复 (1 周内)

- [ ] **实现 Token Version 机制**
  - [ ] 修改 User 模型添加 `token_version`
  - [ ] 修改 `utils/jwt.go` 的 `GenerateToken` 函数
  - [ ] 修改 `middleware/auth.go` 添加版本验证
  - [ ] 更新所有修改 `user_type` 的地方
  - [ ] 执行数据库迁移
- [ ] **添加单元测试** 验证修复

---

## ✅ 验证测试用例

修复后，建议添加以下测试：

```go
// 1. 日期验证测试
func TestGetClassWeakPoints_InvalidDate(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/v1/ai/weak_points/class?class_id=1&start_date=2024-13-45", nil)
    // 应返回 400
}

// 2. 长度限制测试
func TestRegister_UsernameTooLong(t *testing.T) {
    longName := strings.Repeat("a", 101)
    // 应返回 400
}

// 3. 分页参数测试
func TestClassRecords_InvalidPage(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/v1/classes/1/records/debug?page=foo", nil)
    // 应返回 400
}

// 4. Token 版本测试
func TestTokenVersion_Invalidation(t *testing.T) {
    // 1. 登录获取 token
    // 2. 直接更新数据库中的 token_version
    // 3. 使用旧 token 访问接口
    // 应返回 401 "Token已失效，请重新登录"
}
```

---

## 📚 参考文档

- [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)
- [Gin Validator 文档](https://pkg.go.dev/gopkg.in/validator.v2)
- [JWT Best Practices](https://auth0.com/docs/security/tokens/json-web-tokens)

---

**文档版本**: v1.0  
**最后更新**: 2025-02-20  
**维护者**: Roo (软件架构师)