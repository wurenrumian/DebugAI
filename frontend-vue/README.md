# Frontend Vue - AI教学辅助平台前端

## 概述

基于 Vue 3 + Vite 构建的前端应用，为 AI 教学辅助平台提供用户界面。与 Go 后端 ([`backend-go`](backend-go/README.md)) 和 Python AI 服务 ([`ai-python`](ai-python/README.md)) 配合，提供学生登录注册、AI 代码调试、代码评价、题目推荐、历史记录等功能。

## 技术栈

- **框架**: Vue 3.4+ (Composition API)
- **构建工具**: Vite 5.2+
- **路由**: Vue Router 4.3+
- **状态管理**: Pinia 2.1+
- **HTTP 客户端**: Axios 1.6+
- **样式**: 原生 CSS

## 项目结构

```
frontend-vue/
├── index.html              # 入口 HTML
├── package.json            # 项目依赖配置
├── vite.config.js         # Vite 配置 (代理后端 API)
└── src/
    ├── main.js            # 应用入口
    ├── App.vue            # 根组件
    ├── style.css          # 全局样式
    ├── api/
    │   └── index.js       # API 服务封装 (Axios)
    ├── router/
    │   └── index.js       # 路由配置 (含权限守卫)
    ├── stores/
    │   └── auth.js        # 用户认证状态管理
    ├── components/
    │   └── HistoryTabs/   # 历史记录标签页组件
    │       ├── DebugHistoryTab.vue
    │       ├── EvaluateHistoryTab.vue
    │       └── RecommendHistoryTab.vue
    └── views/
        ├── Login.vue       # 登录页面
        ├── Register.vue    # 注册页面
        ├── Profile.vue     # 个人主页
        ├── AIDebug.vue     # AI 对话调试页面
        ├── Evaluate.vue    # AI 代码评价页面
        ├── Recommend.vue   # AI 题目推荐页面
        └── History.vue     # 对话历史页面
```

## 快速开始

### 前置条件

- Node.js 16.0+
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

构建产物将生成在 `dist/` 目录。

## API 代理配置

Vite 开发服务器代理配置：

| 路径      | 目标                           |
| --------- | ------------------------------ |
| `/auth/*` | `http://localhost:8080/auth/*` |
| `/api/*`  | `http://localhost:8080/api/*`  |

所有以 `/auth` 和 `/api` 开头的请求将被转发到 Go 后端。

## 核心功能

### 1. 用户认证

- 学号+密码登录/注册
- JWT Token 自动管理
- 路由权限守卫

### 2. AI 代码调试 ([`/ai-debug`](src/views/AIDebug.vue))

**4轮对话流程**：

| 轮次 | 名称         | 交互方式                   |
| ---- | ------------ | -------------------------- |
| 1    | 理解学生思路 | AI 分析代码，学生确认      |
| 2    | 指出问题点   | AI 指出问题，学生选择/修正 |
| 3    | 调试指导     | AI 提供指导，学生确认      |
| 4    | 详细修改指导 | AI 提供详细建议，结束对话  |

**API 端点**：
- `POST /api/v1/ai/debug_v2` - 发送调试请求
- `GET /api/v1/ai/round_info` - 获取轮次信息
- `POST /api/v1/ai/start` - 开始新对话
- `POST /api/v1/ai/debug/close` - 关闭对话

**特性**：
- 实时轮次状态和提示
- 对话历史展示
- 支持新建对话
- 第2轮支持按钮选择/文本输入
- 第3轮支持按钮选择/文本输入
- 第4轮显示详细修改建议

### 3. AI 代码评价 ([`/evaluate`](src/views/Evaluate.vue))

**API 端点**：
- `POST /api/v1/ai/evaluate` - 提交代码评价

**评价维度**：
- 功能正确性
- 逻辑严谨性
- 算法质量
- 结构规范性

### 4. AI 题目推荐 ([`/recommend`](src/views/Recommend.vue))

**API 端点**：
- `POST /api/v1/ai/recommend` - 获取推荐题目
- `GET /api/v1/ai/weak_points` - 获取用户薄弱点
- `GET /api/v1/ai/weak_points/top` - 获取Top 5薄弱点

**特性**：
- 基于薄弱点智能推荐
- 可调整推荐数量 (3/5/8/10)
- 显示推荐理由和相关度

### 5. 历史记录 ([`/history`](src/views/History.vue))

**API 端点**：
- `GET /api/v1/ai/records` - 获取所有记录
- `GET /api/v1/ai/records/debug` - 调试记录
- `GET /api/v1/ai/records/evaluate` - 评价记录
- `GET /api/v1/ai/records/recommend` - 推荐记录

**特性**：
- 标签页切换 (调试/评价/推荐)
- 按类型筛选
- 查看详细内容

### 6. 个人主页 ([`/profile`](src/views/Profile.vue))

- 用户信息展示
- 功能快捷入口
- 退出登录

## 路由权限

| 路由         | 需要认证 | 说明         |
| ------------ | -------- | ------------ |
| `/login`     | 否       | 登录页面     |
| `/register`  | 否       | 注册页面     |
| `/profile`   | 是       | 个人主页     |
| `/ai-debug`  | 是       | AI 调试页面  |
| `/evaluate`  | 是       | AI 评价页面  |
| `/recommend` | 是       | AI 推荐页面  |
| `/history`   | 是       | 历史记录页面 |

未登录用户访问受保护路由时，自动重定向到登录页面。

## API 集成

### 认证流程

1. 登录成功后，后端返回 JWT Token
2. Token 存储在 `localStorage`
3. Axios 请求拦截器自动添加 `Authorization: Bearer <token>`
4. 401 响应自动清除 Token 并跳转登录

### 主要 API 端点

**认证**：
- `POST /auth/register` - 用户注册
- `POST /auth/login` - 用户登录
- `POST /auth/logout` - 用户登出

**用户**：
- `GET /api/v1/profile` - 获取用户信息

**AI 服务**：
- `POST /api/v1/ai/debug_v2` - AI 调试 (v2)
- `GET /api/v1/ai/round_info` - 获取轮次信息
- `POST /api/v1/ai/start` - 开始新对话
- `POST /api/v1/ai/debug/close` - 关闭对话
- `POST /api/v1/ai/evaluate` - 代码评价
- `POST /api/v1/ai/recommend` - 题目推荐
- `GET /api/v1/ai/weak_points` - 获取薄弱点
- `GET /api/v1/ai/weak_points/top` - 获取Top 5薄弱点
- `GET /api/v1/ai/records` - 获取所有历史记录
- `GET /api/v1/ai/records/debug` - 调试历史
- `GET /api/v1/ai/records/evaluate` - 评价历史
- `GET /api/v1/ai/records/recommend` - 推荐历史

## 常见问题

### 1. 登录后跳转回登录页面
- 确认 Go 后端运行在 `http://localhost:8080`
- 检查浏览器控制台 CORS 错误
- 验证 Token 是否正确存储

### 2. API 请求失败
- 确认后端服务运行状态
- 检查网络请求路径和参数
- 查看后端服务日志

### 3. 前端无法启动
- 确认 Node.js 版本 >= 16.0
- 删除 `node_modules` 和 `package-lock.json`，重新执行 `npm install`

## 开发建议

- 使用 Vue DevTools 调试组件状态
- 浏览器开发者工具查看网络请求
- Go 后端: `http://localhost:8080`
- Python AI 服务: `http://localhost:8000`

## 相关项目

- [后端 Go 服务](../backend-go/README.md)
- [Python AI 服务](../ai-python/README.md)

## 许可证

MIT License
