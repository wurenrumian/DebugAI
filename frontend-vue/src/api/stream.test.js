import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createNdjsonParser, streamNdjson } from './stream'
import { setUnauthorizedHandler } from '../utils/session'

describe('createNdjsonParser', () => {
	it('处理跨 chunk 的半行', () => {
		const out = []
		const parser = createNdjsonParser(obj => out.push(obj))
		parser.push('{"type":"te')
		expect(out).toEqual([])
		parser.push('xt","content":"a"}\n{"type":"text",')
		expect(out).toEqual([{ type: 'text', content: 'a' }])
		parser.push('"content":"b"}\n')
		expect(out).toEqual([{ type: 'text', content: 'a' }, { type: 'text', content: 'b' }])
	})

	it('一个 chunk 中包含多行、空行与 CRLF', () => {
		const out = []
		const parser = createNdjsonParser(obj => out.push(obj))
		parser.push('{"a":1}\r\n\n  \n{"a":2}\n{"a":3}\n')
		expect(out).toEqual([{ a: 1 }, { a: 2 }, { a: 3 }])
	})

	it('flush 处理没有结尾换行的最后一行', () => {
		const out = []
		const parser = createNdjsonParser(obj => out.push(obj))
		parser.push('{"type":"text","content":"x"}\n{"type":"done"}')
		expect(out).toHaveLength(1)
		parser.flush()
		expect(out).toEqual([{ type: 'text', content: 'x' }, { type: 'done' }])
		parser.flush()
		expect(out).toHaveLength(2)
	})

	it('非法 JSON 行被跳过并上报，不影响后续行', () => {
		const out = []
		const bad = []
		const parser = createNdjsonParser(obj => out.push(obj), { onInvalidLine: line => bad.push(line) })
		parser.push('not json\n{"ok":true}\n')
		expect(bad).toEqual(['not json'])
		expect(out).toEqual([{ ok: true }])
	})

	it('内容中包含转义换行时不会被错误切分', () => {
		const out = []
		const parser = createNdjsonParser(obj => out.push(obj))
		parser.push('{"content":"#include <stdio.h>\\nint main()"}\n')
		expect(out[0].content).toBe('#include <stdio.h>\nint main()')
	})

	it('回调抛出的异常向上传播', () => {
		const parser = createNdjsonParser(() => { throw new Error('boom') })
		expect(() => parser.push('{"a":1}\n')).toThrow('boom')
	})
})

function ndjsonResponse(chunks, init = {}) {
	const encoder = new TextEncoder()
	const stream = new ReadableStream({
		start(controller) {
			for (const c of chunks) controller.enqueue(encoder.encode(c))
			controller.close()
		}
	})
	return new Response(stream, { status: 200, ...init })
}

describe('streamNdjson', () => {
	beforeEach(() => {
		setUnauthorizedHandler(null)
	})

	it('按顺序回调事件并跨 chunk 拼接（含多字节字符拆分）', async () => {
		const bytes = new TextEncoder().encode('{"type":"text","content":"你好"}\n{"type":"done"}')
		const stream = new ReadableStream({
			start(controller) {
				// 在中文字符的中间切开
				controller.enqueue(bytes.slice(0, 30))
				controller.enqueue(bytes.slice(30))
				controller.close()
			}
		})
		const fetchImpl = vi.fn().mockResolvedValue(new Response(stream, { status: 200 }))
		const chunks = []
		await streamNdjson('/api/test', { a: 1 }, c => chunks.push(c), { fetchImpl })
		expect(chunks).toEqual([{ type: 'text', content: '你好' }, { type: 'done' }])
		const [, options] = fetchImpl.mock.calls[0]
		expect(options.method).toBe('POST')
		expect(options.body).toBe(JSON.stringify({ a: 1 }))
	})

	it('error 事件以异常抛出并携带服务端消息', async () => {
		const fetchImpl = vi.fn().mockResolvedValue(ndjsonResponse([
			'{"type":"text","content":"a"}\n{"type":"error","message":"模型调用失败"}\n{"type":"done"}\n'
		]))
		const chunks = []
		await expect(streamNdjson('/x', {}, c => chunks.push(c), { fetchImpl }))
			.rejects.toMatchObject({ code: 'server', error: '模型调用失败' })
		expect(chunks).toEqual([{ type: 'text', content: 'a' }])
	})

	it('429 返回友好提示', async () => {
		const fetchImpl = vi.fn().mockResolvedValue(new Response('{"error":"Rate limit exceeded"}', { status: 429 }))
		await expect(streamNdjson('/x', {}, () => {}, { fetchImpl }))
			.rejects.toMatchObject({ status: 429, error: '请求过于频繁，请稍后再试' })
	})

	it('401 触发登录失效处理', async () => {
		const handler = vi.fn()
		setUnauthorizedHandler(handler)
		const fetchImpl = vi.fn().mockResolvedValue(new Response('{"error":"token expired"}', { status: 401 }))
		await expect(streamNdjson('/x', {}, () => {}, { fetchImpl }))
			.rejects.toMatchObject({ status: 401, code: 'unauthorized' })
		expect(handler).toHaveBeenCalledTimes(1)
	})

	it('其他 HTTP 错误使用服务端 error 字段', async () => {
		const fetchImpl = vi.fn().mockResolvedValue(new Response('{"error":"无权访问"}', { status: 403 }))
		await expect(streamNdjson('/x', {}, () => {}, { fetchImpl }))
			.rejects.toMatchObject({ status: 403, error: '无权访问' })
	})

	it('外部 AbortSignal 取消请求', async () => {
		const controller = new AbortController()
		const fetchImpl = vi.fn((url, { signal }) => new Promise((resolve, reject) => {
			signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
		}))
		const p = streamNdjson('/x', {}, () => {}, { fetchImpl, signal: controller.signal })
		controller.abort()
		await expect(p).rejects.toMatchObject({ code: 'aborted' })
	})

	it('整体超时', async () => {
		const fetchImpl = vi.fn((url, { signal }) => new Promise((resolve, reject) => {
			signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
		}))
		await expect(streamNdjson('/x', {}, () => {}, { fetchImpl, timeoutMs: 10 }))
			.rejects.toMatchObject({ code: 'timeout', error: '请求超时，请稍后重试' })
	})

	it('网络错误转换为友好提示', async () => {
		const fetchImpl = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
		await expect(streamNdjson('/x', {}, () => {}, { fetchImpl }))
			.rejects.toMatchObject({ code: 'network' })
	})
})
