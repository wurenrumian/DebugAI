// 生成页面内唯一的 DOM id（用于 label / aria-labelledby 关联）
let seq = 0

export function uid(prefix = 'uid') {
	seq += 1
	return `${prefix}-${seq}`
}
