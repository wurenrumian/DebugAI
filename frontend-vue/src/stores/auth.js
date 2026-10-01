import { defineStore } from 'pinia'
import { authAPI, userAPI } from '../api'
import { decodeJwtPayload, getTokenExpiry, isTokenExpired } from '../utils/jwt'
import { clearStoredSession, getRawStoredToken, notifyUnauthorized } from '../utils/session'

// 用户类型（与后端 models.TypeUser / models.TypeAdmin 保持一致）
export const ROLES = Object.freeze({
	USER: 'user',
	ADMIN: 'admin'
})

// setTimeout 的最大延迟（约 24.8 天），超出会立即触发
const MAX_TIMER_DELAY = 2147483647
let expiryTimer = null

function readJSON(key, fallback) {
	try {
		const raw = localStorage.getItem(key)
		return raw ? JSON.parse(raw) : fallback
	} catch (e) {
		return fallback
	}
}

function readItem(key) {
	try {
		return localStorage.getItem(key)
	} catch (e) {
		return null
	}
}

// 初始化时丢弃已过期的 token
function loadInitialState() {
	const token = getRawStoredToken()
	if (!token || isTokenExpired(token)) {
		if (token) clearStoredSession()
		return { token: '', user: {}, emailVerified: false, email: readItem('email') || '' }
	}
	const user = readJSON('user', {})
	return {
		token,
		user: user && typeof user === 'object' ? user : {},
		emailVerified: readItem('email_verified') === 'true',
		email: readItem('email') || ''
	}
}

