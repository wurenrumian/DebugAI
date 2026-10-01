/**
 * 统一的接口错误类型与错误文案提取
 */

export const ERROR_MESSAGES = Object.freeze({
	network: '网络连接失败，请检查网络后重试',
	timeout: '请求超时，请稍后重试',
	rateLimited: '请求过于频繁，请稍后再试',
	unauthorized: '登录已过期，请重新登录',
	aborted: '请求已取消',
	generic: '操作失败，请稍后重试'
})

/**
 * 接口错误。
 *  - error：可直接展示给用户的文案（服务端返回的 error/message 或已知状态的中文提示），
 *    未知情况下为 undefined，调用方可以使用自己的兜底文案
 *  - status：HTTP 状态码（网络错误 / 超时为 0）
 *  - code：'http' | 'network' | 'timeout' | 'aborted' | 'unauthorized' | 'rate_limited' | 'server'
 */
export class ApiError extends Error {
	constructor(message, { status = 0, code = 'http', data = null, error } = {}) {
		super(message || error || ERROR_MESSAGES.generic)
		this.name = 'ApiError'
		this.isApiError = true
		if (data && typeof data === 'object' && !(typeof Blob !== 'undefined' && data instanceof Blob)) {
			// 保留服务端返回的其他字段，兼容原来直接读取 data 字段的代码
			for (const [key, value] of Object.entries(data)) {
				if (!(key in this)) this[key] = value
			}
		}
		this.status = status
		this.code = code
		this.data = data
		this.error = error
	}
}

/** 从服务端响应体中提取错误文案 */
export function extractServerMessage(data) {
	if (!data) return undefined
	if (typeof data === 'string') {
		const text = data.trim()
		// 避免把 nginx 等返回的 HTML 错误页直接展示给用户
		return text && text.length < 300 && !text.startsWith('<') ? text : undefined
	}
	if (typeof data === 'object') {
		for (const key of ['error', 'message', 'msg']) {
			if (typeof data[key] === 'string' && data[key].trim()) return data[key].trim()
		}
	}
	return undefined
}

/**
 * 解析限流后的等待秒数：优先使用响应体 retry_after，其次是 Retry-After 头（秒数或 HTTP 日期）
 * @returns {number | null}
 */
export function parseRetryAfter(data, retryAfterHeader, now = Date.now()) {
	const fromBody = data && typeof data === 'object' ? Number(data.retry_after) : NaN
	if (Number.isFinite(fromBody) && fromBody > 0) return Math.ceil(fromBody)
	if (retryAfterHeader === undefined || retryAfterHeader === null || retryAfterHeader === '') return null
	const seconds = Number(retryAfterHeader)
	if (Number.isFinite(seconds)) return seconds > 0 ? Math.ceil(seconds) : null
	const date = Date.parse(retryAfterHeader)
	if (Number.isNaN(date)) return null
	const diff = Math.ceil((date - now) / 1000)
	return diff > 0 ? diff : null
}

/**
 * 根据 HTTP 状态和响应体构造 ApiError
 * @param {number} status
 * @param {unknown} data 响应体
 * @param {{ retryAfter?: string | null }} [headers] 相关响应头
 */
export function errorFromResponse(status, data, { retryAfter } = {}) {
	if (status === 429) {
		const seconds = parseRetryAfter(data, retryAfter)
		const msg = seconds ? `请求过于频繁，请 ${seconds} 秒后再试` : ERROR_MESSAGES.rateLimited
		const err = new ApiError(msg, { status, code: 'rate_limited', data, error: msg })
		err.retryAfter = seconds
		return err
	}
	const serverMessage = extractServerMessage(data)
	if (status === 401) {
		const msg = serverMessage || ERROR_MESSAGES.unauthorized
		return new ApiError(msg, { status, code: 'unauthorized', data, error: msg })
	}
	return new ApiError(serverMessage || `请求失败（HTTP ${status}）`, { status, code: 'http', data, error: serverMessage })
}

export function isAbortError(err) {
	return !!err && (err.code === 'aborted' || err.name === 'AbortError')
}

/**
 * 获取可展示给用户的错误文案
 * @param {unknown} err
 * @param {string} [fallback]
 */
export function getErrorMessage(err, fallback = ERROR_MESSAGES.generic) {
	if (!err) return fallback
	if (typeof err === 'string') return err
	if (typeof err.error === 'string' && err.error.trim()) return err.error
	return fallback
}
