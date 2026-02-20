# Frontend Vue - AI 教学辅助平台前端

## 概述

基于 **Vue 3.5+** 和 **Vite 5.4+** 构建的现代化单页应用（SPA），为 AI 教学辅助平台提供完整的用户界面。采用 Composition API + `<script setup>` 语法，配合 Pinia 状态管理和 Vue Router 路由守卫，实现流畅的用户体验。

**技术栈**：Vue 3.5+ | Vite 5.4+ | Vue Router 4.6+ | Pinia 2.3+ | Axios 1.13+

**开发端口**：`http://localhost:5173`

---

## 技术栈详解

| 技术           | 版本要求 | 当前版本 | 用途说明                  |
| -------------- | -------- | -------- | ------------------------- |
| **Vue**        | 3.5+     | 3.5.13   | 核心框架，响应式系统      |
| **Vite**       | 5.4+     | 5.4.21   | 构建工具，开发服务器，HMR |
| **Vue Router** | 4.6+     | 4.6.4    | 客户端路由，导航守卫      |
| **Pinia**      | 2.3+     | 2.3.1    | 状态管理（替代 Vuex）     |
| **Axios**      | 1.6+     | 1.13.5   | HTTP 客户端，请求拦截器   |
| **ESLint**     | 8.x      | 8.57.0   | 代码质量检查              |

### 核心依赖特性

- **Vue 3 Composition API**: 使用 `ref`、`reactive`、`computed`、`watch` 等组合式 API
- **`<script setup>`**: 语法糖简化组件逻辑，自动暴露属性和方法
- **TypeScript 支持**: 可选，推荐新项目使用（当前为 JavaScript）
- **CSS Modules**: 支持模块化 CSS，避免样式冲突
- **HMR (Hot Module Replacement)**: 开发时实时更新，保留应用状态

---

## 项目结构

```
frontend-vue/
├── index.html                    # 入口 HTML 文件
├── package.json                  # 项目依赖和脚本配置
├── vite.config.js                # Vite 配置（代理、构建优化）
├── .env.example                  # 环境变量示例
├── .env                          # 环境变量（本地开发，不提交）
├── public/                       # 静态资源（不经过 Vite 处理）
│   └── favicon.ico
└── src/
    ├── main.js                   # 应用入口，创建 Pinia 和 Router
    ├── App.vue                   # 根组件，路由出口，全局布局
    ├── style.css                 # 全局样式（CSS 变量、重置）
    ├── api/                      # API 服务层
    │   └── index.js              # Axios 实例、拦截器、错误处理
    ├── router/                   # 路由配置
    │   └── index.js              # 路由表、权限守卫、滚动行为
    ├── stores/                   # Pinia 状态管理
    │   ├── auth.js               # 用户认证状态（token、用户信息）
    │   └── class.js              # 班级状态（当前班级、成员列表）
    ├── components/               # 可复用组件
    │   ├── AIResponseDisplay.vue      # AI 响应解析显示（支持 Markdown）
    │   ├── WeakPointDisplay.vue       # 薄弱点选择组件（多选、分组）
    │   ├── HistoryTabs/               # 历史记录标签页组件（可复用）
    │   │   ├── DebugHistoryTab.vue    # 调试历史列表
    │   │   ├── EvaluateHistoryTab.vue # 评价历史列表
    │   │   ├── RecommendHistoryTab.vue# 推荐历史列表
    │   │   └── HistoryDetailModal.vue # 详情弹窗（支持三种类型）
    │   └── class/                     # 班级管理组件
    │       ├── ClassSelector.vue      # 班级选择器（创建/加入/切换）
    │       ├── ClassInfoTab.vue       # 班级信息展示
    │       ├── ClassManageTab.vue     # 成员管理（添加/移除，批量）
    │       ├── ClassHistoryQueryTab.vue # 学生历史查询（筛选）
    │       └── ClassWeakPointsQueryTab.vue # 班级薄弱点查询（导出）
    └── views/                    # 页面级组件（路由对应）
        ├── Login.vue             # 登录页（学号+密码）
        ├── Register.vue          # 注册页
        ├── Profile.vue           # 个人主页（用户信息、快捷入口）
        ├── ClassManage.vue       # 班级管理页（多标签页布局）
        ├── AIDebug.vue           # AI 对话调试页（4轮流程）
        ├── Evaluate.vue          # AI 代码评价页
        ├── Recommend.vue         # AI 题目推荐页
        └── History.vue           # 历史记录页（标签页切换）
```

