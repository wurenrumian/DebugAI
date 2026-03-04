import { defineStore } from 'pinia'
import { authAPI, userAPI } from '../api'

export const useAuthStore = defineStore('auth', {
	state: () => ({
		token: localStorage.getItem('token') || '',
		user: JSON.parse(localStorage.getItem('user') || '{}'),
		isAuthenticated: !!localStorage.getItem('token'),
		emailVerified: localStorage.getItem('email_verified') === 'true',
		email: localStorage.getItem('email') || ''
	}),

	getters: {
		getToken: (state) => state.token,
		getUser: (state) => state.user,
		isLoggedIn: (state) => state.isAuthenticated,
		isEmailVerified: (state) => state.emailVerified,
		getEmail: (state) => state.email
	},

	actions: {
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
					this.isAuthenticated = true

					// 存储到 localStorage
					localStorage.setItem('token', this.token)
					localStorage.setItem('user', JSON.stringify(this.user))

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

		async logout() {
			try {
				await authAPI.logout()
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

		async resendVerificationEmail() {
			try {
				await authAPI.resendVerificationEmail()
				return { success: true }
			} catch (error) {
				return { success: false, error: error.error || '发送失败' }
			}
		},

		clearAuth() {
			this.token = ''
			this.user = {}
			this.isAuthenticated = false
			this.emailVerified = false
			this.email = ''
			localStorage.removeItem('token')
			localStorage.removeItem('user')
			localStorage.removeItem('email_verified')
			localStorage.removeItem('email')
		}
	}
})
