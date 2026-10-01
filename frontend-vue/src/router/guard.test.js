import { describe, it, expect } from 'vitest'
import { resolveNavigation } from './guard'
import { safeRedirectPath } from '../utils/session'
import { errorFromResponse, getErrorMessage, parseRetryAfter } from '../utils/errors'

const route = (fullPath, meta = {}) => ({ fullPath, meta })

describe('resolveNavigation', () => {
	it('未登录访问受保护页面时跳转登录并保留目标', () => {
		expect(resolveNavigation(route('/history?tab=debug', { requiresAuth: true }), { isAuthenticated: false }))
			.toEqual({ path: '/login', query: { redirect: '/history?tab=debug' } })
	})

	it('token 过期时附带 expired 标记', () => {
		expect(resolveNavigation(route('/ai-debug', { requiresAuth: true }), { isAuthenticated: false, sessionExpired: true }))
			.toEqual({ path: '/login', query: { redirect: '/ai-debug', expired: '1' } })
	})

	it('角色不符时拒绝访问', () => {
		const to = route('/admin', { requiresAuth: true, roles: ['admin'] })
		expect(resolveNavigation(to, { isAuthenticated: true, role: 'user' })).toEqual({ path: '/profile', denied: true })
		expect(resolveNavigation(to, { isAuthenticated: true, role: 'admin' })).toBe(true)
	})

	it('声明 roles 的页面隐含需要登录', () => {
		const to = route('/admin', { roles: ['admin'] })
		expect(resolveNavigation(to, { isAuthenticated: false })).toMatchObject({ path: '/login' })
	})

	it('已登录访问登录页跳转个人主页', () => {
		expect(resolveNavigation(route('/login', { guestOnly: true }), { isAuthenticated: true })).toEqual({ path: '/profile' })
		expect(resolveNavigation(route('/login', { guestOnly: true }), { isAuthenticated: false })).toBe(true)
	})
})

describe('safeRedirectPath', () => {
	it('只允许站内路径', () => {
		expect(safeRedirectPath('/history?x=1')).toBe('/history?x=1')
		expect(safeRedirectPath('https://evil.com')).toBe('/profile')
		expect(safeRedirectPath('//evil.com')).toBe('/profile')
		expect(safeRedirectPath('/\\evil.com')).toBe('/profile')
		expect(safeRedirectPath('javascript:alert(1)')).toBe('/profile')
		expect(safeRedirectPath('/login?redirect=/x')).toBe('/profile')
		expect(safeRedirectPath(undefined)).toBe('/profile')
		expect(safeRedirectPath(['/a'])).toBe('/profile')
	})
})

describe('errors', () => {
	it('优先使用服务端 error / message 字段', () => {
		expect(getErrorMessage(errorFromResponse(400, { error: '参数错误' }), '兜底')).toBe('参数错误')
		expect(getErrorMessage(errorFromResponse(400, { message: '缺少字段' }), '兜底')).toBe('缺少字段')
		expect(getErrorMessage(errorFromResponse(500, null), '兜底')).toBe('兜底')
		expect(getErrorMessage(errorFromResponse(502, '<html>Bad Gateway</html>'), '兜底')).toBe('兜底')
	})

	it('保留服务端返回的其他字段', () => {
		const err = errorFromResponse(400, { error: 'x', field: 'email' })
		expect(err.field).toBe('email')
		expect(err.status).toBe(400)
	})

	it('429 提示与 retry_after', () => {
		expect(errorFromResponse(429, {}).error).toBe('请求过于频繁，请稍后再试')
		const err = errorFromResponse(429, { error: 'Too many', retry_after: 30 })
		expect(err.retryAfter).toBe(30)
		expect(err.error).toBe('请求过于频繁，请 30 秒后再试')
		expect(errorFromResponse(429, {}, { retryAfter: '12' }).retryAfter).toBe(12)
	})

	it('parseRetryAfter 支持 HTTP 日期', () => {
		const now = Date.UTC(2026, 0, 1, 0, 0, 0)
		expect(parseRetryAfter(null, new Date(now + 20000).toUTCString(), now)).toBe(20)
		expect(parseRetryAfter(null, 'nonsense', now)).toBeNull()
		expect(parseRetryAfter(null, null, now)).toBeNull()
	})
})