---

## 快速开始

### 前置条件

- **Node.js**: 18.0+（Vite 5.x 要求，推荐 20.x LTS）
- **包管理器**: npm 9+ 或 yarn 1.22+ 或 pnpm 8+
- **浏览器**: Chrome 90+、Firefox 88+、Safari 14+、Edge 90+

### 安装依赖

```bash
cd frontend-vue

# 使用 npm
npm ci  # 或 npm install

# 使用 yarn
yarn install --frozen-lockfile

# 使用 pnpm
pnpm install --frozen-lockfile
```

**依赖说明**：
- `vue` / `vue-router` / `pinia` - 核心框架
- `axios` - HTTP 客户端
- `vite` / `@vitejs/plugin-vue` - 构建工具
- `eslint` / `eslint-plugin-vue` - 代码检查（开发依赖）

### 开发模式

```bash
npm run dev

# 或指定端口和主机
npm run dev -- --port 5173 --host localhost
```

访问 `http://localhost:5173` 查看应用。

**开发服务器特性**：
- 热模块替换（HMR）：修改代码实时更新，不丢失状态
- 自动打开浏览器（可选，配置 `open: true`）
- 代理后端 API 到 `http://localhost:8080`（见下文）

### 生产构建

```bash
# 构建生产版本
npm run build

# 预览构建结果（本地静态服务器）
npm run preview
```

构建产物位于 `dist/` 目录，可直接部署到 Nginx、Netlify、Vercel 等静态托管平台。

### 环境变量配置

创建 `.env` 文件（基于 `.env.example`）：

```bash
# API 基础 URL（开发环境，生产环境由 Vite 代理或 Nginx 配置）
VITE_API_BASE_URL=http://localhost:8080

# 应用标题
VITE_APP_TITLE=AI 教学辅助平台
```

**注意**：Vite 仅暴露以 `VITE_` 开头的环境变量。

---

## API 代理配置

### Vite 开发代理（`vite.config.js`）

```javascript
export default defineConfig({
  server: {
    proxy: {
      '/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/auth/, '/auth')
      },
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '/api')
      }
    }
  }
})
```

**代理规则**：

| 路径前缀  | 目标地址                       | 说明                      |
| --------- | ------------------------------ | ------------------------- |
| `/auth/*` | `http://localhost:8080/auth/*` | 认证相关接口（登录/注册） |
| `/api/*`  | `http://localhost:8080/api/*`  | 所有业务 API              |

