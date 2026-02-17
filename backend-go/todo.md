

## 安全修复（严重漏洞）

### 2.1 问题描述
**存在严重权限漏洞**：多个 AI 接口直接使用请求体中的 `student_id`，未与认证用户进行比对。攻击者可以构造请求访问任意学生的数据。

### 2.2 受影响接口

#### 🔴 POST /api/v1/ai/evaluate
- 文件：`controller/ai_controller.go`
- 问题：多处使用 `req.StudentID` 而非认证用户 ID
  - 第69行：保存请求记录
  - 第79行：创建 job
  - 第105行：保存错误记录
  - 第121行：保存响应记录
  - 第137行：fallback 服务调用

#### 🔴 POST /api/v1/ai/recommend
- 文件：`controller/ai_controller.go`
- 问题：多处使用 `req.StudentID` 而非认证用户 ID
  - 第177行：保存请求记录
  - 第187行：创建 job

#### 🔴 POST /api/v1/ai/debug_v2
- 文件：`controller/ai_proxy_controller.go`
- 问题：多处使用 `req.StudentID` 而非认证用户 ID
  - 第68行：创建 conversation 记录
  - 第97行：保存请求记录
  - 第107行：创建 job
  - 第133行：保存错误记录
  - 第149行：保存响应记录
  - 第159行：保存 weak_points（间接使用）

#### 🔴 POST /api/v1/ai/start
- 文件：`controller/ai_proxy_controller.go:244`
- 问题：请求模型要求 `student_id` 字段，并使用它生成 conversationID（第258行）

### 2.3 修复步骤

#### 步骤 1：统一修改模式

在所有受影响的 Controller 方法开头添加：
```go
studentID := c.MustGet("student_id").(string)
```

然后**将所有**使用 `req.StudentID` 的地方替换为 `studentID`，包括：
- 创建数据库记录（AIRecord、Conversation）
- 创建 job
- 调用服务层方法
- 任何其他数据持久化操作

#### 步骤 2：详细修复清单

##### 2.2.1 controller/ai_controller.go - HandleEvaluate

```go
func (ctrl *AIController) HandleEvaluate(c *gin.Context) {
    // ✅ 在函数开头添加
    studentID := c.MustGet("student_id").(string)
    
    requestBody, err := ioutil.ReadAll(c.Request.Body)
    // ...
    
    var req models.EvaluateRequest
    json.Unmarshal(requestBody, &req)
    
    // ✅ 替换所有 req.StudentID 为 studentID：
    // 第69行：requestRecord.StudentID
    // 第79行：job 的 studentID 参数
    // 第105行：errorRecord.StudentID
    // 第121行：responseRecord.StudentID
    // 第137行：ProxyEvaluate 的 studentID 参数
}
```

##### 2.2.2 controller/ai_controller.go - HandleRecommend

```go
func (ctrl *AIController) HandleRecommend(c *gin.Context) {
    // ✅ 在函数开头添加
    studentID := c.MustGet("student_id").(string)
    
    // ...
    
    // ✅ 替换所有 req.StudentID 为 studentID：
    // 第177行：requestRecord.StudentID
    // 第187行：job 的 studentID 参数
}
```

##### 2.2.3 controller/ai_proxy_controller.go - HandleDebugV2

```go
func (ctrl *AIProxyController) HandleDebugV2(c *gin.Context) {
    // ✅ 在函数开头添加
    studentID := c.MustGet("student_id").(string)
    
    requestBody, err := ioutil.ReadAll(c.Request.Body)
    // ...
    
    var req models.DebugV2Request
    json.Unmarshal(requestBody, &req)
    
    // ✅ 替换所有 req.StudentID 为 studentID：
    // 第68行：conversation.StudentID
    // 第97行：requestRecord.StudentID
    // 第107行：job 的 studentID 参数
    // 第133行：errorRecord.StudentID
    // 第149行：responseRecord.StudentID
    // 第159行：saveWeakPoints 调用（间接使用）
}
```

##### 2.2.4 controller/ai_proxy_controller.go - StartConversation

```go
func (ctrl *AIProxyController) StartConversation(c *gin.Context) {
    // ✅ 从 token 获取
    studentID := c.MustGet("student_id").(string)
    
    // ✅ 修改请求模型，移除 StudentID 字段
    var req struct {
        ProblemDescription string             `json:"problem_description" binding:"required"`
        Code               string             `json:"code" binding:"required"`
        TestPoints         []models.TestPoint `json:"test_points"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }
    
    // ✅ 使用 token 的 studentID 生成 conversationID
    conversationID := fmt.Sprintf("conv_%d_%s", time.Now().Unix(), studentID)
    
    // 后续代码保持不变...
}
```

#### 步骤 3：验证 Service 层安全性

检查以下方法，确保它们正确使用传入的 `studentID` 参数进行数据隔离：

- `service/ai_proxy_service.go:230` `GetAIRecordsByStudentID`
  ✅ 已正确使用 `studentID` 参数

- `service/ai_proxy_service.go:240` `CloseConversation`
  ✅ 已正确使用双重验证（conversation_id AND student_id）

- `service/ai_proxy_service.go:277` `IsConversationClosed`
  ⚠️ **需要修复**：当前只根据 `conversation_id` 查询，应改为同时验证 `student_id`：
  ```go
  func (s *AIProxyService) IsConversationClosed(conversationID, studentID string) (bool, error) {
      var conv models.Conversation
      err := s.DB.Where("conversation_id = ? AND student_id = ?", conversationID, studentID).First(&conv).Error
      // ...
  }
  ```

#### 步骤 4：模型字段调整（可选，不过无所谓）

从以下请求模型中移除 `StudentID` 字段，因为不再需要：
- `models.EvaluateRequest`
- `models.RecommendRequest`
- `models.DebugV2Request`

这可以防止未来开发人员误用。

#### 步骤 5：前端更新

更新前端 API 调用，移除所有请求体中的 `student_id` 字段：
- `frontend-vue/src/api/index.js`
- `frontend-vue/src/views/AIDebug.vue`
- `frontend-vue/src/views/Evaluate.vue`
- `frontend-vue/src/views/Recommend.vue`

#### 步骤 6：测试验证

- [ ] 编写单元测试：用学生A的token，请求中指定学生B的ID，验证实际使用的是A的ID
- [ ] 手动测试所有接口的数据隔离
- [ ] 验证 conversation 相关操作只能访问自己的会话
- [ ] 确保历史记录查询只返回当前用户的数据

---

## 验证步骤
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
- ⚠️ **必须修复** `IsConversationClosed` 方法，添加 `studentID` 参数验证
- 建议在 Service 层添加防御性断言，确保传入的 studentID 与 conversation 归属一致