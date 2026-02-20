# Frontend Vue - AI 教学辅助平台前端

基于 Vue 3.5+ 和 Vite 5.4+ 构建的现代化单页应用（SPA）。

**技术栈**：Vue 3.5+ | Vite 5.4+ | Vue Router 4.6+ | Pinia 2.3+ | Axios 1.13+

**开发端口**：`http://localhost:5173`

## 快速开始

### 前置条件

- Node.js 18.0+（推荐 20.x LTS）
- 包管理器：npm 9+ 或 yarn 1.22+ 或 pnpm 8+

### 安装依赖

```bash
cd frontend-vue
npm ci  # 或 npm install
```

### 开发模式

```bash
npm run dev
```

访问 `http://localhost:5173` 查看应用。

### 生产构建

```bash
npm run build
```

构建产物位于 `dist/` 目录。

### 环境变量配置

创建 `.env` 文件：

```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_APP_TITLE=AI 教学辅助平台
```

## 核心功能模块

### 1. 用户认证

- 学号 + 密码登录
- JWT Token 存储于 `localStorage`
- Axios 拦截器自动添加 `Authorization` 头
- 401 响应自动登出并跳转登录页
- 路由守卫保护需要认证的页面

### 2. AI 代码调试（4轮对话）

**路由**：`/ai-debug`

**流程**：

| 轮次 | 名称     | AI 输出                                            | 用户操作              |
| ---- | -------- | -------------------------------------------------- | --------------------- |
| 1    | 理解思路 | `student_thought`, `suggested_correction`          | 阅读，点击"继续"      |
| 2    | 指出问题 | `problem_summary`, `key_issues[]`, `weak_points[]` | 选择/输入，点击"继续" |
| 3    | 调试指导 | `debug_guidance`, `ask_for_detail`                 | 选择/输入，点击"继续" |
| 4    | 修改建议 | `suggestions[]`                                    | 阅读，自动关闭        |

**API**：
- `POST /api/v1/ai/debug_v2` - 核心交互
- `POST /api/v1/ai/debug/close` - 手动关闭
- `GET /api/v1/ai/round_info` - 获取轮次信息

### 3. AI 代码评价

**路由**：`/evaluate`

**输入**：题目描述、代码、测试点（可选）

**输出**：四个维度评分 + 整体评价
- 功能正确性
- 逻辑严谨性
- 算法质量
- 结构规范性

**API**：`POST /api/v1/ai/evaluate`

### 4. AI 题目推荐

**路由**：`/recommend`

**功能**：
- 从用户薄弱点中多选
- 日期筛选统计薄弱点
- 调整推荐数量（3/5/8/10）
- 显示推荐题目标签、相关度、理由

**API**：
- `GET /api/v1/ai/weak_points` - 获取薄弱点
- `POST /api/v1/ai/recommend` - 推荐题目

### 5. 历史记录

**路由**：`/history`

**功能**：
- 标签页切换：调试/评价/推荐历史
- 按会话分组展示（调试）或按记录分组（评价/推荐）
- 查看详情弹窗，展示完整对话/评价/推荐内容

**可复用组件**：
- `DebugHistoryTab.vue` / `EvaluateHistoryTab.vue` / `RecommendHistoryTab.vue`
- `HistoryDetailModal.vue`（通用详情弹窗）

### 6. 班级管理

**路由**：`/class-manage`

**模块**：
- **班级选择器**：创建/加入/切换班级
- **班级信息**：展示班级详情和当前用户角色
- **成员管理**：添加/移除成员（批量），助教仅可管理学生
- **学生历史查询**：按学生、类型、时间筛选查询历史
- **班级薄弱点**：统计整体薄弱点，支持导出 JSON

## 路由权限

所有需要认证的路由均设置 `meta.requiresAuth = true`，路由守卫检查用户认证状态，未认证重定向到 `/login`。

**认证路由**：
- `/profile` - 个人主页
- `/class-manage` - 班级管理
- `/ai-debug` - AI 调试
- `/evaluate` - 代码评价
- `/recommend` - 题目推荐
- `/history` - 历史记录

## API 集成

### API 服务封装（`src/api/index.js`）

统一管理所有 API 调用，包含：
- Axios 实例配置（`VITE_API_BASE_URL`）
- 请求拦截器：自动添加 JWT Token
- 响应拦截器：处理 401 错误
- 按模块分组：`authApi`、`aiApi`、`classApi`

### 状态管理（Pinia）

**Auth Store**（`src/stores/auth.js`）：
- `token`、`user` 状态（持久化到 `localStorage`）
- `isAuthenticated`、`userType`、`username` 计算属性
- `login()`、`logout()`、`clear()` 动作

**Class Store**（`src/stores/class.js`）：
- `currentClassId`、`myClasses`、`currentClassMembers` 状态
- `currentClass` 计算属性
- `fetchMyClasses()`、`setCurrentClass()`、`fetchClassMembers()` 动作

## 开发规范

### 代码风格

- 使用 `<script setup>` 语法
- 组件名使用大驼峰（`PascalCase`）
- Props 使用 `camelCase`，模板中使用 `kebab-case`
- 遵循 ESLint 规则（`.eslintrc.cjs`）

### 组件设计原则

- 单一职责：每个组件负责一个明确的 UI 功能
- Props 最小化：仅传递必要数据
- 事件驱动：子组件通过 `emit` 通知父组件
- 可复用性：通用组件独立于业务逻辑

### API 调用规范

- 统一通过 `src/api/index.js`
- 使用 `try/catch` 处理错误
- 所有异步操作应有 `isLoading` 状态
- 显示用户友好的错误提示

## 相关项目

- **[Go 后端服务](../backend-go/README.md)** - 中介服务、Worker Pool、限流
- **[Python AI 服务](../ai-python/README.md)** - 核心 AI 能力
- **[项目总览](../README.md)** - 整体架构和快速启动

## 许可证

MIT License
