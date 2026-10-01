import { describe, it, expect } from 'vitest'
import { renderMarkdown, sanitizeHtml, escapeHtml } from './markdown'

function toDom(html) {
	const div = document.createElement('div')
	div.innerHTML = html
	return div
}

function expectNoActiveContent(html) {
	const dom = toDom(html)
	expect(dom.querySelector('script, img, iframe, object, embed, svg, style, form, input')).toBeNull()
	for (const el of dom.querySelectorAll('*')) {
		for (const attr of el.attributes) {
			expect(attr.name.startsWith('on')).toBe(false)
			expect(attr.value.toLowerCase()).not.toMatch(/^\s*(javascript|vbscript|data):/)
		}
	}
}

describe('renderMarkdown XSS 防护', () => {
	const payloads = [
		'<img src=x onerror=alert(1)>',
		'<script>alert(1)</script>',
		'<svg onload=alert(1)></svg>',
		'<iframe src="javascript:alert(1)"></iframe>',
		'<a href="javascript:alert(1)">click</a>',
		'[click](javascript:alert(1))',
		'[click](JaVaScRiPt:alert(1))',
		'[click](data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==)',
		'![x](https://evil.example/track.png)',
		'<div onclick="alert(1)">hi</div>',
		'`<img src=x onerror=alert(1)>`',
		'```\n<script>alert(1)</script>\n```',
		'```c" onmouseover="alert(1)\nint a;\n```'
	]

	for (const payload of payloads) {
		it(`neutralizes: ${payload.replace(/\n/g, '\\n')}`, () => {
			expectNoActiveContent(renderMarkdown(payload))
		})
	}

	it('将原始 HTML 作为文本显示而不是执行', () => {
		const html = renderMarkdown('<img src=x onerror=alert(1)>')
		expect(html).toContain('&lt;img')
		expect(toDom(html).textContent).toContain('<img src=x onerror=alert(1)>')
	})

	it('javascript: 链接不会生成 href', () => {
		const dom = toDom(renderMarkdown('[click](javascript:alert(1))'))
		const link = dom.querySelector('a')
		if (link) {
			expect(link.getAttribute('href') || '').not.toMatch(/javascript:/i)
		}
	})

	it('正常链接保留并添加 rel/target', () => {
		const dom = toDom(renderMarkdown('[文档](https://example.com/doc)'))
		const link = dom.querySelector('a')
		expect(link.getAttribute('href')).toBe('https://example.com/doc')
		expect(link.getAttribute('rel')).toContain('noopener')
		expect(link.getAttribute('target')).toBe('_blank')
	})

	it('sanitizeHtml 过滤危险标签和属性', () => {
		const out = sanitizeHtml('<p onclick="x()">ok</p><script>alert(1)</script><img src=x onerror=alert(1)>')
		expectNoActiveContent(out)
		expect(toDom(out).textContent).toBe('ok')
	})
})

describe('renderMarkdown 代码显示', () => {
	it('代码块中的 #include <stdio.h> 正常显示', () => {
		const html = renderMarkdown('```c\n#include <stdio.h>\nint main() { return 0; }\n```')
		const dom = toDom(html)
		const code = dom.querySelector('pre code')
		expect(code).not.toBeNull()
		expect(code.textContent).toContain('#include <stdio.h>')
		expect(code.textContent).toContain('int main()')
	})

	it('未指定语言的代码块中的尖括号正常显示', () => {
		const dom = toDom(renderMarkdown('```\n#include <iostream>\nvector<int> v;\n```'))
		expect(dom.querySelector('pre code').textContent).toContain('#include <iostream>\nvector<int> v;')
	})

	it('正文中的 #include <stdio.h> 不被吞掉', () => {
		const dom = toDom(renderMarkdown('请检查 #include <stdio.h> 是否缺失'))
		expect(dom.textContent).toContain('#include <stdio.h>')
	})

	it('行内代码转义', () => {
		const dom = toDom(renderMarkdown('使用 `a < b && c > d` 判断'))
		expect(dom.querySelector('code').textContent).toBe('a < b && c > d')
	})

	it('支持常用语言高亮', () => {
		const html = renderMarkdown('```python\ndef f():\n    return 1\n```')
		expect(html).toContain('hljs-keyword')
	})

	it('换行转换为 <br>', () => {
		expect(renderMarkdown('第一行\n第二行')).toContain('<br>')
	})

	it('处理空值和非字符串', () => {
		expect(renderMarkdown('')).toBe('')
		expect(renderMarkdown(null)).toBe('')
		expect(renderMarkdown(undefined)).toBe('')
		expect(toDom(renderMarkdown(42)).textContent).toContain('42')
		expectNoActiveContent(renderMarkdown({ text: '<img src=x onerror=alert(1)>' }))
	})
})

describe('escapeHtml', () => {
	it('转义全部特殊字符', () => {
		expect(escapeHtml(`<a href="x">'&'</a>`)).toBe('&lt;a href=&quot;x&quot;&gt;&#39;&amp;&#39;&lt;/a&gt;')
		expect(escapeHtml(null)).toBe('')
	})
})
