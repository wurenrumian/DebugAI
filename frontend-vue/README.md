# Frontend Vue - AI 教学辅助平台前端

## 概述

基于 Vue 3 + Vite 构建的前端应用，为 AI 教学辅助平台提供用户界面。与 Go 后端 ([`backend-go`](backend-go/README.md)) 和 Python AI 服务 ([`ai-python`](ai-python/README.md)) 配合，提供学生登录注册、AI 代码调试、代码评价、题目推荐、历史记录、班级管理等功能。

## 技术栈

- **框架**: Vue 3.5+ (Composition API + `<script setup>`)
- **构建工具**: Vite 5.4+ (实际 5.4.21)
- **路由**: Vue Router 4.6+ (实际 4.6.4)
- **状态管理**: Pinia 2.3+ (实际 2.3.1)
- **HTTP 客户端**: Axios 1.6+ (实际 1.13.5)
- **样式**: 原生 CSS

## 项目结构

```
frontend-vue/
├── index.html              # 入口 HTML
├── package.json            # 项目依赖配置
├── vite.config.js          # Vite 配置 (代理后端 API)
└── src/
    ├── main.js             # 应用入口
    ├── App.vue             # 根组件
    ├── style.css           # 全局样式
    ├── api/
    │   └── index.js        # API 服务封装 (Axios + 拦截器)
    ├── router/
    │   └── index.js        # 路由配置 (含权限守卫)
    ├── stores/
    │   ├── auth.js         # 用户认证状态管理 (Pinia)
    │   └── class.js        # 班级状态管理 (Pinia + 历史导出)
    ├── components/
    │   ├── AIResponseDisplay.vue     # AI 响应解析显示组件
    │   ├── WeakPointDisplay.vue      # 薄弱点展示组件 (支持选择/分组)
    │   ├── HistoryTabs/              # 历史记录标签页组件（可复用）
    │   │   ├── DebugHistoryTab.vue   # 调试历史列表
    │   │   ├── EvaluateHistoryTab.vue # 评价历史列表
    │   │   ├── RecommendHistoryTab.vue # 推荐历史列表
    │   │   └── HistoryDetailModal.vue # 详情弹窗 (支持三种类型)
    │   └── class/                    # 班级管理组件
    │       ├── ClassSelector.vue     # 班级选择器
    │       ├── ClassInfoTab.vue      # 班级信息标签页
    │       ├── ClassManageTab.vue    # 成员管理标签页 (含添加/移除)
    │       ├── ClassHistoryQueryTab.vue # 学生历史查询标签页
    │       └── ClassWeakPointsQueryTab.vue # 班级薄弱点查询标签页
    └── views/
        ├── Login.vue        # 登录页面 (学号+密码)
        ├── Register.vue     # 注册页面
        ├── Profile.vue      # 个人主页
        ├── ClassManage.vue  # 班级管理页面 (多标签页)
        ├── AIDebug.vue      # AI 对话调试页面 (4轮流程)
        ├── Evaluate.vue     # AI 代码评价页面
        ├── Recommend.vue    # AI 题目推荐页面
        └── History.vue      # 历史记录页面 (标签页切换)
```

## 快速开始

### 前置条件

- Node.js 18.0+ (Vite 5.x 要求)
- npm 或 yarn

### 安装依赖

```bash
cd frontend-vue
npm install
```

### 开发模式

```bash
npm run dev
```

前端服务将在 `http://localhost:5173` 启动，Vite 代理配置将 API 请求转发到 Go 后端 (`http://localhost:8080`)。

### 生产构建

```bash
npm run build
```

构建产物将生成在 `dist/` 目录，可直接部署到静态文件服务器。

### 预览构建结果

```bash
npm run preview
```

启动本地静态文件服务器预览生产构建结果。

## API 代理配置

Vite 开发服务器代理配置（[`vite.config.js`](vite.config.js)）：

| 路径      | 目标                           |
| --------- | ------------------------------ |
| `/auth/*` | `http://localhost:8080/auth/*` |
| `/api/*`  | `http://localhost:8080/api/*`  |