**生产环境配置**（Nginx）：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location /auth/ {
        proxy_pass http://localhost:8080/auth/;
        proxy_set_header Host $host;
    }

    location /api/ {
        proxy_pass http://localhost:8080/api/;
        proxy_set_header Host $host;
    }

    location / {
        root /path/to/frontend-vue/dist;
        try_files $uri $uri/ /index.html;
    }
}
```

---

## 核心功能模块

### 1. 用户认证

**流程**：
1. 用户在登录页输入学号、密码
2. 调用 `POST /auth/login`，获取 JWT Token 和用户信息
3. Token 存储到 `localStorage`，用户信息存储到 Pinia `auth` store
4. Axios 拦截器自动为后续请求添加 `Authorization: Bearer <token>` 头
5. 401 响应触发自动登出：清除 Token、用户信息，跳转登录页
6. 路由守卫 (`router.beforeEach`) 检查 `meta.requiresAuth`，未认证重定向到 `/login`

**关键代码**：

```javascript
// src/api/index.js - 请求拦截器
axios.interceptors.request.use(
  (config) => {
    const token = store.auth.token
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器 - 处理 401
axios.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      store.auth.clear()
      router.push('/login')
    }
    return Promise.reject(error)
  }
)
```

**持久化策略**：
- `auth_token`: `localStorage`（持久化，关闭浏览器不丢失）
- `user_info`: `localStorage` + Pinia store（内存）
- 登出时清除所有存储

### 2. AI 代码调试（4轮对话）

**页面路由**：`/ai-debug`

**核心组件**：`AIDebug.vue`

**4轮流程详解**：

| 轮次 | 名称         | AI 输出字段                                                        | 用户操作                                | 状态控制                 |
| ---- | ------------ | ------------------------------------------------------------------ | --------------------------------------- | ------------------------ |
| 1    | 理解学生思路 | `student_thought`, `suggested_correction`                          | 阅读反馈，点击"继续"                    | 输入后禁用，进入第2轮    |
| 2    | 指出问题点   | `problem_summary`, `key_issues[]`, `weak_points[]`, `ask_for_help` | 按钮选择或文本输入确认/修正，点击"继续" | 支持按钮+文本，进入第3轮 |
| 3    | 调试指导     | `debug_guidance`, `ask_for_detail`                                 | 按钮选择或文本输入确认/继续，点击"继续" | 支持按钮+文本，进入第4轮 |
| 4    | 详细修改指导 | `suggestions[]`                                                    | 阅读建议，对话自动关闭                  | 只读，不可继续           |

**API 调用**：

```javascript
// 开始新对话
await api.ai.start()

// 发送第 N 轮请求
const response = await api.ai.debugV2({
  conversation_id: conversationId,
  code: code,
  problem_description: problemDesc,
  test_points: testPoints,
  current_round: currentRound,
  dialogue_history: history,
  student_response: userInput  // 第2、3轮使用
})

// 关闭对话（手动）
await api.ai.closeDebug(conversationId)
```

**状态管理**（组件内 `ref`/`reactive`）：
- `conversationId`: 当前对话 ID（新建时调用 `/start` 获取）
- `currentRound`: 当前轮次（1-4）
- `dialogueHistory`: 对话历史数组，按轮次分组
- `userInput`: 用户输入（第2、3轮）
- `isLoading`: 加载状态
- `errorMessage`: 错误信息

**轮次信息获取**：`GET /api/v1/ai/round_info` 返回当前轮次的标题、描述、提示，用于动态显示 UI。

### 3. AI 代码评价

**页面路由**：`/evaluate`

**核心组件**：`Evaluate.vue`

**输入**：
- 题目描述（textarea）
- 代码（textarea，支持 C/C++/Python）
- 测试点（可选，每行一个，格式：`输入/输出/状态`）

**输出**：
- 整体评价（`overall_evaluation`）
- 四个维度评分（`functional_correctness`、`logical_rigor`、`algorithm_quality`、`structural_normativity`）
- 每个维度包含 `score`（字符串，如 "85"）和 `comment`

**API 调用**：

```javascript
const response = await api.ai.evaluate({
  code: code,
  problem_description: problemDesc,
  test_points: testPointsArray  // [{input, status}, ...]
})
```

**结果展示**：
- 使用 `AIResponseDisplay.vue` 组件解析和显示 AI 响应
- 支持 Markdown 渲染（如使用 `markdown-it` 或 `marked`）
- 评分高亮显示（如分数 >= 90 绿色，70-89 黄色，<70 红色）

### 4. AI 题目推荐

**页面路由**：`/recommend`

**核心组件**：`Recommend.vue`

**功能特性**：
- **薄弱点选择**：使用 `WeakPointDisplay.vue` 组件，从用户历史薄弱点中多选
- **日期筛选**：选择开始/结束日期，统计指定时间段内的薄弱点
- **Top K 控制**：设置显示前 N 个薄弱点（0 = 显示全部）
- **推荐数量**：调整推荐题目数量（3/5/8/10 下拉选择）
- **推荐结果**：显示题目列表、推荐理由、相关度评分

**API 调用**：

```javascript
// 获取用户薄弱点（支持日期筛选）
const weakPoints = await api.ai.getWeakPoints({
  start_date: startDate,
  end_date: endDate
})

