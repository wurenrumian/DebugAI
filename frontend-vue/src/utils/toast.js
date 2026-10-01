/**
 * 轻量全局消息提示（替代 alert），由 App.vue 中的 <ToastContainer /> 渲染
 *
 * 用法：
 *   import { toast } from '@/utils/toast'
 *   toast.success('保存成功')
 *   toast.error(getErrorMessage(err, '保存失败'))
 */
import { reactive } from 'vue'

export const toastState = reactive({ items: [] })

const MAX_TOASTS = 4
let seq = 0
const timers = new Map()

export function dismissToast(id) {
	const index = toastState.items.findIndex(item => item.id === id)
	if (index !== -1) toastState.items.splice(index, 1)
	if (timers.has(id)) {
		clearTimeout(timers.get(id))
		timers.delete(id)
	}
}

/**
 * @param {string} message
 * @param {{ type?: 'success'|'error'|'warning'|'info', duration?: number }} [options]
 * @returns {number} toast id
 */
export function showToast(message, { type = 'info', duration } = {}) {
	const text = String(message ?? '').trim()
	if (!text) return -1
	const id = ++seq
	toastState.items.push({ id, type, message: text })
	while (toastState.items.length > MAX_TOASTS) {
		dismissToast(toastState.items[0].id)
	}
	const ms = duration ?? (type === 'error' ? 5000 : 3000)
	if (ms > 0) {
		timers.set(id, setTimeout(() => dismissToast(id), ms))
	}
	return id
}

export const toast = {
	success: (message, options) => showToast(message, { ...options, type: 'success' }),
	error: (message, options) => showToast(message, { ...options, type: 'error' }),
	warning: (message, options) => showToast(message, { ...options, type: 'warning' }),
	info: (message, options) => showToast(message, { ...options, type: 'info' })
}
