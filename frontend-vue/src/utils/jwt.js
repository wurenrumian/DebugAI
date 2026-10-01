/**
 * 轻量 JWT 工具（仅用于前端判断登录态是否过期，不做签名校验；
 * 真正的鉴权始终由后端完成）
 */

function base64UrlDecode(segment) {
	let base64 = segment.replace(/-/g, '+').replace(/_/g, '/')
	const pad = base64.length % 4
	if (pad === 1) throw new Error('invalid base64url')
	if (pad) base64 += '='.repeat(4 - pad)

	const binary = atob(base64)
	const bytes = new Uint8Array(binary.length)
	for (let i = 0; i < binary.length; i++) {
		bytes[i] = binary.charCodeAt(i)
	}
	return new TextDecoder().decode(bytes)
}

/**
 * 解码 JWT payload，失败返回 null（不会抛异常）
 * @param {string} token
 * @returns {Record<string, any> | null}
 */
export function decodeJwtPayload(token) {
	if (typeof token !== 'string' || !token) return null
	const parts = token.split('.')
	if (parts.length !== 3 || !parts[1]) return null
	try {
		const payload = JSON.parse(base64UrlDecode(parts[1]))
		return payload && typeof payload === 'object' && !Array.isArray(payload) ? payload : null
	} catch (e) {
		return null
	}
}

/**
 * 获取 token 过期时间（毫秒时间戳），没有 exp 或无法解析时返回 null
 * @param {string} token
 * @returns {number | null}
 */
export function getTokenExpiry(token) {
	const payload = decodeJwtPayload(token)
	if (!payload) return null
	const exp = Number(payload.exp)
	return Number.isFinite(exp) && exp > 0 ? exp * 1000 : null
}

/**
 * 判断 token 是否已过期 / 无效
 *  - 空值或无法解析：视为过期（无效）
 *  - 没有 exp 声明：交给后端判定，视为未过期
 * @param {string} token
 * @param {number} [now] 当前毫秒时间戳
 * @param {number} [leewayMs] 提前多少毫秒视为过期，用于抵消时钟偏差
 */
export function isTokenExpired(token, now = Date.now(), leewayMs = 5000) {
	if (!decodeJwtPayload(token)) return true
	const expiry = getTokenExpiry(token)
	if (expiry === null) return false
	return now >= expiry - leewayMs
}
