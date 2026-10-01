/**
 * 主题（浅色 / 深色）管理
 *
 *  - 偏好：'system'（默认，跟随 prefers-color-scheme）| 'light' | 'dark'
 *  - 显式选择时在 <html> 上设置 data-theme，CSS 变量见 styles/tokens.css
 *  - 偏好保存在 localStorage（读写失败时静默回退到跟随系统）
 *  - public/theme-init.js 会在首屏渲染前同步应用已保存的偏好，避免闪烁
 */
import { reactive } from 'vue'

export const THEME_STORAGE_KEY = 'theme'
export const THEME_PREFERENCES = Object.freeze(['system', 'light', 'dark'])

const DARK_QUERY = '(prefers-color-scheme: dark)'

export function readStoredPreference() {
	try {
		const value = localStorage.getItem(THEME_STORAGE_KEY)
		return THEME_PREFERENCES.includes(value) ? value : 'system'
	} catch (e) {
		return 'system'
	}
}

function writeStoredPreference(preference) {
	try {
		if (preference === 'system') {
			localStorage.removeItem(THEME_STORAGE_KEY)
		} else {
			localStorage.setItem(THEME_STORAGE_KEY, preference)
		}
	} catch (e) {
		// 隐私模式等场景下无法持久化，仅在当前页面生效
	}
}

function getMediaQuery() {
	if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return null
	try {
		return window.matchMedia(DARK_QUERY)
	} catch (e) {
		return null
	}
}

export function systemPrefersDark() {
	const mq = getMediaQuery()
	return !!(mq && mq.matches)
}

/**
 * @param {'system'|'light'|'dark'} preference
 * @param {boolean} prefersDark
 * @returns {'light'|'dark'}
 */
export function resolveTheme(preference, prefersDark) {
	if (preference === 'light' || preference === 'dark') return preference
	return prefersDark ? 'dark' : 'light'
}

export const themeState = reactive({
	/** @type {'system'|'light'|'dark'} */
	preference: 'system',
	/** @type {'light'|'dark'} */
	resolved: 'light'
})

function applyToDocument(preference) {
	if (typeof document === 'undefined') return
	const root = document.documentElement
	if (preference === 'system') {
		root.removeAttribute('data-theme')
	} else {
		root.setAttribute('data-theme', preference)
	}
}

/**
 * 设置主题偏好（立即生效并持久化）
 * @param {'system'|'light'|'dark'} preference
 */
export function setThemePreference(preference) {
	const next = THEME_PREFERENCES.includes(preference) ? preference : 'system'
	applyToDocument(next)
	writeStoredPreference(next)
	themeState.preference = next
	themeState.resolved = resolveTheme(next, systemPrefersDark())
}

/** 在浅色与深色之间切换（基于当前实际显示的主题） */
export function toggleTheme() {
	setThemePreference(themeState.resolved === 'dark' ? 'light' : 'dark')
}

let initialized = false

/** 应用启动时调用一次：读取偏好并监听系统主题变化 */
export function initTheme() {
	const preference = readStoredPreference()
	applyToDocument(preference)
	themeState.preference = preference
	themeState.resolved = resolveTheme(preference, systemPrefersDark())

	if (initialized) return
	initialized = true
	const mq = getMediaQuery()
	if (!mq) return
	const onChange = (event) => {
		themeState.resolved = resolveTheme(themeState.preference, event.matches)
	}
	if (typeof mq.addEventListener === 'function') {
		mq.addEventListener('change', onChange)
	} else if (typeof mq.addListener === 'function') {
		mq.addListener(onChange)
	}
}

/**
 * 读取当前主题下某个 CSS 变量的值（用于 ECharts 等无法直接使用 CSS 变量的场景）
 * @param {string} name 例如 '--color-text'
 * @param {string} [fallback]
 */
export function readCssVar(name, fallback = '') {
	if (typeof document === 'undefined' || typeof getComputedStyle !== 'function') return fallback
	const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
	return value || fallback
}
