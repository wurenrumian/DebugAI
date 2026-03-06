import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

/**
 * 键盘快捷键组合式函数
 * @param {Array<string>} keys - 要监听的按键组合数组，如 ['escape', 'enter', 'ctrl+enter']
 * @param {Function} handler - 按键处理函数，接收 (event, keyCombo, options) 参数
 */
export function useKeyboardShortcut(keys, handler) {
	const router = useRouter()

	const handleKeydown = (event) => {
		// 检查是否在输入框内
		const target = event.target
		const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)
		const isEditable = target.isContentEditable

		// 构建按键组合字符串
		const keyCombo = buildKeyCombo(event)

		// 检查是否匹配
		if (keys.includes(keyCombo)) {
			// 某些快捷键在输入框内需要特殊处理
			if (shouldHandleInInput(keyCombo, isInput)) {
				event.preventDefault()
				handler(event, keyCombo, { isInput })
			}
		}
	}

	const buildKeyCombo = (event) => {
		const parts = []
		if (event.ctrlKey) parts.push('ctrl')
		if (event.altKey) parts.push('alt')
		if (event.shiftKey) parts.push('shift')
		if (event.metaKey) parts.push('meta') // Mac Cmd 键
		parts.push(event.key.toLowerCase())
		return parts.join('+')
	}

	const shouldHandleInInput = (keyCombo, isInput) => {
		// ESC 在输入框内也应该处理（关闭模态框等）
		if (keyCombo === 'escape') return true
		// Ctrl+Enter 在输入框内需要处理
		if (keyCombo === 'ctrl+enter' && isInput) return true
		// 纯 Enter 在输入框内不处理（除非是 textarea 且想提交）
		if (keyCombo === 'enter' && isInput) return false
		// 其他快捷键不在输入框内处理
		return !isInput
	}

	onMounted(() => {
		window.addEventListener('keydown', handleKeydown)
	})

	onUnmounted(() => {
		window.removeEventListener('keydown', handleKeydown)
	})

	return { router }
}

/**
 * 标签页切换快捷键辅助函数
 * @param {Array<string>} tabKeys - 标签页快捷键，如 ['ctrl+1', 'ctrl+2', 'ctrl+3']
 * @param {Function} switchTab - 切换标签的函数，接收索引参数
 */
export function useTabShortcut(tabKeys, switchTab) {
	const handleKeydown = (event) => {
		const keyCombo = buildKeyCombo(event)
		const index = tabKeys.indexOf(keyCombo)
		if (index !== -1) {
			event.preventDefault()
			switchTab(index)
		}
	}

	const buildKeyCombo = (event) => {
		const parts = []
		if (event.ctrlKey) parts.push('ctrl')
		if (event.altKey) parts.push('alt')
		if (event.shiftKey) parts.push('shift')
		if (event.metaKey) parts.push('meta')
		parts.push(event.key.toLowerCase())
		return parts.join('+')
	}

	onMounted(() => {
		window.addEventListener('keydown', handleKeydown)
	})

	onUnmounted(() => {
		window.removeEventListener('keydown', handleKeydown)
	})
}
