import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth'
import { setUnauthorizedHandler, safeRedirectPath } from './utils/session'
import { initTheme } from './utils/theme'
import './styles/tokens.css'
import './styles/base.css'
import './assets/markdown.css'

// 主题偏好（浅色 / 深色 / 跟随系统）
initTheme()

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)

// 登录失效（401 / token 过期）统一处理：清理登录态并跳转登录页，保留原目标路由
const authStore = useAuthStore(pinia)
setUnauthorizedHandler(async () => {
	authStore.clearAuth()
	const current = router.currentRoute.value
	// 当前页面本身不需要登录（如登录 / 注册页），只清理状态即可
	if (!current.meta || !current.meta.requiresAuth) return
	const query = { expired: '1' }
	const redirect = safeRedirectPath(current.fullPath, '')
	if (redirect) query.redirect = redirect
	await router.replace({ path: '/login', query }).catch(() => {})
})
authStore.scheduleExpiryCheck()

app.use(router)
app.mount('#app')
