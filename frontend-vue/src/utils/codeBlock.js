import { renderMarkdown } from './markdown'

/**
 * 把一段源码包装成 Markdown 围栏代码块。
 * 围栏长度取“代码中最长的连续反引号 + 1”（至少 3 个），保证代码内容无法提前闭合围栏。
 * @param {string} code
 * @param {string} [lang]
 */
export function toFencedCode(code, lang = 'cpp') {
	const text = String(code ?? '').replace(/\r\n?/g, '\n').replace(/\n+$/, '')
	const runs = text.match(/`+/g) || []
	const longest = runs.reduce((max, run) => Math.max(max, run.length), 0)
	const fence = '`'.repeat(Math.max(3, longest + 1))
	const language = /^[a-z0-9+#-]*$/i.test(lang) ? lang : ''
	return `${fence}${language}\n${text}\n${fence}`
}

/**
 * 只读代码高亮：复用 renderMarkdown（markdown-it + highlight.js + DOMPurify），结果可安全用于 v-html
 * @param {string} code
 * @param {string} [lang]
 */
export function renderCodeBlock(code, lang = 'cpp') {
	if (code === null || code === undefined || code === '') return ''
	return renderMarkdown(toFencedCode(code, lang))
}