// 获取 Top N 薄弱点
const topWeakPoints = await api.ai.getTopWeakPoints({
  top_k: 5,
  start_date: startDate,
  end_date: endDate
})

// 推荐题目
const recommendations = await api.ai.recommend({
  weak_points: selectedWeakPoints,  // { "循环": 3, "数组": 2 }
  max_recommendations: maxCount
})
```

**状态管理**：
- `selectedWeakPoints`: 选中的薄弱点对象（`{keyword: count}`）
- `dateRange`: 日期范围 `{startDate, endDate}`
- `topK`: Top K 值（默认 5）
- `recommendCount`: 推荐数量（默认 5）
- `recommendations`: 推荐结果数组
- `isLoading`: 加载状态

### 5. 历史记录

**页面路由**：`/history`

**核心组件**：`History.vue` + `HistoryTabs/` 系列组件

**功能**：
- 标签页切换：调试历史、评价历史、推荐历史
- 按会话（conversation_id）分组展示（调试历史）或按记录分组（评价/推荐）
- 显示每会话的轮次、记录数、最新时间
- 查看详情弹窗，展示完整对话/评价/推荐内容
- 详情中包含题目描述、提交代码、AI 响应等完整信息

**可复用组件设计**：

| 组件                      | 用途             | Props                                                         | 事件            |
| ------------------------- | ---------------- | ------------------------------------------------------------- | --------------- |
| `DebugHistoryTab.vue`     | 调试历史列表     | `records: Array`                                              | `@view-details` |
| `EvaluateHistoryTab.vue`  | 评价历史列表     | `records: Array`                                              | `@view-details` |
| `RecommendHistoryTab.vue` | 推荐历史列表     | `records: Array`                                              | `@view-details` |
| `HistoryDetailModal.vue`  | 详情弹窗（通用） | `records: Array`, `initialSubmission: Object`, `type: string` | `@close`        |

**复用示例**（班级管理中的学生历史查询）：

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

**API 调用**：

```javascript
// 获取所有历史（分类型）
const debugRecords = await api.ai.getRecords({ type: 'debug' })
const evaluateRecords = await api.ai.getRecords({ type: 'evaluate' })
const recommendRecords = await api.ai.getRecords({ type: 'recommend' })
```

### 6. 班级管理

**页面路由**：`/class-manage`

**核心组件**：`ClassManage.vue` + `class/` 系列组件

**功能模块**：

#### 6.1 班级选择器（`ClassSelector.vue`）

- 显示当前用户所属班级列表
- 创建新班级（仅 `admin`）
- 加入现有班级（输入班级 ID）
- 切换当前班级（用于后续操作）

**状态输出**：`@select` 事件返回当前选中的班级对象。

#### 6.2 班级信息（`ClassInfoTab.vue`）

展示：
- 班级 ID、名称、创建者
- 创建时间、成员数量
- 当前用户角色（创建者/教师/助教/学生）

#### 6.3 成员管理（`ClassManageTab.vue`）

**功能**：
- 添加成员（批量输入学号列表，选择角色）
- 移除成员（多选学生列表，批量移除）
- 显示成员列表（学号、用户名、角色、是否创建者）

**权限限制**（前端控制）：
- **助教**：角色选择器仅显示"学生"选项；移除时仅显示学生列表
- **学生**：隐藏整个成员管理模块（后端也会验证）

**API 调用**：

```javascript
// 获取班级成员
const members = await api.class.getMembers(classId)

