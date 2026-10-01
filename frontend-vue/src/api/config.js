// 接口基础地址：通过 VITE_API_BASE_URL 配置，默认同源（由 nginx / vite 代理转发）
export const API_BASE_URL = String(import.meta.env.VITE_API_BASE_URL || '').replace(/\/+$/, '')

// 普通接口超时
export const DEFAULT_TIMEOUT_MS = 60000
// AI 相关接口（非流式与流式）超时：略大于后端 AI_STREAM_TIMEOUT（180s），让后端先返回友好的超时提示
export const AI_TIMEOUT_MS = 200000
// 退出登录请求超时（失败也会清理本地状态，没必要久等）
export const LOGOUT_TIMEOUT_MS = 10000

export function buildUrl(path) {
	return `${API_BASE_URL}${path}`
}