所有以 `/auth` 和 `/api` 开头的请求将被 Vite 开发服务器代理到 Go 后端 (`http://localhost:8080`)。生产环境需配置 Nginx 或其他反向代理实现相同效果。

## 核心功能

### 1. 用户认证

- 学号+密码登录/注册
- JWT Token 自动管理（存储在 `localStorage`）
- Axios 拦截器自动添加认证头
- 401 响应自动清除 Token 并跳转登录
- 路由权限守卫保护受保护页面

### 2. AI 代码调试 ([`/ai-debug`](src/views/AIDebug.vue))

**4轮对话流程**：

| 轮次 | 名称         | 交互方式                                                   |
| ---- | ------------ | ---------------------------------------------------------- |
| 1    | 理解学生思路 | 学生输入问题描述、代码、测试点 → AI 分析代码并给出初步反馈 |
| 2    | 指出问题点   | AI 指出问题 → 学生通过按钮选择或文本输入确认/修正          |
| 3    | 调试指导     | AI 提供调试指导 → 学生通过按钮选择或文本输入确认/继续      |
| 4    | 详细修改指导 | AI 提供详细修改建议，结束对话                              |

**API 端点**：
- `POST /api/v1/ai/debug_v2` - 发送调试请求（核心交互）
- `GET /api/v1/ai/round_info` - 获取当前轮次信息和提示
- `POST /api/v1/ai/start` - 开始新对话
- `POST /api/v1/ai/debug/close` - 关闭当前对话

**特性**：
- 实时轮次状态显示（第 X/4 轮）
- 轮次标题和描述提示
- 对话历史按轮次分组展示
- 支持新建对话（重置状态）
- 第1轮输入后禁用，后续轮次可继续交互
- 第2、3轮支持按钮快速选择和文本输入
- 第4轮显示详细修改建议后自动完成

### 3. AI 代码评价 ([`/evaluate`](src/views/Evaluate.vue))

**API 端点**：
- `POST /api/v1/ai/evaluate` - 提交代码评价请求

**评价维度**：
- **功能正确性**：代码是否满足题目要求
- **逻辑严谨性**：边界条件、异常处理
- **算法质量**：时间/空间复杂度、算法选择
- **结构规范性**：代码结构、命名规范、可读性

**输入**：
- 题目描述
- 代码（C/C++）
- 测试点（可选，每行一个，格式：输入/输出/状态）

**输出**：综合评价报告，包含各维度评分和具体建议

### 4. AI 题目推荐 ([`/recommend`](src/views/Recommend.vue))

**API 端点**：
- `POST /api/v1/ai/recommend` - 基于薄弱点获取推荐题目
- `GET /api/v1/ai/weak_points` - 获取用户薄弱点（支持日期筛选）
- `GET /api/v1/ai/weak_points/top` - 获取Top N薄弱点（支持日期筛选）

**特性**：
- **薄弱点选择**：从用户历史薄弱点中多选（使用 [`WeakPointDisplay.vue`](src/components/WeakPointDisplay.vue) 组件）
- **日期筛选**：可选择开始/结束日期范围，统计指定时间段内的薄弱点
- **Top K 控制**：可设置显示前 N 个薄弱点（0 = 显示全部）
- **推荐数量**：可调整推荐题目数量（3/5/8/10）
- **推荐结果**：显示题目列表、推荐理由、相关度评分

### 5. 历史记录 ([`/history`](src/views/History.vue))

**API 端点**：
- `GET /api/v1/ai/records` - 获取所有类型历史记录
- `GET /api/v1/ai/records/debug` - 获取调试历史
- `GET /api/v1/ai/records/evaluate` - 获取评价历史
- `GET /api/v1/ai/records/recommend` - 获取推荐历史

**特性**：
- 标签页切换（调试/评价/推荐）
- 按会话（conversation_id）分组展示
- 显示每会话的轮次、记录数、最新时间
- 查看详情弹窗，展示完整对话/评价/推荐内容
- 详情中包含题目描述、提交代码、AI 响应等完整信息

**可复用组件**：
历史记录相关组件已封装为独立可复用组件，适用于班级管理等场景：

