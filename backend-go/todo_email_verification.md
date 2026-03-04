# 邮箱验证功能实现计划

## 任务清单

- [ ] 1. 扩展 User 模型，添加邮箱验证相关字段
- [ ] 2. 更新配置，添加 SMTP 邮件服务器配置
- [ ] 3. 创建邮件服务 (service/email_service.go)
- [ ] 4. 修改注册流程：生成验证token（需要人机检测），发送验证邮件
- [ ] 5. 添加邮箱验证接口 (GET /auth/verify-email)
- [ ] 6. 添加重新发送验证邮件接口 (POST /auth/resend-verification)
- [ ] 7. 修改登录流程：检查邮箱验证状态
- [ ] 8. 添加中间件：检查邮箱是否已验证
- [ ] 9. 更新 .env.example 和 README 文档
- [ ] 10. 编写测试

## 详细设计

### 1. User 模型扩展

在 `models/user.go` 中添加字段：
```go
Email                  string     // 邮箱地址（可选）
EmailVerified         bool       // 邮箱是否已验证
EmailVerificationToken string    // 邮箱验证token
EmailVerificationSentAt *time.Time // 验证邮件发送时间
EmailVerificationExpiresAt *time.Time // 验证链接过期时间
```

### 2. 配置更新

在 `config/config.go` 的 Config 结构体中添加：
```go
SMTPHost     string
SMTPPort     int
SMTPUsername string
SMTPPassword string
SMTPFrom     string
```

### 3. 邮件服务

创建 `service/email_service.go`：
- `SendVerificationEmail(user *models.User) error`
- `GenerateVerificationToken() string`
- `VerifyToken(token string) (uint, error)`

### 4. 注册流程修改

在 `controller/auth.go` 的 Register 函数中：
- 如果提供了邮箱字段，生成验证token
- 调用邮件服务发送验证邮件
- 设置 `email_verification_sent_at` 和 `email_verification_expires_at`

### 5. 邮箱验证接口

新增 `GET /auth/verify-email`：
- 接收 token 参数
- 验证 token 有效性
- 更新用户 `email_verified` 字段
- 返回验证结果页面或 JSON

### 6. 重新发送验证邮件

新增 `POST /auth/resend-verification`：
- 检查用户是否已登录
- 检查是否已验证
- **频率限制**：每小时最多5次（Redis 计数）
- 重新生成 token 并发送邮件

### 7. 登录流程修改

在 `controller/auth.go` 的 Login 函数中：
- 查询用户后检查 `email_verified`
- 如果未验证且需要邮箱验证，返回提示

### 8. 中间件（可选）

创建 `middleware/email_verified.go`：
```go
func RequireEmailVerified() gin.HandlerFunc
```

### 9. 文档更新

- 更新 `backend-go/.env.example` 添加 SMTP 配置项
- 更新 `backend-go/README.md` 说明邮箱验证功能

### 10. 测试

创建 `backend-go/tests/test_email_verification.go`：
- 测试邮件发送
- 测试验证token生成和验证
- 测试验证接口
- 测试重新发送接口

## 注意事项

1. 邮箱验证是可选的，用户可以不提供邮箱
2. 验证token有效期建议 24 小时
3. 需要处理 Redis 不可用的情况（邮件服务降级）
4. **频率限制实现**：
   - 使用 Redis 键：`email_verification_rate:{student_id}:{hour}`
   - 每小时最多5次：`INCR` + `EXPIRE 1h`
   - 示例代码：
   ```go
   key := fmt.Sprintf("email_verification_rate:%s:%s", studentID, time.Now().Format("2006010215"))
   count, _ := config.RedisClient.Incr(ctx, key).Result()
   if count == 1 {
       config.RedisClient.Expire(ctx, key, 1*time.Hour)
   }
   if count > 5 {
       return errors.New("发送频率过高")
   }
   ```
5. 验证链接需要是安全的随机token