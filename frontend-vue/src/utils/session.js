/**
 * 会话（token）存储与 401 统一处理
 *
 * 目前 token 仍保存在 localStorage（迁移到 httpOnly Cookie 需要后端配合），
 * 但所有读取都会先检查 JWT exp，过期 token 视为未登录。
 */
import { isTokenExpired } from './jwt'

export const STORAGE_KEYS = Object.freeze({
	token: 'token',
	user: 'user',
	emailVerified: 'email_verified',
	email: 'email'
})

function safeGet(key) {
	try {
		return localStorage.getItem(key)
	} catch (e) {
		return null
	}
}

function safeRemove(key) {
	try {
		localStorage.removeItem(key)
	} catch (e) {
		// ignore
	}
}

/**
 * 读取未过期的 token；过期或不存在时返回 null
 */
export function getStoredToken() {
	const token = safeGet(STORAGE_KEYS.token)
	if (!token || isTokenExpired(token)) return null
	return token
}

/** 读取原始 token（不做过期检查） */
export function getRawStoredToken() {
	return safeGet(STORAGE_KEYS.token) || ''
}

export function clearStoredSession() {
	Object.values(STORAGE_KEYS).forEach(safeRemove)
}

/**
 * 只允许站内相对路径作为登录后的跳转目标，防止开放重定向
 * @param {unknown} target
 * @param {string} [fallback]
 */
export function safeRedirectPath(target, fallback = '/profile') {
	if (typeof target !== 'string') return fallback
	if (!target.startsWith('/') || target.startsWith('//') || target.startsWith('/\\')) return fallback
	if (target === '/login' || target.startsWith('/login?')) return fallback
	return target
}

let unauthorizedHandler = null
let handling = false

/**
 * 注册 401 / token 过期的处理函数（在 main.js 中注册，负责清理 store 并跳转登录页）
 * @param {() => (void | Promise<void>)} handler
 */
export function setUnauthorizedHandler(handler) {
	unauthorizedHandler = handler
}

/**
 * 通知登录已失效。多次并发调用只会触发一次跳转。
 */
export function notifyUnauthorized() {
	if (handling) return
	handling = true
	const done = () => {
		handling = false
	}
	try {
		if (unauthorizedHandler) {
			Promise.resolve(unauthorizedHandler()).finally(done)
			return
		}
		clearStoredSession()
		if (typeof window !== 'undefined') {
			window.location.href = '/login'
		}
	} catch (e) {
		console.error('Unauthorized handler error:', e)
	}
	done()
}