| 组件                                                                            | 说明             | 事件输出                                                                 |
| ------------------------------------------------------------------------------- | ---------------- | ------------------------------------------------------------------------ |
| [`DebugHistoryTab.vue`](src/components/HistoryTabs/DebugHistoryTab.vue)         | 调试历史列表     | `@view-details` 返回 `{ records, initialSubmission, type: 'debug' }`     |
| [`EvaluateHistoryTab.vue`](src/components/HistoryTabs/EvaluateHistoryTab.vue)   | 评价历史列表     | `@view-details` 返回 `{ records, initialSubmission, type: 'evaluate' }`  |
| [`RecommendHistoryTab.vue`](src/components/HistoryTabs/RecommendHistoryTab.vue) | 推荐历史列表     | `@view-details` 返回 `{ records, initialSubmission, type: 'recommend' }` |
| [`HistoryDetailModal.vue`](src/components/HistoryTabs/HistoryDetailModal.vue)   | 详情弹窗（通用） | 接收 `records`, `initialSubmission`, `type` props，发出 `@close` 事件    |

**复用示例**：
```vue
<template>
  <DebugHistoryTab
    :records="classDebugRecords"
    @view-details="handleViewDetails"
  />
  
  <HistoryDetailModal
    v-if="showModal"
    :records="selectedRecords"
    :initial-submission="initialSubmission"
    :type="selectedType"
    @close="showModal = false"
  />
</template>

<script setup>
import DebugHistoryTab from '@/components/HistoryTabs/DebugHistoryTab.vue'
import HistoryDetailModal from '@/components/HistoryTabs/HistoryDetailModal.vue'

const showModal = ref(false)
const selectedRecords = ref([])
const initialSubmission = ref(null)
const selectedType = ref('debug')

const handleViewDetails = ({ records, initialSubmission: initSub, type }) => {
  selectedRecords.value = records
  initialSubmission.value = initSub
  selectedType.value = type
  showModal.value = true
}
</script>
```

### 6. 个人主页 ([`/profile`](src/views/Profile.vue))

- 用户基本信息展示（学号、用户名、角色）
- 快捷入口：AI 调试、代码评价、题目推荐、历史记录、班级管理
- 退出登录功能（清除 Token 和本地存储）

### 7. 班级管理 ([`/class-manage`](src/views/ClassManage.vue))

**组件结构**：
- [`ClassSelector.vue`](src/components/class/ClassSelector.vue) - 班级选择器（创建/加入/切换班级）
- [`ClassInfoTab.vue`](src/components/class/ClassInfoTab.vue) - 班级基本信息展示
- [`ClassManageTab.vue`](src/components/class/ClassManageTab.vue) - 成员管理（添加/移除，支持批量）
- [`ClassHistoryQueryTab.vue`](src/components/class/ClassHistoryQueryTab.vue) - 学生历史查询（按学生、时间筛选）
- [`ClassWeakPointsQueryTab.vue`](src/components/class/ClassWeakPointsQueryTab.vue) - 班级薄弱点查询（按学生、时间筛选，支持导出）

**功能说明**：
- **班级选择**：创建新班级、加入现有班级、切换当前班级
- **班级信息**：查看班级 ID、名称、创建者、创建时间
- **成员管理**：添加/移除班级成员（批量操作，支持多选学生）
- **学生历史**：查询本班级指定学生的 Debug/Evaluate/Recommend 历史（按学生、时间筛选）
- **班级薄弱点**：统计班级整体薄弱点（按学生、时间筛选），支持导出 JSON

**权限说明**（后端控制，前端根据用户角色显示/隐藏功能）：
| 角色       | 班级选择 | 查看信息 | 成员管理   | 学生历史 | 班级薄弱点 |
| ---------- | -------- | -------- | ---------- | -------- | ---------- |
| 系统管理员 | ✅        | ✅        | ✅          | ✅        | ✅          |
| 班级创建者 | ✅        | ✅        | ✅          | ✅        | ✅          |
| 教师       | ✅        | ✅        | ✅          | ✅        | ✅          |
| 助教       | ✅        | ✅        | ⚠️ 有限权限 | ✅        | ✅          |
| 学生       | ✅        | ✅        | ❌          | ❌        | ❌          |

