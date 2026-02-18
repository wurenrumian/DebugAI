# 薄弱点功能开发任务

## 背景
基于现有薄弱点模型（UserWeakPoint、WeakPoint）和已实现的个人查询接口，需要扩展班级查询功能并创建可复用的薄弱点展示组件。

## 现状分析
- ✅ 后端已实现：个人薄弱点查询（支持时间范围）、Top N查询
- ✅ 前端已实现：Recommend页面薄弱点选择功能（需重构为组件）
- ✅ 数据模型：UserWeakPoint（学生-薄弱点关联，按日期隔离）、WeakPoint（字典）
- ❌ 缺失：班级级薄弱点查询接口
- ❌ 缺失：可复用的薄弱点展示组件

## 后端任务

### 1. 班级薄弱点查询接口
**路径**: `GET /api/v1/ai/weak_points/class`
**权限**: 仅班级管理员（teacher/TA）或系统admin可访问
**查询参数**:
  - `class_id` (必填): 班级ID
  - `start_date` (可选): 开始日期，格式 YYYY-MM-DD，不填默认为当天
  - `end_date` (可选): 结束日期，格式 YYYY-MM-DD，不填默认为当天
  - `student_ids` (可选): 学生ID列表（JSON 数组），不传则返回班级所有学生

**响应格式**:
```json
{
  "message": "班级薄弱点查询成功",
  "data": [
    {
      "student_id": "S001",
      "username": "张三",
      "weak_points": [
        {
          "keyword": "数组",
          "category": "数据结构",
          "count": 5,
          "description": "数组操作相关知识点"
        }
      ],
      "total_count": 15
    }
  ]
}
```

**实现要点**:
- 在 `backend-go/controller/ai_controller.go` 添加 `GetClassWeakPoints`
- 在 `backend-go/service/ai_service.go` 添加 `GetClassWeakPoints`
- 默认日期：未提供日期参数时查询当天数据
- 权限验证：`service.IsClassAdmin(currentUserID, classID)`
- 学生成员验证：提供的 `student_ids` 必须属于该班级
- 查询逻辑：联表 ClassMember → User → UserWeakPoint → WeakPoint
- 按学生分组，每个学生的薄弱点按 count 降序

### 2. 接口完善
- `GetUserWeakPoints` 补充返回 `category` 和 `description`
- 统一默认日期行为：所有薄弱点查询接口不传日期参数时返回当天数据

## 前端任务

### 1. 创建 WeakPointDisplay 组件
**路径**: `frontend-vue/src/components/WeakPointDisplay.vue`

**Props**:
  - `weakPoints`: `[{keyword, category, count, description?}]`
  - `selectable`: Boolean (default false)
  - `selected`: String[] (selectable 时有效)
  - `showDescription`: Boolean (default false)
  - `maxDisplay`: Number (default 0, 全部显示)

**Events**:
  - `update:selected`: 选中状态变化，返回 String[]

**核心功能**:
- 按 category 分组展示
- 显示 `keyword (count次)`
- 可选中模式：多选/取消，高亮选中状态
- 可选显示描述（tooltip）
- `maxDisplay` 限制显示数量，超出显示"查看更多"
- 支持 `v-model:selected` 双向绑定

**注意**：
- 组件仅负责展示和选择交互，不包含时间范围或 topK 筛选逻辑
- 数据获取由父组件负责，通过 props 传递已筛选好的数据

### 2. 集成到 Recommend 页面
**路径**: `frontend-vue/src/views/Recommend.vue`

**改造步骤**:
1. 引入 WeakPointDisplay 组件
2. 在左侧面板顶部添加筛选控件：
   - 时间范围选择：开始日期 `input type="date"`、结束日期 `input type="date"`
   - Top K 输入：数字输入框 `input type="number"`（默认值 5）
3. 将筛选条件绑定到响应式变量：`startDate`、`endDate`、`topK`
4. 修改 `fetchUserWeakPoints` 方法，根据筛选条件调用不同接口：
   - 如果 `topK > 0`：调用 `aiAPI.getTopWeakPoints({ start_date: startDate, end_date: endDate })`
   - 否则：调用 `aiAPI.getWeakPoints({ start_date: startDate, end_date: endDate })`
5. 替换第 10-35 行的薄弱点列表为 WeakPointDisplay 组件
6. 绑定: `:weakPoints="userWeakPoints" :selectable="true" v-model:selected="selectedWeakPoints"`
7. 删除原 `v-for` 循环和 `toggleWeakPoint` 方法
8. 保留推荐数量设置和 `submitRecommend` 逻辑
9. 删除第 193 行自动全选逻辑
10. 监听筛选条件变化，自动重新获取数据（使用 `watch`）

**数据流**:
- Recommend 通过 `v-model:selected` 绑定 `selectedWeakPoints`
- WeakPointDisplay emit `update:selected` 更新
- `submitRecommend` 直接使用 `selectedWeakPoints` 构建 `weak_points` 字典
- 筛选条件变化 → 重新调用 `fetchUserWeakPoints` → 更新 `userWeakPoints`

**新增 API 调用**:
- `aiAPI.getWeakPoints({ start_date, end_date })` - 获取指定时间范围的薄弱点
- `aiAPI.getTopWeakPoints({ start_date, end_date })` - 获取指定时间范围的 Top K 薄弱点

## 测试验证

### 后端测试
- 个人查询：无日期、日期范围、无效格式
- 班级查询：无权限、权限正常、班级不存在、学生不在班级
- 数据聚合：多学生、日期过滤、排序

### 前端测试
- 组件：选中/取消、双向绑定、数量限制
- Recommend：组件集成后正确获取选中薄弱点
- 权限：非班级管理员拒绝访问班级接口

## 注意事项
- 班级接口严格权限控制
- 数据量大时考虑分页或限制
- 组件样式与现有设计系统一致
- 保持向后兼容