// 添加成员（批量）
await api.class.addMembers(classId, {
  student_ids: ['2023001', '2023002'],
  member_role: 'student'  // 'student' | 'ta' | 'teacher'
})

// 移除成员（批量）
await api.class.removeMembers(classId, {
  student_ids: ['2023003']
})
```

#### 6.4 学生历史查询（`ClassHistoryQueryTab.vue`）

**功能**：
- 选择学生（多选，默认全选）
- 选择历史类型（debug/evaluate/recommend）
- 选择时间范围（开始/结束日期）
- 查询并展示历史记录（复用 `HistoryTabs/` 组件）

**API 调用**：

```javascript
const records = await api.class.getHistory(classId, {
  student_ids: selectedStudents,  // 不传则查询全班
  type: 'debug',
  start_date: startDate,
  end_date: endDate
})
```

#### 6.5 班级薄弱点查询（`ClassWeakPointsQueryTab.vue`）

**功能**：
- 统计班级整体薄弱点（按学生、时间筛选）
- 支持导出 JSON（调用 `/ai/weak_points/class` + 下载）
- 显示每个薄弱点的出现次数和涉及学生数

**API 调用**：

```javascript
const weakPoints = await api.ai.getClassWeakPoints(classId, {
  student_ids: selectedStudents,
  start_date: startDate,
  end_date: endDate
})
```

---

## 路由权限

**路由配置**（`src/router/index.js`）：

```javascript
const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/Register.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: () => import('@/views/Profile.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/class-manage',
    name: 'ClassManage',
    component: () => import('@/views/ClassManage.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/ai-debug',
    name: 'AIDebug',
    component: () => import('@/views/AIDebug.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/evaluate',
    name: 'Evaluate',
    component: () => import('@/views/Evaluate.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/recommend',
    name: 'Recommend',
    component: () => import('@/views/Recommend.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/history',
    name: 'History',
    component: () => import('@/views/History.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/',
    redirect: '/profile'
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue')
  }
]
```

**权限守卫**：

```javascript
router.beforeEach((to, from, next) => {
  const isAuthenticated = store.auth.isAuthenticated

  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.name === 'Login' && isAuthenticated) {
    next({ name: 'Profile' })
  } else {
    next()
  }
})
```

---

## API 集成

### API 服务封装（`src/api/index.js`）

```javascript
import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 120000,  // 120 秒，匹配后端超时
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器：添加 Token
api.interceptors.request.use(
  (config) => {
    const authStore = useAuthStore()
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器：处理 401、错误统一格式
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      const authStore = useAuthStore()
      authStore.clear()
      window.location.href = '/login'
    }
    // 统一错误格式
    const message = error.response?.data?.message || error.message
    return Promise.reject(new Error(message))
  }
)

// 认证相关
export const authApi = {
  register: (data) => api.post('/auth/register', data),
  login: (data) => api.post('/auth/login', data),
  logout: () => api.post('/auth/logout')
}

// 用户相关
export const userApi = {
  getProfile: () => api.get('/api/v1/profile')
}

// AI 服务相关
export const aiApi = {
  // 调试
  startDebug: () => api.post('/api/v1/ai/start'),
  debugV2: (data) => api.post('/api/v1/ai/debug_v2', data),
  closeDebug: (conversationId) => api.post('/api/v1/ai/debug/close', { conversation_id: conversationId }),
  getRoundInfo: () => api.get('/api/v1/ai/round_info'),

  // 评价
  evaluate: (data) => api.post('/api/v1/ai/evaluate', data),

  // 推荐
  recommend: (data) => api.post('/api/v1/ai/recommend', data),

  // 薄弱点
  getWeakPoints: (params) => api.get('/api/v1/ai/weak_points', { params }),
  getTopWeakPoints: (params) => api.get('/api/v1/ai/weak_points/top', { params }),
  getClassWeakPoints: (classId, params) => api.get(`/api/v1/ai/weak_points/class?class_id=${classId}`, { params }),

  // 历史记录
  getRecords: (params) => api.get('/api/v1/ai/records', { params }),
  getDebugRecords: (params) => api.get('/api/v1/ai/records/debug', { params }),
  getEvaluateRecords: (params) => api.get('/api/v1/ai/records/evaluate', { params }),
  getRecommendRecords: (params) => api.get('/api/v1/ai/records/recommend', { params })
}

// 班级管理相关
export const classApi = {
  // 班级列表
  getAllClasses: () => api.get('/api/v1/classes'),
  getMyClasses: () => api.get('/api/v1/classes/my'),
  createClass: (data) => api.post('/api/v1/classes', data),
  getClass: (classId) => api.get(`/api/v1/classes/${classId}`),
  joinClass: (classId) => api.post(`/api/v1/classes/${classId}/join`),

  // 成员管理
  getMembers: (classId) => api.get(`/api/v1/classes/${classId}/members`),
  addMembers: (classId, data) => api.post(`/api/v1/classes/${classId}/members/add`, data),
  removeMembers: (classId, data) => api.post(`/api/v1/classes/${classId}/members/remove`, data),

  // 历史记录查询
  getClassHistory: (classId, params) => api.get(`/api/v1/classes/${classId}/records/debug`, { params }),
  getClassEvaluateHistory: (classId, params) => api.get(`/api/v1/classes/${classId}/records/evaluate`, { params }),
  getClassRecommendHistory: (classId, params) => api.get(`/api/v1/classes/${classId}/records/recommend`, { params }),
  exportClassHistory: (classId, type, params) => api.get(`/api/v1/classes/${classId}/records/${type}/export`, { params, responseType: 'blob' })
}

// 默认导出
export default api
```

---

## 状态管理（Pinia）

### Auth Store（`src/stores/auth.js`）

```javascript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  // 状态
  const token = ref(localStorage.getItem('auth_token') || '')
  const user = ref(JSON.parse(localStorage.getItem('user_info') || 'null'))

  // 计算属性
  const isAuthenticated = computed(() => !!token.value)
  const userId = computed(() => user.value?.id)
  const userType = computed(() => user.value?.user_type)  // 'admin' | 'user'
  const username = computed(() => user.value?.username)

  // 动作
  async function login(credentials) {
    const data = await authApi.login(credentials)
    token.value = data.token
    user.value = data.user
    _persist()
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      clear()
    }
  }

  function clear() {
    token.value = ''
    user.value = null
    localStorage.removeItem('auth_token')
    localStorage.removeItem('user_info')
  }

  function _persist() {
    localStorage.setItem('auth_token', token.value)
    localStorage.setItem('user_info', JSON.stringify(user.value))
  }

  return {
    token,
    user,
    isAuthenticated,
    userId,
    userType,
    username,
    login,
    logout,
    clear
  }
})
```

### Class Store（`src/stores/class.js`）

```javascript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { classApi } from '@/api'