**助教权限限制**（前端实现）：
- 添加成员：角色选择器仅显示"学生"选项
- 移除成员：仅显示学生列表，隐藏教师/助教

**API 端点**：
- `GET /api/v1/classes` - 获取所有班级列表
- `GET /api/v1/classes/my` - 获取当前用户所属班级
- `POST /api/v1/classes` - 创建新班级（仅系统管理员）
- `POST /api/v1/classes/:id/join` - 加入指定班级
- `GET /api/v1/classes/:id` - 获取班级详情
- `GET /api/v1/classes/:id/members` - 获取班级成员列表
- `POST /api/v1/classes/:id/members/add` - 添加成员（支持批量）
- `POST /api/v1/classes/:id/members/remove` - 移除成员（支持批量）
- `GET /api/v1/classes/:id/records/debug` - 获取班级 Debug 历史（支持学生、时间筛选）
- `GET /api/v1/classes/:id/records/evaluate` - 获取班级 Evaluate 历史（支持学生、时间筛选）
- `GET /api/v1/classes/:id/records/recommend` - 获取班级 Recommend 历史（支持学生、时间筛选）
- `GET /api/v1/classes/:id/records/debug/export` - 导出班级 Debug 历史（JSON）
- `GET /api/v1/classes/:id/records/evaluate/export` - 导出班级 Evaluate 历史（JSON）
- `GET /api/v1/classes/:id/records/recommend/export` - 导出班级 Recommend 历史（JSON）
- `GET /api/v1/ai/weak_points/class` - 获取班级薄弱点统计（支持学生、时间筛选）

## 路由权限

| 路由            | 需要认证 | 说明         |
| --------------- | -------- | ------------ |
| `/login`        | 否       | 登录页面     |
| `/register`     | 否       | 注册页面     |
| `/profile`      | 是       | 个人主页     |
| `/class-manage` | 是       | 班级管理页面 |
| `/ai-debug`     | 是       | AI 调试页面  |
| `/evaluate`     | 是       | AI 评价页面  |
| `/recommend`    | 是       | AI 推荐页面  |
| `/history`      | 是       | 历史记录页面 |

未登录用户访问受保护路由时，自动重定向到登录页面。

## API 集成

### 认证流程

1. 用户通过学号+密码登录（`/auth/login`）
2. 后端返回 JWT Token 和用户信息
3. Token 存储在 `localStorage`，用户信息存储在 Pinia store 和 `localStorage`
4. Axios 请求拦截器自动为所有 API 请求添加 `Authorization: Bearer <token>` 头
5. 响应拦截器处理 401 错误：清除 Token 和用户信息，重定向到登录页面
6. 路由守卫检查 `meta.requiresAuth`，未认证自动重定向

### 主要 API 端点

**认证**：
- `POST /auth/register` - 用户注册（学号、用户名、密码）
- `POST /auth/login` - 用户登录（返回 Token 和用户信息）
- `POST /auth/logout` - 用户登出（后端清除 Token）

**用户**：
- `GET /api/v1/profile` - 获取当前用户信息

**AI 服务**：
- `POST /api/v1/ai/debug_v2` - AI 对话调试（核心接口，支持多轮交互）
- `GET /api/v1/ai/round_info` - 获取当前轮次信息（标题、描述、提示）
- `POST /api/v1/ai/start` - 开始新对话（重置状态）
- `POST /api/v1/ai/debug/close` - 关闭当前对话
- `POST /api/v1/ai/evaluate` - 代码评价（返回各维度评分和建议）
- `POST /api/v1/ai/recommend` - 基于薄弱点推荐题目
- `GET /api/v1/ai/weak_points` - 获取用户薄弱点统计（支持 `start_date`、`end_date` 查询参数）
- `GET /api/v1/ai/weak_points/top` - 获取 Top N 薄弱点（支持 `top_k`、`start_date`、`end_date` 参数）
- `GET /api/v1/ai/records` - 获取所有类型历史记录
- `GET /api/v1/ai/records/debug` - 获取调试历史记录
- `GET /api/v1/ai/records/evaluate` - 获取评价历史记录
- `GET /api/v1/ai/records/recommend` - 获取推荐历史记录

