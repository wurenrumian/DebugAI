/**
 * 相对时间格式化（刚刚 / N分钟前 / N小时前 / N天前 / 完整日期）
 * @param {number|string|Date} timestamp
 */
export function formatRelativeTime(timestamp) {
	const date = new Date(timestamp)
	const now = new Date()
	const diff = now - date

	if (diff < 60000) return '刚刚'
	if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
	if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
	if (diff < 604800000) return Math.floor(diff / 86400000) + '天前'

	return date.toLocaleDateString('zh-CN', {
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit'
	})
}
