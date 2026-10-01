/**
 * 路由守卫的纯逻辑部分（便于单元测试）
 *
 * 路由 meta 约定：
 *  - requiresAuth: true   需要登录
 *  - guestOnly: true      已登录用户访问时跳转到个人主页（登录 / 注册等页面）
 *  - roles: ['admin']     仅允许指定用户类型访问（user_type，见 stores/auth.js ROLES）
 *
 * 注意：前端校验只用于改善体验，真正的权限控制在后端。
 */

export const HOME_PATH = '/profile'
export const LOGIN_PATH = '/login'

/**
 * @param {{ fullPath: string, meta?: Record<string, any> }} to
 * @param {{ isAuthenticated: boolean, sessionExpired?: boolean, role?: string }} session
 * @returns {true | { path: string, query?: Record<string, string>, denied?: boolean }}
 */
export function resolveNavigation(to, session) {
	const meta = to.meta || {}
	const roles = Array.isArray(meta.roles) ? meta.roles : []
	const requiresAuth = !!meta.requiresAuth || roles.length > 0

	if (requiresAuth && !session.isAuthenticated) {
		const query = { redirect: to.fullPath }
		if (session.sessionExpired) query.expired = '1'
		return { path: LOGIN_PATH, query }
	}

	if (roles.length > 0 && !roles.includes(session.role)) {
		return { path: HOME_PATH, denied: true }
	}

	if (meta.guestOnly && session.isAuthenticated) {
		return { path: HOME_PATH }
	}

	return true
}
