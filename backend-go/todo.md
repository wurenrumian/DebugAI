

## 安全修复（严重漏洞）

### 2.1 问题描述
**存在严重权限漏洞**：多个 AI 接口直接使用请求体中的 `student_id`，未与认证用户进行比对。攻击者可以构造请求访问任意学生的数据。

### 2.2 受影响接口

#### 🔴 POST /api/v1/ai/evaluate
- 文件：`controller/ai_controller.go:38`
- 问题：使用 `req.StudentID` 而非认证用户 ID

#### 🔴 POST /api/v1/ai/recommend  
- 文件：`controller/ai_controller.go:148`
- 问题：使用 `req.StudentID` 而非认证用户 ID

#### 🔴 POST /api/v1/ai/debug_v2
- 文件：`controller/ai_proxy_controller.go:32`
- 问题：使用 `req.StudentID` 而非认证用户 ID

#### 🔴 POST /api/v1/ai/start
- 文件：`controller/ai_proxy_controller.go:244`
- 问题：使用请求中的 `StudentID` 生成 conversationID

### 2.3 修复步骤

#### 步骤 1：修改 `controller/ai_controller.go`

**HandleEvaluate 方法（第38行）：**
```go
// 修改前
func (ctrl *AIController) HandleEvaluate(c *gin.Context) {
    var req models.EvaluateRequest
    json.Unmarshal(requestBody, &req)
    job := service.NewAIJob(models.JobTypeEvaluate, req, req.StudentID, req.ConversationID)
    // ...
}

// 修改后
func (ctrl *AIController) HandleEvaluate(c *gin.Context) {
    studentID := c.MustGet("student_id").(string)  // ✅ 从 token 获取
    
    var req models.EvaluateRequest
    json.Unmarshal(requestBody, &req)
    
    // 使用认证用户的 studentID
    job := service.NewAIJob(models.JobTypeEvaluate, req, studentID, req.ConversationID)
    // ...
    aiResponse, err := ctrl.AIService.ProxyEvaluate(requestBody, studentID, req.ConversationID)
}
```

**HandleRecommend 方法（第148行）：**
```go
// 修改后
func (ctrl *AIController) HandleRecommend(c *gin.Context) {
    studentID := c.MustGet("student_id").(string)  // ✅ 从 token 获取
    
    var req models.RecommendRequest
    json.Unmarshal(requestBody, &req)
    
    // 使用认证用户的 studentID
    aiResponse, err := ctrl.AIService.ProxyRecommend(requestBody, studentID)
}
```

#### 步骤 2：修改 `controller/ai_proxy_controller.go`

**HandleDebugV2 方法（第32行）：**
```go
// 修改后
func (ctrl *AIProxyController) HandleDebugV2(c *gin.Context) {
    studentID := c.MustGet("student_id").(string)  // ✅ 从 token 获取
    
    var req models.DebugV2Request
    if err := json.Unmarshal(requestBody, &req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON request body"})
        return
    }
    
    // 使用认证用户的 studentID（覆盖请求体中的值）
    req.StudentID = studentID
    
    // 后续代码保持不变...
}
```

**StartConversation 方法（第244行）：**
```go
// 修改后
func (ctrl *AIProxyController) StartConversation(c *gin.Context) {
    var req struct {
        ProblemDescription string             `json:"problem_description" binding:"required"`
        Code               string             `json:"code" binding:"required"`
        TestPoints         []models.TestPoint `json:"test_points"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }
    
    // ✅ 从 token 获取，不使用请求体中的 StudentID
    studentID := c.MustGet("student_id").(string)
    
    // 使用认证用户的 studentID 生成 conversationID
    conversationID := fmt.Sprintf("conv_%d_%s", time.Now().Unix(), studentID)
    
    // 后续代码保持不变...
}
```

#### 步骤 3：添加断言验证（可选增强）

在服务层添加防御性检查：

```go
// 在 service/ai_service.go 的 ProxyEvaluate 开头添加
func (s *AIService) ProxyEvaluate(requestBody []byte, studentID, conversationID string) (map[string]interface{}, error) {
    // 从请求体中解析 student_id（如果存在）
    var reqData map[string]interface{}
    json.Unmarshal(requestBody, &reqData)
    if reqStudentID, ok := reqData["student_id"]; ok && reqStudentID != studentID {
        return nil, fmt.Errorf("student_id mismatch: token says %s but request says %s", studentID, reqStudentID)
    }
    
    // 原有逻辑...
}
```

#### 步骤 4：测试验证

- [ ] 编写单元测试验证学生无法通过构造请求访问他人数据
- [ ] 手动测试：用学生A的token，请求中指定学生B的ID，应返回错误或使用A的ID
- [ ] 确保所有历史记录查询都返回当前认证用户的数据

---

## 验证步骤

### 4.1 编译检查
```bash
go build -o backend-go.exe
```
确保编译无错误

### 4.2 测试运行
```bash
go test ./...
```
确保所有测试通过

### 4.3 安全测试
```bash
# 1. 用学生A登录，获取token
# 2. 构造请求，指定其他学生的 student_id
curl -X POST http://localhost:8080/api/v1/ai/evaluate \
  -H "Authorization: Bearer <A的token>" \
  -d '{"student_id":"B的学号","conversation_id":"test","code":"print(1)","problem_description":"test","test_points":[]}'

# 预期：记录应使用A的学号，而非请求体中的B的学号
```

### 4.4 功能验证
- 启动服务：`go run main.go`
- 测试所有 API 接口是否正常工作
- 验证数据隔离：每个用户只能看到自己的数据

---

## 五、注意事项

- 删除代码前建议提交 Git，便于回滚
- 安全修复应优先于代码清理
- 修复后需更新前端代码，移除请求体中的 `student_id` 字段（如果前端发送了）
- 考虑在未来的 API 设计中，将 `student_id` 完全从请求体移除，仅使用 token 认证