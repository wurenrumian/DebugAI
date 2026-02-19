import { defineStore } from 'pinia'
import { classAPI } from '../api'

// 辅助函数：触发 JSON 文件下载
function downloadJSON(data, filename) {
	const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
	const url = URL.createObjectURL(blob)
	const link = document.createElement('a')
	link.href = url
	link.download = filename
	document.body.appendChild(link)
	link.click()
	document.body.removeChild(link)
	URL.revokeObjectURL(url)
}

export const useClassStore = defineStore('class', {
	state: () => ({
		classes: [],
		currentClass: null,
		members: [],
		loading: false,
		error: null,
		// 班级历史记录
		classDebugRecords: { total: 0, data: [] },
		classEvaluateRecords: { total: 0, data: [] },
		classRecommendRecords: { total: 0, data: [] },
		// 班级薄弱点
		classWeakPoints: []
	}),

	getters: {
		getClasses: (state) => state.classes,
		getCurrentClass: (state) => state.currentClass,
		getMembers: (state) => state.members,
		isLoading: (state) => state.loading,
		getError: (state) => state.error,
		// 获取当前用户在当前班级中的角色
		getCurrentUserRole: (state) => {
			const user = JSON.parse(localStorage.getItem('user') || '{}')
			if (!state.currentClass || !user.student_id) return null
			const member = state.members.find(m => m.student_id === user.student_id)
			return member ? member.role : null
		},
		// 检查当前用户是否是班级管理员 (teacher 或 ta)
		isClassAdmin: (state) => {
			const role = state.members.find(m => {
				const user = JSON.parse(localStorage.getItem('user') || '{}')
				return m.student_id === user.student_id
			})?.role
			return role === 'teacher' || role === 'ta'
		}
	},

	actions: {
		// 获取所有班级
		async fetchClasses() {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getClasses()
				this.classes = response.data || []
				return { success: true, data: this.classes }
			} catch (error) {
				this.error = error.error || '获取班级列表失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 获取我的班级
		async fetchMyClasses() {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getMyClasses()
				this.classes = response.data || []
				return { success: true, data: this.classes }
			} catch (error) {
				this.error = error.error || '获取我的班级失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 创建班级
		async createClass(data) {
			this.loading = true
			this.error = null
			try {
				// 后端期望 class_name 字段
				const payload = { class_name: data.name || data.class_name }
				const response = await classAPI.createClass(payload)
				const newClass = response.data
				if (newClass) {
					this.classes.push(newClass)
					this.currentClass = newClass
				}
				return { success: true, data: newClass }
			} catch (error) {
				this.error = error.error || '创建班级失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 加入班级
		async joinClass(classId) {
			this.loading = true
			this.error = null
			try {
				await classAPI.joinClass(classId)
				// 重新获取我的班级
				await this.fetchMyClasses()
				// 设置当前班级
				const joinedClass = this.classes.find(c => c.id === classId)
				if (joinedClass) {
					this.currentClass = joinedClass
				}
				return { success: true }
			} catch (error) {
				this.error = error.error || '加入班级失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 获取班级详情
		async fetchClassDetail(classId) {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getClassDetail(classId)
				this.currentClass = response.data
				return { success: true, data: this.currentClass }
			} catch (error) {
				this.error = error.error || '获取班级详情失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 获取班级成员
		async fetchMembers(classId) {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getClassMembers(classId)
				this.members = response.data || []
				return { success: true, data: this.members }
			} catch (error) {
				this.error = error.error || '获取成员列表失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 添加成员
		async addMembers(classId, studentIds, role) {
			this.loading = true
			this.error = null
			try {
				await classAPI.addMembers(classId, studentIds, role)
				// 重新获取成员列表
				await this.fetchMembers(classId)
				return { success: true }
			} catch (error) {
				this.error = error.error || '添加成员失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 移除成员
		async removeMembers(classId, studentIds) {
			this.loading = true
			this.error = null
			try {
				await classAPI.removeMembers(classId, studentIds)
				// 重新获取成员列表
				await this.fetchMembers(classId)
				return { success: true }
			} catch (error) {
				this.error = error.error || '移除成员失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 设置当前班级
		setCurrentClass(classId) {
			const selectedClass = this.classes.find(c => c.id === classId)
			if (selectedClass) {
				this.currentClass = selectedClass
			}
		},

		// 清空状态
		clearState() {
			this.classes = []
			this.currentClass = null
			this.members = []
			this.loading = false
			this.error = null
			this.classDebugRecords = { total: 0, data: [] }
			this.classEvaluateRecords = { total: 0, data: [] }
			this.classRecommendRecords = { total: 0, data: [] }
			this.classWeakPoints = []
		},

		// ===== 班级历史记录查询 =====
		// 查询班级Debug历史记录
		async fetchClassDebugRecords(classId, filters = {}) {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getClassDebugRecords(classId, filters)
				this.classDebugRecords = response.data || { total: 0, data: [] }
				return { success: true, data: this.classDebugRecords }
			} catch (error) {
				this.error = error.error || '查询Debug历史失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 查询班级Evaluate历史记录
		async fetchClassEvaluateRecords(classId, filters = {}) {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getClassEvaluateRecords(classId, filters)
				this.classEvaluateRecords = response.data || { total: 0, data: [] }
				return { success: true, data: this.classEvaluateRecords }
			} catch (error) {
				this.error = error.error || '查询Evaluate历史失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 查询班级Recommend历史记录
		async fetchClassRecommendRecords(classId, filters = {}) {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getClassRecommendRecords(classId, filters)
				this.classRecommendRecords = response.data || { total: 0, data: [] }
				return { success: true, data: this.classRecommendRecords }
			} catch (error) {
				this.error = error.error || '查询Recommend历史失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		},

		// 导出班级Debug历史记录
		async exportClassDebugRecords(classId, filters = {}) {
			try {
				const response = await classAPI.exportClassDebugRecords(classId, filters)
				downloadJSON(response, `debug_history_${classId}.json`)
				return { success: true }
			} catch (error) {
				this.error = error.error || '导出Debug历史失败'
				return { success: false, error: this.error }
			}
		},

		// 导出班级Evaluate历史记录
		async exportClassEvaluateRecords(classId, filters = {}) {
			try {
				const response = await classAPI.exportClassEvaluateRecords(classId, filters)
				downloadJSON(response, `evaluate_history_${classId}.json`)
				return { success: true }
			} catch (error) {
				this.error = error.error || '导出Evaluate历史失败'
				return { success: false, error: this.error }
			}
		},

		// 导出班级Recommend历史记录
		async exportClassRecommendRecords(classId, filters = {}) {
			try {
				const response = await classAPI.exportClassRecommendRecords(classId, filters)
				downloadJSON(response, `recommend_history_${classId}.json`)
				return { success: true }
			} catch (error) {
				this.error = error.error || '导出Recommend历史失败'
				return { success: false, error: this.error }
			}
		},

		// 查询班级薄弱点
		async fetchClassWeakPoints(classId, filters = {}) {
			this.loading = true
			this.error = null
			try {
				const response = await classAPI.getClassWeakPoints(classId, filters)
				this.classWeakPoints = response.data || []
				return { success: true, data: this.classWeakPoints }
			} catch (error) {
				this.error = error.error || '查询班级薄弱点失败'
				return { success: false, error: this.error }
			} finally {
				this.loading = false
			}
		}
	}
})