export const useClassStore = defineStore('class', () => {
  // 状态
  const currentClassId = ref(null)
  const myClasses = ref([])
  const currentClassMembers = ref([])

  // 计算属性
  const currentClass = computed(() =>
    myClasses.value.find(c => c.id === currentClassId.value)
  )

  // 动作
  async function fetchMyClasses() {
    const data = await classApi.getMyClasses()
    myClasses.value = data.classes
    if (myClasses.value.length > 0 && !currentClassId.value) {
      currentClassId.value = myClasses.value[0].id
    }
  }

  function setCurrentClass(classId) {
    currentClassId.value = classId
  }

  async function fetchClassMembers(classId) {
    const data = await classApi.getMembers(classId)
    currentClassMembers.value = data.members
  }

  return {
    currentClassId,
    myClasses,
    currentClassMembers,
    currentClass,
    fetchMyClasses,
    setCurrentClass,
    fetchClassMembers
  }
})
```

---

## 开发规范

### 代码风格

- **ESLint**: 使用项目配置（`.eslintrc.cjs`），运行 `npm run lint` 检查
- **Prettier**: 可选，统一代码格式（推荐配置 `.prettierrc`）
- **Vue 3 推荐**:
  - 使用 `<script setup>` 语法
  - 使用 `ref` 而非 `reactive` 处理基本类型
  - 组件名使用大驼峰（`PascalCase`）
  - Props 使用 `camelCase`，模板中使用 `kebab-case`

**示例组件**：

```vue
<template>
  <div class="my-component">
    <h1>{{ title }}</h1>
    <button @click="handleClick">点击</button>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'

