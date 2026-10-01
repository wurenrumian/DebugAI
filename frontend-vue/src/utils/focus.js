// 焦点管理工具（模态框焦点陷阱等）

const FOCUSABLE_SELECTOR = [
	'a[href]',
	'area[href]',
	'button:not([disabled])',
	'input:not([disabled]):not([type="hidden"])',
	'select:not([disabled])',
	'textarea:not([disabled])',
	'iframe',
	'[contenteditable="true"]',
	'[tabindex]:not([tabindex="-1"])'
].join(',')

/**
 * 返回容器内可通过 Tab 访问的元素（按 DOM 顺序，跳过隐藏元素）
 * @param {HTMLElement} container
 * @returns {HTMLElement[]}
 */
export function getFocusable(container) {
	if (!container) return []
	return Array.from(container.querySelectorAll(FOCUSABLE_SELECTOR)).filter((el) => {
		if (el.getAttribute('aria-hidden') === 'true') return false
		if (el.closest('[inert]')) return false
		// jsdom 中没有布局信息，getClientRects 为空时仍视为可见
		const style = typeof getComputedStyle === 'function' ? getComputedStyle(el) : null
		if (style && (style.display === 'none' || style.visibility === 'hidden')) return false
		return true
	})
}

/**
 * 处理 Tab / Shift+Tab，使焦点在容器内循环
 * @param {KeyboardEvent} event
 * @param {HTMLElement} container
 * @returns {boolean} 是否处理了该事件
 */
export function trapTabKey(event, container) {
	if (event.key !== 'Tab' || !container) return false
	const focusable = getFocusable(container)
	if (focusable.length === 0) {
		event.preventDefault()
		container.focus()
		return true
	}
	const first = focusable[0]
	const last = focusable[focusable.length - 1]
	const active = document.activeElement
	const inside = container.contains(active)

	if (event.shiftKey) {
		if (!inside || active === first || active === container) {
			event.preventDefault()
			last.focus()
			return true
		}
	} else if (!inside || active === last) {
		event.preventDefault()
		first.focus()
		return true
	}
	return false
}
