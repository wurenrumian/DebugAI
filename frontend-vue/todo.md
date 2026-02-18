# 班级管理系统 - 简化版需求

## 目标
为后端班级管理 API 添加基础前端页面，实现班级的创建、查询、成员管理（学生）的核心功能。

## 页面结构与路由

### 入口：个人主页 (`/profile`)
- 添加"我的班级"卡片/入口
- 点击进入"我的班级"页面

### 我的班级页面 (`/profile/classes`)
**布局**：
- 顶部导航：显示当前用户的所有班级（横向滚动或下拉选择）
- 内容区：两个 Tab
  - **Tab 1: 班级信息**（所有成员可见）
  - **Tab 2: 成员管理**（仅班级管理员 teacher/ta 可见）

**Tab 1: 班级信息**
- 显示当前选中班级的基本信息（名称、创建者、创建时间）
- 显示班级成员列表（学号、姓名、角色）
- 学生只能看到这个 tab

**Tab 2: 成员管理**
- 添加成员：批量输入学号，选择角色（teacher/ta/student）
- 移除成员：多选成员后移除
- 创建者标记：`is_creator` 的成员不可移除/降级
- 仅班级管理员（teacher/ta）可见

### 创建班级
- 在"我的班级"页面，admin 用户显示"创建班级"按钮
- 点击弹出简单表单，输入班级名称
- 创建成功后自动加入并显示在新班级中

**路由设计**：
```javascript
// 在 profile 页面内通过子路由或状态管理切换班级
{
  path: '/profile',
  component: () => import('@/views/Profile.vue'),
  meta: { requiresAuth: true },
  children: [
    {
      path: 'classes',
      // 在 Profile.vue 内嵌组件显示班级管理功能
    }
  ]
}
```

## 技术实现

### 新增 API 封装 (`src/api/index.js`)
```javascript
export const classAPI = {
  getClasses()               // GET /api/v1/classes
  getMyClasses()             // GET /api/v1/classes/my
  createClass(data)          // POST /api/v1/classes
  joinClass(classId)         // POST /api/v1/classes/:id/join
  getClassDetail(classId)    // GET /api/v1/classes/:id
  getClassMembers(classId)   // GET /api/v1/classes/:id/members
  addMembers(classId, studentIds, role)  // POST /api/v1/classes/:id/members/add
  removeMembers(classId, studentIds)     // POST /api/v1/classes/:id/members/remove
}
```

### 新增 Pinia Store (`src/stores/class.js`)
```javascript
export const useClassStore = defineStore('class', {
  state: () => ({ classes: [], currentClass: null, members: [] }),
  actions: {
    async fetchClasses()           // 获取班级列表
    async fetchMyClasses()        // 获取我的班级
    async fetchClassDetail(id)    // 获取班级详情
    async fetchMembers(classId)   // 获取成员列表
    async joinClass(classId)      // 加入班级
    async addMembers(...)         // 添加成员
    async removeMembers(...)      // 移除成员
  }
})
```

### 路由调整

**移除独立班级路由**，改为在 Profile 页面内集成：

```javascript
// src/router/index.js - 仅添加一个通用详情页（可选）
{
  path: '/profile',
  name: 'Profile',
  component: () => import('@/views/Profile.vue'),
  meta: { requiresAuth: true }
}
```

**Profile.vue 内部**：
- 显示"我的班级"区域
- 班级选择器（下拉或横向滚动）
- Tab 切换：班级信息 / 成员管理（仅管理员）
- 动态加载当前选中班级的数据

### 组件结构

```
src/views/Profile.vue          # 个人主页 + 班级管理集成
src/components/profile/
├── ClassSelector.vue         # 班级选择器（显示所有班级，支持创建）
├── ClassInfoTab.vue          # Tab 1: 班级基本信息
└── ClassManageTab.vue        # Tab 2: 成员管理（仅管理员可见）
```

### 权限控制

- **创建班级**：仅 `authStore.getUser.user_type === 'admin'` 可见按钮
- **成员管理 Tab**：仅当 `classStore.currentClass` 中当前用户角色为 `teacher` 或 `ta` 时显示
- **创建者保护**：成员列表中 `is_creator === true` 的移除按钮禁用
- **路由**：所有班级相关页面需要 `requiresAuth: true`

## 验收标准
- [ ] admin 在 Profile 页面可见"创建班级"按钮，非 admin 不可见
- [ ] 班级选择器正常显示用户加入的所有班级
- [ ] 切换班级后，Tab 内容正确更新
- [ ] Tab 1（班级信息）显示班级基本信息和成员列表
- [ ] Tab 2（成员管理）仅班级管理员可见，可添加/移除学生
- [ ] 创建者不可被移除（按钮禁用）
- [ ] 学生只能看到 Tab 1，看不到 Tab 2
- [ ] 权限控制正确（按钮显示/隐藏、功能禁用）

## 开发建议
1. 复用现有 `src/api/index.js` 的 axios 实例
2. 使用现有 `src/stores/auth.js` 的认证状态
3. 操作成功后需重新 fetch 数据刷新 UI
4. 前端权限仅为 UI 层面，后端必须验证