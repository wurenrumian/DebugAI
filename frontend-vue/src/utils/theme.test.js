import { describe, it, expect, beforeEach, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import {
	THEME_STORAGE_KEY,
	initTheme,
	readStoredPreference,
	resolveTheme,
	setThemePreference,
	themeState,
	toggleTheme
} from './theme'

function mockMatchMedia(matches) {
	window.matchMedia = vi.fn().mockImplementation((query) => ({
		matches,
		media: query,
		addEventListener: vi.fn(),
		removeEventListener: vi.fn()
	}))
}

describe('resolveTheme', () => {
	it('显式偏好优先，system 跟随系统', () => {
		expect(resolveTheme('light', true)).toBe('light')
		expect(resolveTheme('dark', false)).toBe('dark')
		expect(resolveTheme('system', true)).toBe('dark')
		expect(resolveTheme('system', false)).toBe('light')
	})
})

describe('主题偏好读写', () => {
	beforeEach(() => {
		localStorage.clear()
		document.documentElement.removeAttribute('data-theme')
		mockMatchMedia(false)
	})

	it('默认跟随系统且不设置 data-theme', () => {
		initTheme()
		expect(themeState.preference).toBe('system')
		expect(themeState.resolved).toBe('light')
		expect(document.documentElement.hasAttribute('data-theme')).toBe(false)
	})

	it('跟随系统深色', () => {
		mockMatchMedia(true)
		initTheme()
		expect(themeState.resolved).toBe('dark')
	})

	it('toggleTheme 切换并持久化到 localStorage', () => {
		initTheme()
		toggleTheme()
		expect(themeState.preference).toBe('dark')
		expect(themeState.resolved).toBe('dark')
		expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
		expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark')

		toggleTheme()
		expect(themeState.resolved).toBe('light')
		expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
	})

	it('恢复为跟随系统时移除属性与存储', () => {
		setThemePreference('dark')
		setThemePreference('system')
		expect(document.documentElement.hasAttribute('data-theme')).toBe(false)
		expect(localStorage.getItem(THEME_STORAGE_KEY)).toBeNull()
	})

	it('非法存储值回退到 system', () => {
		localStorage.setItem(THEME_STORAGE_KEY, 'purple')
		expect(readStoredPreference()).toBe('system')
	})

	it('localStorage 不可用时不抛异常', () => {
		const getItem = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
			throw new Error('denied')
		})
		const setItem = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
			throw new Error('denied')
		})
		expect(readStoredPreference()).toBe('system')
		expect(() => setThemePreference('dark')).not.toThrow()
		expect(themeState.resolved).toBe('dark')
		getItem.mockRestore()
		setItem.mockRestore()
	})
})

describe('tokens.css', () => {
	it('跟随系统的深色变量与手动深色变量保持一致', () => {
		// vitest 从 frontend-vue 目录运行
		const css = readFileSync(resolve(process.cwd(), 'src/styles/tokens.css'), 'utf8')
		const extract = (block) => block.match(/--[\w-]+:\s*[^;]+;/g).map(s => s.replace(/\s+/g, ' '))
		const mediaStart = css.indexOf('@media (prefers-color-scheme: dark)')
		const manualStart = css.indexOf(":root[data-theme='dark']")
		expect(mediaStart).toBeGreaterThan(0)
		expect(manualStart).toBeGreaterThan(mediaStart)
		const mediaVars = extract(css.slice(mediaStart, manualStart))
		const manualVars = extract(css.slice(manualStart))
		expect(manualVars.length).toBeGreaterThan(40)
		expect(manualVars).toEqual(mediaVars)
	})
})
