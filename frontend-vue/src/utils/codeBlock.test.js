import { describe, it, expect } from 'vitest'
import { renderCodeBlock, toFencedCode } from './codeBlock'

describe('toFencedCode', () => {
	it('默认使用三个反引号围栏', () => {
		expect(toFencedCode('int a;', 'cpp')).toBe('```cpp\nint a;\n```')
	})

	it('围栏长度大于代码中最长的反引号串，代码无法提前闭合围栏', () => {
		const fenced = toFencedCode('a ```` b', 'cpp')
		expect(fenced.startsWith('`````cpp\n')).toBe(true)
		expect(fenced.endsWith('\n`````')).toBe(true)
	})

	it('非法语言名被丢弃', () => {
		expect(toFencedCode('x', 'c"onerror=1')).toBe('```\nx\n```')
	})
})

describe('renderCodeBlock', () => {
	it('代码中的 HTML 被转义为文本', () => {
		const html = renderCodeBlock('#include <stdio.h>\n<img src=x onerror=alert(1)>', 'cpp')
		expect(html).toContain('&lt;stdio.h&gt;')
		expect(html).not.toContain('<img')
		expect(html).toContain('class="hljs"')
	})

	it('包含反引号的代码不会逃逸出代码块', () => {
		const html = renderCodeBlock('```\n<script>alert(1)</script>\n```', 'cpp')
		expect(html).not.toContain('<script')
		expect(html.match(/<pre/g)).toHaveLength(1)
	})

	it('空代码返回空字符串', () => {
		expect(renderCodeBlock('')).toBe('')
		expect(renderCodeBlock(null)).toBe('')
	})
})
