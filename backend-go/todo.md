

## 2. Debug 对话关闭机制

### 目标
为多轮调试对话添加显式的关闭状态，防止对话结束后被继续使用或滥用。

### 实现方案
- **数据库模型**：在 `models/ai_record.go` 的 `AIRecord` 表中添加 `IsClosed` 字段（布尔值，默认 false）
  - 注意：这个逻辑只对debug功能生效，其他功能不受影响
- **关闭接口**：
  - 新增 `POST /api/v1/ai/debug/close` 接口
  - 请求体：`{"conversation_id": "string"}`
  - 逻辑：验证用户权限，将指定 conversation_id 的 `IsClosed` 设为 true
- **防护检查**：
  - 在 `POST /api/v1/ai/debug_v2` 接口中，检查传入的 `conversation_id` 是否已关闭
  - 已关闭的对话返回 HTTP 400，消息："Conversation already closed"
- **自动关闭**（可选）：
  - 当对话达到最大轮次（4轮），或前端发来的响应结束标记时，自动关闭对话


### 涉及文件
- `models/ai_record.go` - 添加 `IsClosed` 字段和迁移
- `controller/ai_proxy_controller.go` - 添加关闭接口和检查逻辑
- `service/ai_proxy_service.go` - 实现关闭业务逻辑
- `README.md` - 更新 API 文档


- 为 `ai_records` 表添加 `is_closed` 列（默认 false）
- 确保现有数据兼容（设置默认值）

