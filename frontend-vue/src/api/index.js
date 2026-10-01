import axios from 'axios'
import { API_BASE_URL, AI_TIMEOUT_MS, DEFAULT_TIMEOUT_MS, LOGOUT_TIMEOUT_MS } from './config'
import { streamNdjson } from './stream'
import { getStoredToken, notifyUnauthorized } from '../utils/session'
import { ApiError, ERROR_MESSAGES, errorFromResponse } from '../utils/errors'

const api = axios.create({
	baseURL: API_BASE_URL,
	timeout: DEFAULT_TIMEOUT_MS
})

// AI 接口（非流式）使用更长的超时
const aiConfig = { timeout: AI_TIMEOUT_MS }

// 请求拦截器 - 添加 Token（过期 token 不再发送）
api.interceptors.request.use(
	config => {
		const token = getStoredToken()
		if (token) {
			config.headers.Authorization = `Bearer ${token}`
		}
		return config
	},
	error => {
		return Promise.reject(error)
	}
)

// /auth/* 接口的 401 属于业务错误（如密码错误、邮箱未验证），不应触发跳转
const isAuthEndpoint = (url = '') => /^\/auth\//.test(url.replace(API_BASE_URL, ''))

// 响应拦截器 - 处理错误
// 失败时统一 reject 一个 ApiError：error 字段为可展示的文案（服务端 error/message 或已知状态的中文提示）
api.interceptors.response.use(
	response => {
		return response.data
	},
	error => {
		const config = error.config || {}
		if (error.response) {
			const { status, data } = error.response
			if (status === 401 && !config.skipAuthRedirect && !isAuthEndpoint(config.url)) {
				// Token 过期或无效，清除登录态并跳转登录
				notifyUnauthorized()
			}
			const headers = error.response.headers || {}
			const retryAfter = typeof headers.get === 'function' ? headers.get('retry-after') : headers['retry-after']
			return Promise.reject(errorFromResponse(status, data, { retryAfter }))
		}
		if (axios.isCancel(error)) {
			return Promise.reject(new ApiError(ERROR_MESSAGES.aborted, { code: 'aborted', error: ERROR_MESSAGES.aborted }))
		}
		if (error.code === 'ECONNABORTED' || error.code === 'ETIMEDOUT') {
			return Promise.reject(new ApiError(ERROR_MESSAGES.timeout, { code: 'timeout', error: ERROR_MESSAGES.timeout }))
		}
		return Promise.reject(new ApiError(ERROR_MESSAGES.network, { code: 'network', error: ERROR_MESSAGES.network }))
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
	// 退出登录（需要 Authorization 头，由拦截器添加；失败不跳转，调用方负责清理本地状态）
	logout() {
		return api.post('/auth/logout', null, { timeout: LOGOUT_TIMEOUT_MS, skipAuthRedirect: true })
	},
	// 邮箱验证
	verifyEmail(token) {
		return api.get('/auth/verify-email', { params: { token } })
	},
	// 重新发送验证邮件
	// 后端要求请求体 { email }，用于查找待验证的注册信息（不存在时返回 404）
	resendVerificationEmail(email) {
		return api.post('/auth/resend-verification', { email })
	},
	// 忘记密码
	forgotPassword(data) {
		return api.post('/auth/forgot-password', data)
	},
	// 重置密码
	resetPassword(data) {
		return api.post('/auth/reset-password', data)
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
		return api.post('/api/v1/ai/debug_v2', data, aiConfig)
	},
	// 流式调试请求（options.signal 可用于取消）
	async debugV2Stream(data, onChunk, options = {}) {
		return streamNdjson('/api/v1/ai/debug_v2/stream', data, onChunk, options)
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
		return api.post('/api/v1/ai/start', data, aiConfig)
	},
	// 代码评价
	evaluate(data) {
		return api.post('/api/v1/ai/evaluate', data, aiConfig)
	},
	// 流式代码评价
	async evaluateStream(data, onChunk, options = {}) {
		return streamNdjson('/api/v1/ai/evaluate/stream', data, onChunk, options)
	},
	// 题目推荐
	recommend(data) {
		return api.post('/api/v1/ai/recommend', data, aiConfig)
	},
	// 流式题目推荐
	async recommendStream(data, onChunk, options = {}) {
		return streamNdjson('/api/v1/ai/recommend/stream', data, onChunk, options)
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
	},
	// ===== 班级历史记录查询 =====
	// 获取班级Debug历史记录
	getClassDebugRecords(classId, params) {
		return api.get(`/api/v1/classes/${classId}/records/debug`, { params })
	},
	// 获取班级Evaluate历史记录
	getClassEvaluateRecords(classId, params) {
		return api.get(`/api/v1/classes/${classId}/records/evaluate`, { params })
	},
	// 获取班级Recommend历史记录
	getClassRecommendRecords(classId, params) {
		return api.get(`/api/v1/classes/${classId}/records/recommend`, { params })
	},
	// 导出班级Debug历史记录
	exportClassDebugRecords(classId, params) {
		return api.get(`/api/v1/classes/${classId}/records/debug/export`, { params })
	},
	// 导出班级Evaluate历史记录
	exportClassEvaluateRecords(classId, params) {
		return api.get(`/api/v1/classes/${classId}/records/evaluate/export`, { params })
	},
	// 导出班级Recommend历史记录
	exportClassRecommendRecords(classId, params) {
		return api.get(`/api/v1/classes/${classId}/records/recommend/export`, { params })
	},
	// 获取班级薄弱点
	getClassWeakPoints(classId, params) {
		return api.get(`/api/v1/ai/weak_points/class`, { params: { class_id: classId, ...params } })
	},
	// 导出班级薄弱点CSV
	exportClassWeakPointsCSV(classId, params) {
		return api.get(`/api/v1/ai/weak_points/class/export`, {
			params: { class_id: classId, ...params },
			responseType: 'blob'
		})
	}
}

export default api
