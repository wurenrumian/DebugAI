import { describe, it, expect } from 'vitest'
import { decodeJwtPayload, getTokenExpiry, isTokenExpired } from './jwt'

function base64url(obj) {
	const bytes = new TextEncoder().encode(JSON.stringify(obj))
	let binary = ''
	bytes.forEach(b => { binary += String.fromCharCode(b) })
	return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function makeToken(payload) {
	return `${base64url({ alg: 'HS256', typ: 'JWT' })}.${base64url(payload)}.signature`
}

const NOW = 1_700_000_000_000

describe('decodeJwtPayload', () => {
	it('解码 base64url payload（含中文与 -_ 字符）', () => {
		const payload = { id: 1, student_id: '2023001', user_type: 'admin', name: '张三??>>' }
		expect(decodeJwtPayload(makeToken(payload))).toEqual(payload)
	})

	it('非法输入返回 null 而不抛异常', () => {
		expect(decodeJwtPayload('')).toBeNull()
		expect(decodeJwtPayload(null)).toBeNull()
		expect(decodeJwtPayload(123)).toBeNull()
		expect(decodeJwtPayload('abc')).toBeNull()
		expect(decodeJwtPayload('a.b')).toBeNull()
		expect(decodeJwtPayload('a.!!!.c')).toBeNull()
		expect(decodeJwtPayload(`a.${btoa('not json')}.c`)).toBeNull()
		expect(decodeJwtPayload(`a.${base64url([1, 2])}.c`)).toBeNull()
	})
})

describe('getTokenExpiry / isTokenExpired', () => {
	it('读取 exp（秒）并转换为毫秒', () => {
		expect(getTokenExpiry(makeToken({ exp: 1_700_000_100 }))).toBe(1_700_000_100_000)
		expect(getTokenExpiry(makeToken({ sub: 'x' }))).toBeNull()
	})

	it('未过期 token', () => {
		const token = makeToken({ exp: NOW / 1000 + 3600 })
		expect(isTokenExpired(token, NOW)).toBe(false)
	})

	it('已过期 token', () => {
		const token = makeToken({ exp: NOW / 1000 - 1 })
		expect(isTokenExpired(token, NOW)).toBe(true)
	})

	it('在宽限期内即将过期视为过期', () => {
		const token = makeToken({ exp: NOW / 1000 + 2 })
		expect(isTokenExpired(token, NOW, 5000)).toBe(true)
		expect(isTokenExpired(token, NOW, 0)).toBe(false)
	})

	it('无效 token 视为过期，缺少 exp 交给后端判断', () => {
		expect(isTokenExpired('', NOW)).toBe(true)
		expect(isTokenExpired('garbage', NOW)).toBe(true)
		expect(isTokenExpired(makeToken({ sub: 'x' }), NOW)).toBe(false)
		expect(isTokenExpired(makeToken({ exp: 'abc' }), NOW)).toBe(false)
	})
})
