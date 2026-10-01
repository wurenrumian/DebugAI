import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, ref, nextTick } from 'vue'
import CodeEditor from './CodeEditor.vue'

// jsdom 缺少部分布局 API，CodeMirror 测量时会用到
if (!document.createRange().getClientRects) {
	Range.prototype.getClientRects = () => []
	Range.prototype.getBoundingClientRect = () => ({ top: 0, left: 0, right: 0, bottom: 0, width: 0, height: 0 })
}

async function waitForEditor(wrapper) {
	for (let i = 0; i < 50; i++) {
		await flushPromises()
		if (wrapper.find('.cm-editor').exists()) return
		await new Promise(r => setTimeout(r, 10))
	}
	throw new Error('CodeMirror 未加载')
}

describe('CodeEditor', () => {
	it('加载前显示可输入的 textarea 回退，并支持 v-model', async () => {
		const wrapper = mount(CodeEditor, {
			props: { modelValue: 'int a;', 'onUpdate:modelValue': (v) => wrapper.setProps({ modelValue: v }) }
		})
		const textarea = wrapper.find('textarea')
		expect(textarea.exists()).toBe(true)
		expect(textarea.element.value).toBe('int a;')
		await textarea.setValue('int b;')
		expect(wrapper.props('modelValue')).toBe('int b;')
		wrapper.unmount()
	})

	it('CodeMirror 加载后与 v-model 双向同步', async () => {
		const Host = defineComponent({
			components: { CodeEditor },
			setup() {
				const code = ref('int main() {}')
				return { code }
			},
			template: '<CodeEditor v-model="code" />'
		})
		const wrapper = mount(Host, { attachTo: document.body })
		await waitForEditor(wrapper)

		const content = wrapper.find('.cm-content')
		expect(content.text()).toContain('int main() {}')

		// 外部修改 → 编辑器
		wrapper.vm.code = 'return 0;'
		await nextTick()
		expect(wrapper.find('.cm-content').text()).toContain('return 0;')

		// 编辑器修改 → 外部
		const { EditorView } = await import('@codemirror/view')
		const view = EditorView.findFromDOM(wrapper.find('.cm-editor').element)
		view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: 'x = 1;' } })
		expect(wrapper.vm.code).toBe('x = 1;')

		wrapper.unmount()
	})

	it('disabled 时编辑器只读', async () => {
		const wrapper = mount(CodeEditor, {
			props: { modelValue: 'a', disabled: true },
			attachTo: document.body
		})
		await waitForEditor(wrapper)
		expect(wrapper.find('.cm-content').attributes('contenteditable')).toBe('false')
		await wrapper.setProps({ disabled: false })
		expect(wrapper.find('.cm-content').attributes('contenteditable')).toBe('true')
		wrapper.unmount()
	})

	it('Tab 键用于缩进（indentWithTab）', async () => {
		const onUpdate = vi.fn()
		const wrapper = mount(CodeEditor, {
			props: { modelValue: 'x', 'onUpdate:modelValue': onUpdate },
			attachTo: document.body
		})
		await waitForEditor(wrapper)
		const { EditorView } = await import('@codemirror/view')
		const view = EditorView.findFromDOM(wrapper.find('.cm-editor').element)
		view.focus()
		view.dispatch({ selection: { anchor: 0 } })
		const event = new KeyboardEvent('keydown', { key: 'Tab', keyCode: 9, bubbles: true, cancelable: true })
		view.contentDOM.dispatchEvent(event)
		expect(event.defaultPrevented).toBe(true)
		expect(view.state.doc.toString()).toBe('    x')
		expect(onUpdate).toHaveBeenLastCalledWith('    x')
		wrapper.unmount()
	})
})