**班级管理**：
- `GET /api/v1/classes` - 获取所有班级（系统管理员）
- `GET /api/v1/classes/my` - 获取当前用户所属班级
- `POST /api/v1/classes` - 创建新班级（仅系统管理员）
- `POST /api/v1/classes/:id/join` - 加入指定班级
- `GET /api/v1/classes/:id` - 获取班级详情
- `GET /api/v1/classes/:id/members` - 获取班级成员列表
- `POST /api/v1/classes/:id/members/add` - 添加成员（`student_ids` 数组，`member_role` 参数）
- `POST /api/v1/classes/:id/members/remove` - 移除成员（`student_ids` 数组）
- `GET /api/v1/classes/:id/records/debug` - 获取班级 Debug 历史（支持 `student_ids`、`start_date`、`end_date` 筛选）
- `GET /api/v1/classes/:id/records/evaluate` - 获取班级 Evaluate 历史（支持筛选）
- `GET /api/v1/classes/:id/records/recommend` - 获取班级 Recommend 历史（支持筛选）
- `GET /api/v1/classes/:id/records/debug/export` - 导出班级 Debug 历史（JSON）
- `GET /api/v1/classes/:id/records/evaluate/export` - 导出班级 Evaluate 历史（JSON）
- `GET /api/v1/classes/:id/records/recommend/export` - 导出班级 Recommend 历史（JSON）
- `GET /api/v1/ai/weak_points/class` - 获取班级薄弱点统计（`class_id` 参数，支持 `student_ids`、`start_date`、`end_date` 筛选）

## 常见问题

### 1. 登录后跳转回登录页面
- 确认 Go 后端运行在 `http://localhost:8080` 且可访问
- 检查浏览器控制台是否有 CORS 错误（开发环境由 Vite 代理解决）
- 验证 Token 是否正确存储在 `localStorage` 中
- 确认后端 `/auth/login` 接口返回格式包含 `token` 字段

### 2. API 请求失败
- 确认后端服务运行状态（Go 后端 `http://localhost:8080`，Python AI 服务 `http://localhost:8000`）
- 检查网络请求路径和参数（使用浏览器开发者工具 Network 面板）
- 查看后端服务日志获取详细错误信息
- 确认 Vite 代理配置正确（开发环境）

### 3. 前端无法启动
- 确认 Node.js 版本 >= 18.0（Vite 5.x 要求）
- 删除 `node_modules` 和 `package-lock.json`，重新执行 `npm install`
- 检查 `package.json` 中的依赖版本兼容性

### 4. 班级管理权限问题
- 学生用户只能查看班级信息和加入班级，无法创建班级、管理成员、查看历史
- 助教可以管理成员但只能添加/移除学生（前端限制角色选择）
- 所有权限最终由后端验证，前端仅做 UI 控制

### 5. 历史记录详情显示异常
- 确认后端返回的数据结构包含 `initial_submission` 字段（题目描述和代码）
- 详情弹窗使用 [`HistoryDetailModal.vue`](src/components/HistoryTabs/HistoryDetailModal.vue) 组件，根据 `type` 参数渲染不同内容
- 调试历史按会话分组，评价和推荐历史按记录分组

## 开发建议

- 使用 Vue DevTools 调试组件状态和 Pinia store
- 浏览器开发者工具 Network 面板查看 API 请求/响应
- 后端服务：Go 后端 `http://localhost:8080`，Python AI 服务 `http://localhost:8000`
- 组件开发：优先使用 `<script setup>` 语法，保持代码简洁
- 状态管理：认证信息用 Pinia `auth` store，班级数据用 `class` store
- API 调用：统一通过 [`src/api/index.js`](src/api/index.js) 模块，避免硬编码 URL

## 相关项目

- [后端 Go 服务](../backend-go/README.md)
- [Python AI 服务](../ai-python/README.md)

## 许可证

MIT License
