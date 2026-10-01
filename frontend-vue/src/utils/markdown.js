/**
 * 统一的安全 Markdown 渲染器
 *
 * 所有需要通过 v-html 输出的内容都必须经过 renderMarkdown()：
 *  - markdown-it 关闭原始 HTML（html: false），用户/AI 输入中的标签会被转义成文本
 *  - DOMPurify 对最终 HTML 再做一次白名单过滤（纵深防御）
 *  - highlight.js 只按需注册常用语言，控制打包体积
 *
 * 学生提交的代码、题目描述等内容优先使用 {{ }} 纯文本插值，不要走 v-html。
 */
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js/lib/core'
import c from 'highlight.js/lib/languages/c'
import cpp from 'highlight.js/lib/languages/cpp'
import python from 'highlight.js/lib/languages/python'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import plaintext from 'highlight.js/lib/languages/plaintext'
import { escapeHtml } from './html'

export { escapeHtml }

hljs.registerLanguage('c', c)
hljs.registerLanguage('cpp', cpp)
hljs.registerLanguage('python', python)
hljs.registerLanguage('java', java)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('plaintext', plaintext)

const md = new MarkdownIt({
	html: false, // 绝不信任输入中的 HTML
	linkify: false,
	breaks: true, // 保持原来 "换行即 <br>" 的显示效果
	highlight(code, lang) {
		const language = (lang || '').trim().toLowerCase()
		if (language && hljs.getLanguage(language)) {
			try {
				const highlighted = hljs.highlight(code, { language, ignoreIllegals: true }).value
				return `<pre class="hljs"><code class="language-${md.utils.escapeHtml(language)}">${highlighted}</code></pre>`
			} catch (e) {
				// 回退到 markdown-it 默认的转义输出
			}
		}
		return ''
	}
})

// 禁止渲染图片：避免 AI 输出被诱导加载外部图片（追踪 / 数据外带）
md.disable('image')

const ALLOWED_TAGS = [
	'p', 'br', 'hr', 'strong', 'em', 'del', 's', 'blockquote',
	'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
	'ul', 'ol', 'li',
	'pre', 'code', 'span',
	'a',
	'table', 'thead', 'tbody', 'tr', 'th', 'td'
]
const ALLOWED_ATTR = ['href', 'title', 'class', 'start']

let hooksInstalled = false
function installHooks() {
	if (hooksInstalled || typeof DOMPurify.addHook !== 'function') return
	DOMPurify.addHook('afterSanitizeAttributes', (node) => {
		if (node.tagName === 'A' && node.getAttribute('href')) {
			node.setAttribute('target', '_blank')
			node.setAttribute('rel', 'noopener noreferrer nofollow')
		}
	})
	hooksInstalled = true
}

/**
 * 对 HTML 片段做白名单过滤
 * @param {string} html
 * @returns {string}
 */
export function sanitizeHtml(html) {
	if (!html) return ''
	if (typeof DOMPurify.sanitize !== 'function') {
		// 非浏览器环境（无 DOM）下无法安全过滤，直接全部转义
		return escapeHtml(html)
	}
	installHooks()
	return DOMPurify.sanitize(html, {
		ALLOWED_TAGS,
		ALLOWED_ATTR: [...ALLOWED_ATTR, 'target', 'rel'],
		ALLOW_DATA_ATTR: false
	})
}

function toText(content) {
	if (content === null || content === undefined) return ''
	if (typeof content === 'string') return content
	if (typeof content === 'number' || typeof content === 'boolean') return String(content)
	try {
		return JSON.stringify(content, null, 2)
	} catch (e) {
		return String(content)
	}
}

/**
 * 将 Markdown 文本渲染为安全的 HTML，可直接用于 v-html
 * @param {unknown} content
 * @returns {string}
 */
export function renderMarkdown(content) {
	const text = toText(content)
	if (!text) return ''
	return sanitizeHtml(md.render(text))
}

export default renderMarkdown
