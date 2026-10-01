// 代码评价等级（与后端 / AI 输出保持一致：优秀 / 合格 / 待改进）
export const GRADE_META = Object.freeze({
	优秀: { tone: 'excellent', icon: 'check-circle' },
	合格: { tone: 'pass', icon: 'check' },
	待改进: { tone: 'improve', icon: 'alert' }
})

/**
 * @param {string} grade
 * @returns {{ tone: string, icon: string, label: string }}
 */
export function gradeMeta(grade) {
	const key = typeof grade === 'string' ? grade.trim() : ''
	const meta = GRADE_META[key]
	if (meta) return { ...meta, label: key }
	return { tone: 'neutral', icon: '', label: key || 'N/A' }
}
