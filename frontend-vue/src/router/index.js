import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
	{
		path: '/',
		redirect: '/login'
	},
	{
		path: '/login',
		name: 'Login',
		component: () => import('../views/Login.vue'),
		meta: { requiresAuth: false }
	},
	{
		path: '/register',
		name: 'Register',
		component: () => import('../views/Register.vue'),
		meta: { requiresAuth: false }
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
	}
]

const router = createRouter({
	history: createWebHistory(),
	routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
	const authStore = useAuthStore()

	if (to.meta.requiresAuth && !authStore.isAuthenticated) {
		next('/login')
	} else if ((to.path === '/login' || to.path === '/register') && authStore.isAuthenticated) {
		next('/profile')
	} else {
		next()
	}
})

export default router
