import { describe, it, expect, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AppModal from './AppModal.vue'

function press(key, options = {}) {
	const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...options })
	document.activeElement.dispatchEvent(event)
	return event
}

describe('AppModal', () => {
	afterEach(() => {
		document.body.innerHTML = ''
		document.body.style.overflow = ''
	})

	const mountModal = () => mount(AppModal, {
		props: { title: '测试对话框' },
		slots: { default: '<input id="first" /><button id="last">确定</button>' },
		attachTo: document.body
	})

	it('具有对话框语义并以标题命名', async () => {
		const wrapper = mountModal()
		await flushPromises()
		const dialog = document.querySelector('[role="dialog"]')
		expect(dialog.getAttribute('aria-modal')).toBe('true')
		const title = document.getElementById(dialog.getAttribute('aria-labelledby'))
		expect(title.textContent).toBe('测试对话框')
		wrapper.unmount()
	})

	it('打开时聚焦第一个表单控件，并锁定页面滚动', async () => {
		const wrapper = mountModal()
		await flushPromises()
		expect(document.activeElement.id).toBe('first')
		expect(document.body.style.overflow).toBe('hidden')
		wrapper.unmount()
		expect(document.body.style.overflow).toBe('')
	})

	it('Esc 触发 close', async () => {
		const wrapper = mountModal()
		await flushPromises()
		press('Escape')
		expect(wrapper.emitted('close')).toHaveLength(1)
		wrapper.unmount()
	})

	it('Tab 焦点在对话框内循环', async () => {
		const wrapper = mountModal()
		await flushPromises()
		document.getElementById('last').focus()
		const forward = press('Tab')
		expect(forward.defaultPrevented).toBe(true)
		// 第一个可聚焦元素是右上角关闭按钮
		expect(document.activeElement.getAttribute('aria-label')).toBe('关闭对话框')

		const backward = press('Tab', { shiftKey: true })
		expect(backward.defaultPrevented).toBe(true)
		expect(document.activeElement.id).toBe('last')
		wrapper.unmount()
	})

	it('关闭后焦点回到打开前的元素', async () => {
		const trigger = document.createElement('button')
		document.body.appendChild(trigger)
		trigger.focus()
		const wrapper = mountModal()
		await flushPromises()
		expect(document.activeElement).not.toBe(trigger)
		wrapper.unmount()
		expect(document.activeElement).toBe(trigger)
	})
})
