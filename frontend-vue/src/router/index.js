import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { resolveNavigation } from './guard'
import { toast } from '../utils/toast'

// meta 说明见 ./guard.js（requiresAuth / guestOnly / roles）

const routes = [
	{
		path: '/',
		redirect: '/login'
	},
	{
		path: '/login',
		name: 'Login',
		component: () => import('../views/Login.vue'),
		meta: { requiresAuth: false, guestOnly: true }
	},
	{
		path: '/register',
		name: 'Register',
		component: () => import('../views/Register.vue'),
		meta: { requiresAuth: false, guestOnly: true }
	},
	{
		path: '/forgot-password',
		name: 'ForgotPassword',
		component: () => import('../views/ForgotPassword.vue'),
		meta: { requiresAuth: false, guestOnly: true }
	},
	{
		path: '/reset-password',
		name: 'ResetPassword',
		component: () => import('../views/ResetPassword.vue'),
		meta: { requiresAuth: false, guestOnly: true }
	},
	{
		path: '/profile',
		name: 'Profile',
		component: () => import('../views/Profile.vue'),
		meta: { requiresAuth: true }
	},
	{
		path: '/ai-debug',
		name: 'AIDebug',
		component: () => import('../views/AIDebug.vue'),
		meta: { requiresAuth: true }
	},
	{
		path: '/history',
		name: 'History',
		component: () => import('../views/History.vue'),
		meta: { requiresAuth: true }
	},
	{
		path: '/evaluate',
		name: 'Evaluate',
		component: () => import('../views/Evaluate.vue'),
		meta: { requiresAuth: true }
	},
	{
		path: '/recommend',
		name: 'Recommend',
		component: () => import('../views/Recommend.vue'),
		meta: { requiresAuth: true }
	},
	{
		path: '/profile/classes',
		name: 'ClassManage',
		component: () => import('../views/ClassManage.vue'),
		// 学生也需要查看自己所在班级，因此不限制 user_type；
		// 班级内的教师/助教权限取决于班级成员角色，在页面内按成员信息控制，后端同样校验
		meta: { requiresAuth: true }
	}
]

const router = createRouter({
	history: createWebHistory(),
	routes
})

// 路由守卫：登录态（含 token 过期）与用户类型（meta.roles）校验
router.beforeEach((to) => {
	const authStore = useAuthStore()
	const hadToken = !!authStore.token
	const valid = authStore.validateSession()

	const result = resolveNavigation(to, {
		isAuthenticated: valid,
		sessionExpired: hadToken && !valid,
		role: authStore.role
	})

	if (result === true) return true
	if (result.denied) {
		toast.warning('没有权限访问该页面')
		return { path: result.path }
	}
	return result
})

export default router
