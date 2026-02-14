import axios from 'axios'

const api = axios.create({
	baseURL: '',
	timeout: 60000
})

// 请求拦截器 - 添加 Token
api.interceptors.request.use(
	config => {
		const token = localStorage.getItem('token')
		if (token) {
			config.headers.Authorization = `Bearer ${token}`
		}
		return config
	},
	error => {
		return Promise.reject(error)
	}
)

// 响应拦截器 - 处理错误
api.interceptors.response.use(
	response => {
		return response.data
	},
	error => {
		if (error.response) {
			const { status, data } = error.response
			if (status === 401) {
				// Token 过期或无效，清除本地存储并跳转登录
				localStorage.removeItem('token')
				localStorage.removeItem('user')
				window.location.href = '/login'
			}
			return Promise.reject(data)
		}
		return Promise.reject(error)
	}
)

// 认证 API
export const authAPI = {
	register(data) {
		return api.post('/auth/register', data)
	},
	login(data) {
		return api.post('/auth/login', data)
	},
	logout() {
		return api.post('/auth/logout')
	}
}

// 用户 API
export const userAPI = {
	getProfile() {
		return api.get('/api/v1/profile')
	}
}

// AI API
export const aiAPI = {
	// 发送调试请求
	debugV2(data) {
		return api.post('/api/v1/ai/debug_v2', data)
	},
	// 获取AI交互历史
	getRecords() {
		return api.get('/api/v1/ai/records')
	},
	// 获取轮次信息
	getRoundInfo(round, response = '') {
		return api.get('/api/v1/ai/round_info', {
			params: { round, response }
		})
	},
	// 开始新对话
	startConversation(data) {
		return api.post('/api/v1/ai/start', data)
	},
	// 代码评价
	evaluate(data) {
		return api.post('/api/v1/ai/evaluate', data)
	},
	// 题目推荐
	recommend(data) {
		return api.post('/api/v1/ai/recommend', data)
	},
	// 获取用户薄弱点
	getWeakPoints() {
		return api.get('/api/v1/ai/weak_points')
	},
	// 获取前5个薄弱点
	getTopWeakPoints() {
		return api.get('/api/v1/ai/weak_points/top')
	}
}

export default api
