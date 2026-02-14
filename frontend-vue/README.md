# Frontend Vue - AI教学辅助平台前端

## 概述

这是一个基于 Vue 3 + Vite 构建的前端应用，为 AI 教学辅助平台提供用户界面。该应用与 Go 后端 (`backend-go`) 和 Python AI 服务 (`ai-python`) 配合使用，提供学生登录注册、AI 代码调试、历史记录查看等功能。

## 功能特性

- **用户认证**: 学号+密码登录，注册新账户
- **个人主页**: 查看用户信息，快速访问各项功能
- **AI 代码调试**: 
  - 支持最多 4 轮对话的智能代码调试
  - 实时显示轮次信息和对话进度
  - 支持代码、问题描述、测试点输入
- **对话历史**: 查看所有 AI 交互历史记录，支持按会话分组

## 技术栈

- **框架**: Vue 3 (Composition API)
- **构建工具**: Vite 5
- **路由**: Vue Router 4
- **状态管理**: Pinia
- **HTTP 客户端**: Axios
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
    └── views/
        ├── Login.vue       # 登录页面
        ├── Register.vue    # 注册页面
        ├── Profile.vue     # 个人主页
        ├── AIDebug.vue     # AI 对话调试页面
        └── History.vue     # 对话历史页面
```

## 快速开始

### 前置条件

- Node.js 16.0 或更高版本
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

前端服务将在 `http://localhost:5173` 启动，并自动代理 API 请求到 `http://localhost:8080` (Go 后端)。

### 生产构建

```bash
npm run build
```

构建产物将生成在 `dist/` 目录。

## API 代理配置

Vite 配置了开发环境的代理，将请求转发到 Go 后端：

| 路径      | 目标                           |
| --------- | ------------------------------ |
| `/auth/*` | `http://localhost:8080/auth/*` |
| `/api/*`  | `http://localhost:8080/api/*`  |

## 页面功能详解

### 1. 登录页面 (`/login`)

- 学号和密码登录
- 登录成功后自动保存 JWT Token
- 跳转到个人主页

### 2. 注册页面 (`/register`)

- 新用户注册
- 验证：学号、用户名、密码
- 密码确认校验

### 3. 个人主页 (`/profile`)

- 显示用户信息（学号、用户名、账户类型）
- 快捷入口：AI 调试、历史记录
- 退出登录功能

### 4. AI 调试页面 (`/ai-debug`)

#### 对话流程 (4 轮)

| 轮次 | 名称         | 说明                                  |
| ---- | ------------ | ------------------------------------- |
| 1    | 理解学生思路 | AI 分析代码，理解解题思路             |
| 2    | 指出问题点   | AI 结合确认结果指出问题点和薄弱点     |
| 3    | 调试指导     | AI 提供调试要点，引导思考             |
| 4    | 详细修改指导 | AI 提供详细修改建议（不提供完整代码） |

#### 功能特性

- 左侧输入：问题描述、代码、测试点
- 右侧输出：对话历史、AI 回复
- 实时轮次信息和提示
- 支持新建对话

### 5. 历史记录页面 (`/history`)

- 按会话分组显示所有 AI 交互记录
- 显示每条记录的时间、轮次、状态
- 支持查看详细请求/响应内容

## 与后端集成

### 认证流程

1. 用户登录成功后，后端返回 JWT Token
2. 前端将 Token 存储在 `localStorage`
3. 每次 API 请求在请求头中携带 `Authorization: Bearer <token>`
4. Token 过期或无效时自动跳转登录页面

### 请求/响应格式

#### 登录请求
```json
{
  "student_id": "12345678",
  "password": "password123"
}
```

#### 登录响应
```json
{
  "message": "登录成功",
  "data": {
    "username": "testuser",
    "user_type": "student",
    "student_id": "12345678",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

#### AI 调试请求
```json
{
  "student_id": "12345678",
  "conversation_id": "conv_1234567890",
  "code": "int main() {...}",
  "problem_description": "实现一个排序算法",
  "test_points": [],
  "current_round": 1,
  "dialogue_history": [],
  "student_response": ""
}
```

#### AI 调试响应
```json
{
  "student_id": "12345678",
  "conversation_id": "conv_1234567890",
  "current_round": 1,
  "ai_response": {
    "student_thought": "学生使用了冒泡排序..."
  },
  "dialogue_turn": {
    "round_number": 1,
    "role": "assistant",
    "content": "..."
  },
  "round_info": {
    "round_number": 1,
    "round_title": "理解学生思路",
    "round_description": "AI 将分析你的代码，理解你的解题思路",
    "can_proceed": true,
    "next_round_hint": "确认 AI 对你思路的理解是否正确",
    "is_completed": false
  }
}
```

## 路由权限

| 路由        | 需要认证 | 说明         |
| ----------- | -------- | ------------ |
| `/login`    | 否       | 登录页面     |
| `/register` | 否       | 注册页面     |
| `/profile`  | 是       | 个人主页     |
| `/ai-debug` | 是       | AI 调试页面  |
| `/history`  | 是       | 历史记录页面 |

未登录用户访问受保护路由时，将自动重定向到登录页面。

## 常见问题

### 1. 登录后跳转回登录页面
- 检查 Go 后端是否运行在 `http://localhost:8080`
- 检查浏览器控制台是否有 CORS 错误
- 确认 Token 是否正确存储

### 2. AI 调试请求失败
- 确认 Python AI 服务运行在 `http://localhost:8000`
- 检查 Go 后端日志
- 确认请求参数格式正确

### 3. 前端无法启动
- 确认 Node.js 版本 >= 16.0
- 删除 `node_modules` 和 `package-lock.json`，重新执行 `npm install`

## 开发建议

- 使用 Vue DevTools 调试组件状态
- 使用浏览器开发者工具查看网络请求
- 后端 Go 服务运行在 `:8080`
- Python AI 服务运行在 `:8000`

## 许可证

MIT License