export const useAuthStore = defineStore('auth', {
	state: () => loadInitialState(),

	getters: {
		getToken: (state) => state.token,
		getUser: (state) => state.user,
		// 有 token 且未过期才视为已登录（过期定时器 / 路由守卫会及时清理过期 token）
		isAuthenticated: (state) => !!state.token && !isTokenExpired(state.token),
		isLoggedIn() {
			return this.isAuthenticated
		},
		isEmailVerified: (state) => state.emailVerified,
		getEmail: (state) => state.email,
		// 当前用户类型：优先使用登录/资料接口返回的 user_type，其次是 JWT 声明
		role: (state) => {
			if (!state.token) return ''
			const fromUser = state.user && state.user.user_type
			if (fromUser) return fromUser
			const claims = decodeJwtPayload(state.token)
			return (claims && claims.user_type) || ROLES.USER
		},
		isAdmin() {
			return this.isAuthenticated && this.role === ROLES.ADMIN
		}
	},

	actions: {
		/**
		 * 按当前时间校验登录态：token 过期则清理并返回 false
		 */
		validateSession() {
			if (!this.token) return false
			if (isTokenExpired(this.token)) {
				this.clearAuth()
				return false
			}
			return true
		},

		/**
		 * 在 token 过期时自动登出（清理状态并跳转登录页）
		 */
		scheduleExpiryCheck() {
			if (expiryTimer) {
				clearTimeout(expiryTimer)
				expiryTimer = null
			}
			if (!this.token) return
			const expiry = getTokenExpiry(this.token)
			if (expiry === null) return
			const delay = Math.min(Math.max(expiry - Date.now(), 0), MAX_TIMER_DELAY)
			expiryTimer = setTimeout(() => {
				expiryTimer = null
				if (this.token && isTokenExpired(this.token)) {
					notifyUnauthorized()
				} else {
					this.scheduleExpiryCheck()
				}
			}, delay)
		},

		async login(credentials) {
			try {
				const response = await authAPI.login(credentials)
				if (response.data && response.data.token) {
					this.token = response.data.token
					this.user = {
						username: response.data.username,
						user_type: response.data.user_type,
						student_id: response.data.student_id
					}

					// 存储到 localStorage
					localStorage.setItem('token', this.token)
					localStorage.setItem('user', JSON.stringify(this.user))
					this.scheduleExpiryCheck()

					// 检查是否有邮箱验证状态
					if (response.data.email_verified !== undefined) {
						this.emailVerified = response.data.email_verified
						localStorage.setItem('email_verified', this.emailVerified)
					}
					if (response.data.email) {
						this.email = response.data.email
						localStorage.setItem('email', this.email)
					}

					return { success: true, data: response.data }
				}
				return { success: false, error: '登录失败' }
			} catch (error) {
				if (error && error.status === 429) {
					// 登录被限流：后端返回 Retry-After 头和 retry_after（秒）
					const seconds = error.retryAfter
					return {
						success: false,
						error: seconds ? `尝试次数过多，请 ${seconds} 秒后再试` : '尝试次数过多，请稍后再试',
						retryAfter: seconds || null
					}
				}
				return { success: false, error: error.error || '登录失败' }
			}
		},

		async register(userData) {
			try {
				const response = await authAPI.register(userData)
				// 注册第一步：发送验证邮件成功
				if (response.data) {
					// 保存邮箱到本地，用于后续重发验证邮件
					if (userData.email) {
						this.email = userData.email
						localStorage.setItem('email', this.email)
					}
					return { success: true, data: response.data }
				}
				return { success: false, error: '发送失败' }
			} catch (error) {
				return { success: false, error: error.error || '发送失败' }
			}
		},

		// 无论后端退出接口成功与否，都会清理本地登录态
		async logout() {
			try {
				if (this.token && !isTokenExpired(this.token)) {
					await authAPI.logout()
				}
			} catch (error) {
				console.error('Logout error:', error)
			} finally {
				this.clearAuth()
			}
		},

		async fetchProfile() {
			try {
				const response = await userAPI.getProfile()
				if (response.data) {
					this.user = {
						username: response.data.username,
						user_type: response.data.user_type,
						student_id: response.data.student_id
					}
					// 更新邮箱验证状态
					if (response.data.email_verified !== undefined) {
						this.emailVerified = response.data.email_verified
						localStorage.setItem('email_verified', this.emailVerified)
					}
					if (response.data.email) {
						this.email = response.data.email
						localStorage.setItem('email', this.email)
					}
					localStorage.setItem('user', JSON.stringify(this.user))
					return { success: true, data: response.data }
				}
				return { success: false, error: '获取用户信息失败' }
			} catch (error) {
				return { success: false, error: error.error || '获取用户信息失败' }
			}
		},

		async verifyEmail(token) {
			try {
				const response = await authAPI.verifyEmail(token)
				if (response.data) {
					this.emailVerified = true
					localStorage.setItem('email_verified', 'true')
					return { success: true, data: response.data }
				}
				return { success: false, error: '验证失败' }
			} catch (error) {
				return { success: false, error: error.error || '验证失败' }
			}
		},

		/**
		 * 重新发送注册验证邮件
		 * @param {string | { email?: string }} [payload] 邮箱，缺省时使用注册时保存的邮箱
		 */
		async resendVerificationEmail(payload) {
			const email = String(
				(payload && typeof payload === 'object' ? payload.email : payload) || this.email || ''
			).trim()
			if (!email) {
				return { success: false, error: '缺少邮箱地址，请重新填写注册信息' }
			}
			try {
				await authAPI.resendVerificationEmail(email)
				return { success: true }
			} catch (error) {
				if (error && error.status === 404) {
					return { success: false, error: error.error || '未找到该邮箱的待验证注册信息，请重新注册' }
				}
				return { success: false, error: error.error || '发送失败' }
			}
		},

		async forgotPassword(data) {
			try {
				const response = await authAPI.forgotPassword(data)
				return { success: true, data: response.data }
			} catch (error) {
				return { success: false, error: error.error || '请求失败' }
			}
		},

		async resetPassword(data) {
			try {
				const response = await authAPI.resetPassword(data)
				return { success: true, data: response.data }
			} catch (error) {
				return { success: false, error: error.error || '重置失败' }
			}
		},

		clearAuth() {
			if (expiryTimer) {
				clearTimeout(expiryTimer)
				expiryTimer = null
			}
			this.token = ''
			this.user = {}
			this.emailVerified = false
			this.email = ''
			clearStoredSession()
		}
	}
})
