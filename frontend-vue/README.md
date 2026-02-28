# Frontend Vue - AI 教学辅助平台前端

基于 Vue 3.4+ 和 Vite 5.2+ 构建的现代化单页应用（SPA）。

**技术栈**：Vue 3.4+ | Vite 5.2+ | Vue Router 4.3+ | Pinia 2.1+ | Axios 1.6+

**开发端口**：`http://localhost:5173`

**核心依赖版本**（见 [`package.json`](package.json:1)）：
- `vue: ^3.4.21` - 渐进式框架
- `vite: ^5.2.8` - 构建工具
- `vue-router: ^4.3.0` - 路由管理
- `pinia: ^2.1.7` - 状态管理
- `axios: ^1.6.8` - HTTP 客户端
- `echarts: ^6.0.0` - 数据可视化
- `vue-echarts: ^8.0.1` - ECharts Vue 组件

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

#### 方式一：使用 Vite 代理（推荐开发环境）

项目已配置 [`vite.config.js`](vite.config.js:1) 代理，无需手动设置 `VITE_API_BASE_URL`：

```javascript
server: {
  port: 5173,
  proxy: {
    '/api': { target: 'http://localhost:8080' },
    '/auth': { target: 'http://localhost:8080' }
  }
}
```

开发时所有 `/api` 和 `/auth` 请求自动代理到后端。

#### 方式二：手动配置环境变量

创建 `.env` 文件（生产环境或禁用代理时使用）：

```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_APP_TITLE=AI 教学辅助平台
```

**注意**：
- 使用代理时，API 调用使用相对路径（如 `/api/v1/ai/evaluate`）
- 不使用代理时，需设置 `VITE_API_BASE_URL` 并配置 `src/api/index.js` 中的 `baseURL`

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

**路由**：`/profile/classes`（在个人主页内）

**模块**：
- **班级选择器**（`ClassSelector.vue`）：创建/加入/切换班级
- **班级信息**（`ClassInfoTab.vue`）：展示班级详情和当前用户角色
- **成员管理**（`ClassManageTab.vue`）：添加/移除成员（批量），助教仅可管理学生
- **学生历史查询**（`ClassHistoryQueryTab.vue`）：按学生、类型、时间筛选查询历史
- **班级薄弱点**（`ClassWeakPointsQueryTab.vue`）：统计整体薄弱点，支持导出 JSON

**权限控制**：
- 仅班级成员可访问
- 教师/创建者可管理成员
- 助教仅可管理学生成员
- 创建者（`is_creator=true`）不可被移除

## 路由权限

所有需要认证的路由均设置 `meta.requiresAuth = true`，路由守卫（见 [`src/router/index.js`](src/router/index.js:64)）检查用户认证状态：

- 未认证用户访问认证路由 → 重定向到 `/login`
- 已认证用户访问 `/login` 或 `/register` → 重定向到 `/profile`

**公开路由**（`requiresAuth: false`）：
- `/login` - 登录页
- `/register` - 注册页

**认证路由**（`requiresAuth: true`）：
- `/profile` - 个人主页
- `/profile/classes` - 班级管理（实际路径）
- `/ai-debug` - AI 调试
- `/evaluate` - 代码评价
- `/recommend` - 题目推荐
- `/history` - 历史记录

**注意**：文档中提到的 `/class-manage` 路由实际为 `/profile/classes`（见 [`src/router/index.js`](src/router/index.js:52)）。

## API 集成

### API 服务封装（`src/api/index.js`）

统一管理所有 API 调用，核心特性：

**Axios 配置**：
- `baseURL: ''` - 使用相对路径，配合 Vite 代理
- `timeout: 60000` - 60 秒超时
- 响应拦截器自动解包 `response.data`

**请求拦截器**：
- 自动从 `localStorage` 读取 Token
- 添加 `Authorization: Bearer <token>` 头

**响应拦截器**：
- 401 错误自动清除本地存储并跳转登录页
- 统一错误处理，返回 `Promise.reject(data)`

**API 模块**：
- `authAPI` - 注册、登录、登出
- `userAPI` - 获取用户信息
- `aiAPI` - 所有 AI 相关操作（调试、评价、推荐、历史、薄弱点）
- `classAPI` - 班级管理、成员管理、历史查询

### 状态管理（Pinia）

#### Auth Store（`src/stores/auth.js`）

**状态**：
- `token` - JWT Token（持久化）
- `user` - 用户信息（持久化）
- `isAuthenticated` - 计算属性（`!!token`）
- `userType` - 计算属性（`user.user_type`）
- `username` - 计算属性（`user.username`）

**动作**：
- `login(token, user)` - 登录后设置状态
- `logout()` - 清除状态并跳转登录
- `clear()` - 仅清除状态（不跳转）

#### Class Store（`src/stores/class.js`）

**状态**：
- `currentClassId` - 当前选中的班级 ID
- `myClasses` - 用户所属班级列表
- `currentClassMembers` - 当前班级成员列表
- `currentClass` - 计算属性（根据 `currentClassId` 查找）

**动作**：
- `fetchMyClasses()` - 获取我的班级列表
- `setCurrentClass(classId)` - 设置当前班级并自动获取成员
- `fetchClassMembers(classId?)` - 获取班级成员列表
- `clearCurrentClass()` - 清除当前班级状态

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

### 组件开发示例

**推荐使用 `<script setup>` 语法**：
```vue
<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import { aiAPI } from '../api'

const authStore = useAuthStore()
const loading = ref(false)
const data = ref(null)

onMounted(async () => {
  loading.value = true
  try {
    data.value = await aiAPI.getRecords()
  } catch (error) {
    console.error('Failed to fetch records:', error)
  } finally {
    loading.value = false
  }
})
</script>
```

## 生产部署

### 构建优化

```bash
# 生产构建
npm run build

# 预览构建结果
npm run preview
```

构建产物位于 `dist/` 目录，可直接部署到任何静态文件服务器（Nginx、Apache、Vercel、Netlify 等）。

### Nginx 配置示例

```nginx
server {
    listen 80;
    server_name your-domain.com;
    root /path/to/frontend-vue/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 代理到后端
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /auth/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 环境变量（生产）

生产环境需设置正确的 `VITE_API_BASE_URL`：

```bash
# .env.production
VITE_API_BASE_URL=https://api.your-domain.com
VITE_APP_TITLE=AI 教学辅助平台
```

构建时自动加载：
```bash
npm run build -- --mode production
```

## 相关项目

- **[Go 后端服务](../backend-go/README.md)** - 中介服务、Worker Pool、限流
- **[Python AI 服务](../ai-python/README.md)** - 核心 AI 能力
- **[项目总览](../README.md)** - 整体架构和快速启动

## 许可证

MIT License