const props = defineProps({
  title: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['click'])

const router = useRouter()
const count = ref(0)

const doubleCount = computed(() => count.value * 2)

function handleClick() {
  count.value++
  emit('click', count.value)
  router.push('/profile')
}
</script>

<style scoped>
.my-component {
  padding: 16px;
}
</style>
```

### 组件设计原则

- **单一职责**: 每个组件只负责一个明确的 UI 功能
- **Props 最小化**: 仅传递必要数据，避免深层嵌套 props
- **事件驱动**: 子组件通过 `emit` 通知父组件，而非直接修改父级状态
- **可复用性**: 通用组件（如 `HistoryDetailModal`）应独立于业务逻辑
- **组合优于继承**: 使用组件组合（slot、嵌套）而非复杂继承

### 状态管理策略

- **组件内状态**: 使用 `ref`/`reactive`，仅当前组件使用
- **跨组件共享**: 使用 Pinia store（如 `auth`、`class`）
- **URL 状态**: 路由参数、查询参数（如筛选条件）
- **持久化状态**: `localStorage`（如 Token、用户信息）

### API 调用规范

- **统一通过 `src/api/index.js`**: 避免硬编码 URL
- **错误处理**: 使用 `try/catch`，显示用户友好错误信息
- **加载状态**: 所有异步操作应有 `isLoading` 状态
- **请求取消**: 使用 `AbortController` 或 Axios `CancelToken` 取消未完成请求
- **请求去重**: 相同请求短时间内不重复发起（可选）

---

## 测试

### 单元测试（Vitest）

```bash
# 安装依赖
npm install -D vitest @vue/test-utils @testing-library/vue

# 运行测试
npm run test

# 监控模式
npm run test -- --watch

# 覆盖率
npm run test -- --coverage
```

**测试示例**（组件测试）：

```javascript
// tests/components/WeakPointDisplay.spec.js
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import WeakPointDisplay from '@/components/WeakPointDisplay.vue'

describe('WeakPointDisplay', () => {
  it('应该渲染所有薄弱点选项', () => {
    const wrapper = mount(WeakPointDisplay, {
      props: {
        weakPoints: [
          { keyword: '循环', count: 3 },
          { keyword: '数组', count: 2 }
        ]
      }
    })
    expect(wrapper.text()).toContain('循环')
    expect(wrapper.text()).toContain('数组')
  })
})
```

### E2E 测试（Cypress / Playwright）

```bash
# 安装 Cypress
npm install -D cypress

# 打开 Cypress UI
npx cypress open

# 运行所有测试
npx cypress run
```

**测试场景**：
- 用户登录/登出流程
- AI 调试 4 轮对话流程
- 班级创建、加入、成员管理
- 历史记录查看和导出

---

## 性能优化

### 构建优化

- **代码分割**: 使用 `import()` 动态导入路由组件（Vue Router 自动处理）
- **Tree Shaking**: 确保 ES Module 导出，避免 `import * as` 引入整个库
- **压缩**: Vite 生产构建自动压缩 JS/CSS
- **图片优化**: 使用 `vite-plugin-imagemin` 压缩图片
- **Gzip/Brotli**: Nginx 配置压缩

### 运行时优化

- **虚拟列表**: 长列表（如历史记录）使用 `vue-virtual-scroller`
- **图片懒加载**: 使用 `loading="lazy"` 或 `IntersectionObserver`
- **防抖/节流**: 搜索输入、窗口调整等事件使用 `lodash/debounce`、`lodash/throttle`
- **缓存策略**: API 响应缓存（如 `swrv`、`vue-query`）
- **Web Worker**: 复杂计算（如代码解析）放入 Worker

### 性能监控

- **Lighthouse**: 定期审计性能、可访问性、SEO
- **Chrome DevTools**: Performance 面板分析运行时性能
- **Vite 分析插件**: `rollup-plugin-visualizer` 查看包大小

---

## 常见问题

### 1. 登录后跳转回登录页面

**原因**：
- Go 后端未运行或不可访问
- CORS 错误（开发环境由 Vite 代理解决）
- Token 未正确存储
- 后端 `/auth/login` 返回格式不包含 `token` 字段

**排查**：
1. 确认 Go 后端运行在 `http://localhost:8080`
2. 检查浏览器控制台 Network 面板，查看登录请求响应
3. 验证 `localStorage` 中是否有 `auth_token`
4. 确认后端返回 `{ token: "...", user: {...} }`

### 2. API 请求失败（404/500）

**原因**：
- 后端服务未启动
- 请求路径错误
- Vite 代理配置不正确
- 跨域问题（生产环境）

**排查**：
1. 访问 `http://localhost:8080/health` 确认后端运行
2. 检查 Network 面板请求 URL 是否正确
3. 验证 `vite.config.js` 代理配置
4. 生产环境检查 Nginx 反向代理配置

### 3. 前端无法启动（npm run dev）

**原因**：
- Node.js 版本过低（< 18.0）
- `node_modules` 损坏
- 依赖版本冲突

**解决**：
```bash
# 检查 Node 版本
node --version  # 应 >= 18.0

# 清理并重装
rm -rf node_modules package-lock.json  # Windows: rmdir /s node_modules
npm install

# 或使用 npm 修复
npm ci --force
```

### 4. 班级管理权限问题

**原因**：
- 用户角色不是管理员/教师/助教
- 后端权限验证失败

**排查**：
1. 确认用户 `user_type` 为 `admin` 或 `user`
2. 确认用户在班级中的 `member_role` 为 `teacher` 或 `ta`
3. 学生用户（`member_role: student`）只能查看，不能管理

### 5. 历史记录详情显示异常

**原因**：
- 后端返回的数据结构缺少 `initial_submission` 字段
- 详情弹窗组件 `type` 参数不正确

**排查**：
1. 检查 API 响应是否包含 `initial_submission`（题目描述和代码）
2. 确认 `@view-details` 事件传递的 `type` 为 `'debug'`、`'evaluate'` 或 `'recommend'`
3. 查看浏览器控制台是否有组件渲染错误

---

## 开发建议

- **Vue DevTools**: 安装浏览器插件，调试组件状态和 Pinia store
- **网络请求**: 使用浏览器 DevTools Network 面板查看 API 请求/响应
- **组件开发**: 优先使用 `<script setup>`，保持代码简洁
- **状态管理**: 认证信息用 `auth` store，班级数据用 `class` store
- **API 调用**: 统一通过 `@/api` 模块，避免硬编码 URL
- **错误处理**: 所有 API 调用使用 `try/catch`，显示 `Toast` 或 `Message` 提示
- **路由懒加载**: 使用动态 `import()` 实现按需加载
- **环境区分**: 使用 `.env.development`、`.env.production` 区分环境配置
- **Git 提交**: 遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范

---

## 相关项目

- **[Go 后端服务](../backend-go/README.md)** - 中介服务、Worker Pool、限流、权限
- **[Python AI 服务](../ai-python/README.md)** - 核心 AI 能力（评价、推荐、调试）
- **[项目总览](../README.md)** - 整体架构和快速启动

---

## 许可证

MIT License
