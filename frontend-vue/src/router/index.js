import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Profile from '../views/Profile.vue'
import AIDebugCore from '../views/AIDebugCore.vue'
import ConversationHistory from '../views/ConversationHistory.vue'

const routes = [
	{
		path: '/',
		redirect: '/login'
	},
	{
		path: '/login',
		name: 'Login',
		component: Login
	},
	{
		path: '/register',
		name: 'Register',
		component: Register
	},
	{
		path: '/profile',
		name: 'Profile',
		component: Profile,
		meta: { requiresAuth: true }
	},
	{
		path: '/aidebug',
		name: 'AIDebugCore',
		component: AIDebugCore,
		meta: { requiresAuth: true }
	},
	{
		path: '/history',
		name: 'ConversationHistory',
		component: ConversationHistory,
		meta: { requiresAuth: true }
	},
	{
    	path: '/aidebugv2',
    	name: 'AIDebugV2',
    	component: () => import('../views/AIDebugV2.vue'),
    	meta: { requiresAuth: true }
	},
	{
    	path: '/debugv2-test',
    	name: 'DebugV2Test',
    	component: () => import('../views/DebugV2Test.vue'),
    	meta: { requiresAuth: false }  // 测试页面不需要登录
	}
]

const router = createRouter({
	history: createWebHistory(import.meta.env.BASE_URL),
	routes
})

router.beforeEach((to, from, next) => {
	const user = localStorage.getItem('user')
	if (to.meta.requiresAuth && !user) {
		next('/login')
	} else {
		next()
	}
})

export default router
