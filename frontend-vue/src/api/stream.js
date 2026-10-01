/**
 * NDJSON 流式请求（fetch 实现，axios 不支持流式读取）
 *
 * 由于 fetch 不经过 axios 拦截器，这里需要自行处理：
 *  - Authorization 头
 *  - 401：清理登录态并跳转登录页
 *  - 429：友好提示
 *  - {"type":"error"} 事件：以 ApiError 抛出
 *  - 取消（AbortSignal）与整体超时
 *  - 跨 chunk 的半行拼接
 */
import { AI_TIMEOUT_MS, buildUrl } from './config'
import { getStoredToken, notifyUnauthorized } from '../utils/session'
import { ApiError, ERROR_MESSAGES, errorFromResponse } from '../utils/errors'

/**
 * 创建 NDJSON 行解析器
 * @param {(obj: any) => void} onObject 每解析出一个 JSON 对象时调用（其抛出的异常会向上传播）
 * @param {{ onInvalidLine?: (line: string, err: unknown) => void }} [options]
 */
export function createNdjsonParser(onObject, { onInvalidLine } = {}) {
	let buffer = ''

	const handleLine = (line) => {
		const trimmed = line.trim()
		if (!trimmed) return
		let obj
		try {
			obj = JSON.parse(trimmed)
		} catch (e) {
			if (onInvalidLine) onInvalidLine(trimmed, e)
			return
		}
		onObject(obj)
	}

	return {
		/** 追加一段文本，处理其中所有完整的行，不完整的尾部留在缓冲区 */
		push(text) {
			if (!text) return
			buffer += text
			const lines = buffer.split('\n')
			buffer = lines.pop()
			for (const line of lines) handleLine(line)
		},
		/** 流结束时处理缓冲区中剩余的最后一行（可能没有结尾换行） */
		flush() {
			const rest = buffer
			buffer = ''
			handleLine(rest)
		}
	}
}

async function readErrorBody(response) {
	try {
		const text = await response.text()
		if (!text) return null
		try {
			return JSON.parse(text)
		} catch (e) {
			return text
		}
	} catch (e) {
		return null
	}
}

/**
 * 发起 NDJSON 流式 POST 请求
 *
 * @param {string} path 接口路径，如 /api/v1/ai/evaluate/stream
 * @param {object} body 请求体（原样 JSON 序列化）
 * @param {(chunk: any) => void} onChunk 每个事件回调（error 事件除外，error 事件会以异常抛出）
 * @param {{ signal?: AbortSignal, timeoutMs?: number, fetchImpl?: typeof fetch }} [options]
 * @returns {Promise<void>}
 * @throws {ApiError} code: unauthorized | rate_limited | http | server | timeout | aborted | network
 */
export async function streamNdjson(path, body, onChunk, { signal, timeoutMs = AI_TIMEOUT_MS, fetchImpl } = {}) {
	const doFetch = fetchImpl || globalThis.fetch.bind(globalThis)
	const controller = new AbortController()
	let timedOut = false
	const timer = setTimeout(() => {
		timedOut = true
		controller.abort()
	}, timeoutMs)
	const onExternalAbort = () => controller.abort()
	if (signal) {
		if (signal.aborted) controller.abort()
		else signal.addEventListener('abort', onExternalAbort, { once: true })
	}

	// 网络层调用：把 fetch / read 的 TypeError 转为友好的网络错误
	const netCall = async (promise) => {
		try {
			return await promise
		} catch (e) {
			if (!controller.signal.aborted && e instanceof TypeError) {
				throw new ApiError(ERROR_MESSAGES.network, { code: 'network', error: ERROR_MESSAGES.network })
			}
			throw e
		}
	}

	let reader = null
	try {
		const token = getStoredToken()
		const headers = { 'Content-Type': 'application/json', Accept: 'application/x-ndjson, application/json' }
		if (token) headers.Authorization = `Bearer ${token}`

		const response = await netCall(doFetch(buildUrl(path), {
			method: 'POST',
			headers,
			body: JSON.stringify(body),
			signal: controller.signal
		}))

		if (!response.ok) {
			const data = await readErrorBody(response)
			if (response.status === 401) notifyUnauthorized()
			const retryAfter = response.headers && typeof response.headers.get === 'function'
				? response.headers.get('retry-after')
				: null
			throw errorFromResponse(response.status, data, { retryAfter })
		}

		const parser = createNdjsonParser(
			(chunk) => {
				if (chunk && chunk.type === 'error') {
					const msg = (typeof chunk.message === 'string' && chunk.message) ||
						(typeof chunk.error === 'string' && chunk.error) ||
						'AI 服务处理失败，请稍后重试'
					throw new ApiError(msg, { code: 'server', status: response.status, data: chunk, error: msg })
				}
				onChunk(chunk)
			},
			{ onInvalidLine: (line, e) => console.error('Failed to parse chunk:', line, e) }
		)

		if (!response.body || typeof response.body.getReader !== 'function') {
			parser.push(await netCall(response.text()))
			parser.flush()
			return
		}

		reader = response.body.getReader()
		const decoder = new TextDecoder()
		while (true) {
			const { done, value } = await netCall(reader.read())
			if (done) break
			parser.push(decoder.decode(value, { stream: true }))
		}
		parser.push(decoder.decode())
		parser.flush()
		reader = null
	} catch (err) {
		if (controller.signal.aborted && !(err && err.isApiError)) {
			if (timedOut) {
				throw new ApiError(ERROR_MESSAGES.timeout, { code: 'timeout', error: ERROR_MESSAGES.timeout })
			}
			throw new ApiError(ERROR_MESSAGES.aborted, { code: 'aborted', error: ERROR_MESSAGES.aborted })
		}
		throw err
	} finally {
		clearTimeout(timer)
		if (signal) signal.removeEventListener('abort', onExternalAbort)
		if (reader) {
			// 出错 / 取消时释放连接
			reader.cancel().catch(() => {})
		}
	}
}
