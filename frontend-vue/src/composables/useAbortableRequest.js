import { onBeforeUnmount } from 'vue'

/**
 * 管理可取消的请求（主要用于 AI 流式接口）
 *  - start()：取消上一个未完成的请求，返回新的 AbortSignal
 *  - cancel()：取消当前请求
 *  - 组件卸载时自动取消
 */
export function useAbortableRequest() {
	let controller = null

	const cancel = () => {
		if (controller) {
			controller.abort()
			controller = null
		}
	}

	const start = () => {
		cancel()
		controller = new AbortController()
		return controller.signal
	}

	onBeforeUnmount(cancel)

	return { start, cancel }
}
