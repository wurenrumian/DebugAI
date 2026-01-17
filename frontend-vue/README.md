# Frontend Vue - AI 教学辅助平台

## 项目描述
这是一个基于 Vue 3 的前端应用，与后端 Go 服务配合，实现用户登录、注册和个人主页功能。用户可以通过学号和密码注册/登录，登录成功后显示个人基本信息（学号、用户名、用户类型）。

## 依赖安装
确保 Node.js 已安装，然后运行：
```
npm install
```
已安装的依赖包括：
- Vue 3
- Vue Router 4
- Axios (用于 API 请求)

## 运行开发服务器
```
npm run dev
```
应用将在 http://localhost:5173 启动。打开浏览器访问此地址。

## 与后端集成
- 后端服务需运行在 http://localhost:8080
- 注册 API: POST /auth/register
- 登录 API: POST /auth/login
- 个人资料 API: GET /api/v1/profile (受 JWT 保护)

## 功能
- **注册页面** (/register): 输入学号、用户名、密码、用户类型（学生/管理员）
- **登录页面** (/login): 输入学号和密码，成功后跳转到个人主页
- **个人主页** (/profile): 显示用户基本信息，支持退出登录
- 导航守卫确保未登录用户无法访问个人主页

## 项目结构
- `src/router/index.js`: 路由配置
- `src/views/Login.vue`: 登录组件
- `src/views/Register.vue`: 注册组件
- `src/views/Profile.vue`: 个人主页组件
- `src/App.vue`: 根组件，包含导航栏
- `src/main.js`: 应用入口，集成路由

## 注意事项
- 无验证码验证
- Token 存储在 localStorage 中
- 确保后端服务已启动，否则 API 请求将失败

