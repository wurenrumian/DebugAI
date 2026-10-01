/**
 * 转义 HTML 特殊字符（用于必须拼接 HTML 字符串的场景，如 ECharts tooltip formatter）
 * @param {unknown} text
 * @returns {string}
 */
export function escapeHtml(text) {
	return String(text ?? '')
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&#39;')
}
