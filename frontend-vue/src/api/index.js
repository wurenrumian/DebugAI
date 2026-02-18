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
	// 获取AI交互历史（所有类型）
	getRecords() {
		return api.get('/api/v1/ai/records')
	},
	// 分类型获取历史记录
	getDebugRecords() {
		return api.get('/api/v1/ai/records/debug')
	},
	getEvaluateRecords() {
		return api.get('/api/v1/ai/records/evaluate')
	},
	getRecommendRecords() {
		return api.get('/api/v1/ai/records/recommend')
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
	// 获取用户薄弱点（支持日期筛选）
	getWeakPoints(params = {}) {
		return api.get('/api/v1/ai/weak_points', { params })
	},
	// 获取前N个薄弱点（支持日期筛选）
	getTopWeakPoints(params = {}) {
		return api.get('/api/v1/ai/weak_points/top', { params })
	},
	// 关闭对话
	closeConversation(conversationId) {
		return api.post('/api/v1/ai/debug/close', { conversation_id: conversationId })
	}
}

// 班级 API
export const classAPI = {
	// 获取所有班级
	getClasses() {
		return api.get('/api/v1/classes')
	},
	// 获取我的班级
	getMyClasses() {
		return api.get('/api/v1/classes/my')
	},
	// 创建班级
	createClass(data) {
		return api.post('/api/v1/classes', data)
	},
	// 加入班级
	joinClass(classId) {
		return api.post(`/api/v1/classes/${classId}/join`)
	},
	// 获取班级详情
	getClassDetail(classId) {
		return api.get(`/api/v1/classes/${classId}`)
	},
	// 获取班级成员
	getClassMembers(classId) {
		return api.get(`/api/v1/classes/${classId}/members`)
	},
	// 添加成员
	addMembers(classId, studentIds, role) {
		return api.post(`/api/v1/classes/${classId}/members/add`, { student_ids: studentIds, member_role: role })
	},
	// 移除成员
	removeMembers(classId, studentIds) {
		return api.post(`/api/v1/classes/${classId}/members/remove`, { student_ids: studentIds })
	}
}

export default api
