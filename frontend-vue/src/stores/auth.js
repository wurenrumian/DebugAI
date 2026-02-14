import { defineStore } from 'pinia'
import { authAPI, userAPI } from '../api'

export const useAuthStore = defineStore('auth', {
	state: () => ({
		token: localStorage.getItem('token') || '',
		user: JSON.parse(localStorage.getItem('user') || '{}'),
		isAuthenticated: !!localStorage.getItem('token')
	}),

	getters: {
		getToken: (state) => state.token,
		getUser: (state) => state.user,
		isLoggedIn: (state) => state.isAuthenticated
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

					return { success: true, data: response.data }
				}
				return { success: false, error: '登录失败' }
			} catch (error) {
				return { success: false, error: error.error || '登录失败' }
			}
		},

		async register(userData) {
			try {
				await authAPI.register(userData)
				return { success: true }
			} catch (error) {
				return { success: false, error: error.error || '注册失败' }
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
					localStorage.setItem('user', JSON.stringify(this.user))
					return { success: true, data: response.data }
				}
				return { success: false, error: '获取用户信息失败' }
			} catch (error) {
				return { success: false, error: error.error || '获取用户信息失败' }
			}
		},

		clearAuth() {
			this.token = ''
			this.user = {}
			this.isAuthenticated = false
			localStorage.removeItem('token')
			localStorage.removeItem('user')
		}
	}
})
