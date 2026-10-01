/**
 * CodeMirror 6 编辑器创建逻辑（由 CodeEditor.vue 动态 import，单独打包，不进入主 chunk）
 *
 * 配色全部使用 CSS 变量（styles/tokens.css 中的 --code-* / --hl-*），
 * 因此切换浅色 / 深色主题时无需重建编辑器；dark 标志只影响 CodeMirror 内置面板（搜索框等）的基础样式。
 */
import { minimalSetup } from 'codemirror'
import {
	EditorView,
	keymap,
	placeholder as placeholderExt,
	lineNumbers,
	highlightActiveLine,
	highlightActiveLineGutter,
	dropCursor
} from '@codemirror/view'
import { EditorState, Compartment, Prec } from '@codemirror/state'
import { indentWithTab } from '@codemirror/commands'
import { HighlightStyle, syntaxHighlighting, indentUnit, indentOnInput, bracketMatching } from '@codemirror/language'
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { cpp } from '@codemirror/lang-cpp'
import { tags as t } from '@lezer/highlight'

const highlightStyle = HighlightStyle.define([
	{ tag: [t.comment, t.lineComment, t.blockComment, t.docComment], color: 'var(--hl-comment)', fontStyle: 'italic' },
	{ tag: [t.keyword, t.controlKeyword, t.operatorKeyword, t.definitionKeyword, t.modifier, t.self], color: 'var(--hl-keyword)' },
	{ tag: [t.string, t.special(t.string), t.character, t.regexp], color: 'var(--hl-string)' },
	{ tag: [t.number, t.bool, t.null, t.atom], color: 'var(--hl-number)' },
	{ tag: [t.function(t.variableName), t.function(t.propertyName), t.macroName], color: 'var(--hl-function)' },
	{ tag: [t.typeName, t.className, t.namespace, t.standard(t.typeName)], color: 'var(--hl-type)' },
	{ tag: [t.processingInstruction, t.meta, t.annotation], color: 'var(--hl-meta)' },
	{ tag: [t.propertyName, t.attributeName], color: 'var(--hl-attr)' },
	{ tag: [t.variableName, t.definition(t.variableName)], color: 'var(--hl-variable)' },
	{ tag: t.invalid, color: 'var(--color-danger)' }
])

const baseTheme = EditorView.theme({
	'&': {
		color: 'var(--code-text)',
		backgroundColor: 'var(--code-bg)',
		fontSize: '13.5px',
		height: '100%'
	},
	'&.cm-focused': {
		outline: 'none'
	},
	'.cm-scroller': {
		fontFamily: 'var(--font-mono)',
		lineHeight: '1.65',
		overflow: 'auto'
	},
	'.cm-content': {
		padding: '10px 0',
		caretColor: 'var(--color-primary-text)'
	},
	'.cm-line': {
		padding: '0 12px 0 8px'
	},
	'.cm-cursor, .cm-dropCursor': {
		borderLeftColor: 'var(--color-text)',
		borderLeftWidth: '2px'
	},
	'.cm-gutters': {
		color: 'var(--code-gutter-text)',
		backgroundColor: 'var(--code-bg)',
		borderRight: '1px solid var(--code-border)'
	},
	'.cm-lineNumbers .cm-gutterElement': {
		padding: '0 10px 0 12px',
		minWidth: '36px'
	},
	'.cm-activeLine': {
		backgroundColor: 'var(--code-active-line)'
	},
	'.cm-activeLineGutter': {
		color: 'var(--color-text-secondary)',
		backgroundColor: 'var(--code-active-line)'
	},
	'&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
		backgroundColor: 'var(--color-selection) !important'
	},
	'.cm-matchingBracket, &.cm-focused .cm-matchingBracket': {
		color: 'inherit',
		backgroundColor: 'var(--color-primary-soft)',
		outline: '1px solid var(--color-primary-soft-border)'
	},
	'.cm-nonmatchingBracket, &.cm-focused .cm-nonmatchingBracket': {
		backgroundColor: 'var(--color-danger-soft)'
	},
	'.cm-selectionMatch': {
		backgroundColor: 'var(--color-primary-soft)'
	},
	'.cm-placeholder': {
		color: 'var(--color-text-muted)',
		fontStyle: 'normal'
	},
	'.cm-foldPlaceholder': {
		color: 'var(--color-text-muted)',
		backgroundColor: 'var(--color-surface-3)',
		border: '1px solid var(--color-border)'
	},
	'.cm-panels': {
		color: 'var(--color-text)',
		backgroundColor: 'var(--color-surface-2)'
	},
	'.cm-panels.cm-panels-bottom': {
		borderTop: '1px solid var(--color-border)'
	},
	'.cm-tooltip': {
		color: 'var(--color-text)',
		backgroundColor: 'var(--color-surface)',
		border: '1px solid var(--color-border)'
	}
})

/**
 * @param {object} options
 * @param {HTMLElement} options.parent
 * @param {string} options.doc
 * @param {string} [options.placeholder]
 * @param {boolean} [options.readOnly]
 * @param {boolean} [options.dark]
 * @param {Record<string,string>} [options.contentAttributes] 例如 aria-labelledby / aria-describedby
 * @param {(value: string) => void} options.onChange
 */
export function createCodeEditor(options) {
	const editableCompartment = new Compartment()
	const themeCompartment = new Compartment()

	const editableExtensions = (readOnly) => [
		EditorState.readOnly.of(!!readOnly),
		EditorView.editable.of(!readOnly)
	]

	const view = new EditorView({
		parent: options.parent,
		state: EditorState.create({
			doc: options.doc || '',
			extensions: [
				// 精简配置：不含自动补全 / lint / 搜索面板，控制按需加载的体积
				minimalSetup,
				lineNumbers(),
				highlightActiveLine(),
				highlightActiveLineGutter(),
				dropCursor(),
				indentOnInput(),
				bracketMatching(),
				closeBrackets(),
				keymap.of([...closeBracketsKeymap, indentWithTab]),
				// Ctrl/Cmd+Enter 留给页面的“提交”快捷键（事件会继续冒泡到 window），这里只阻止默认的插入空行
				Prec.highest(keymap.of([{ key: 'Mod-Enter', run: () => true }])),
				indentUnit.of('    '),
				EditorState.tabSize.of(4),
				cpp(),
				syntaxHighlighting(highlightStyle),
				baseTheme,
				themeCompartment.of(EditorView.theme({}, { dark: !!options.dark })),
				editableCompartment.of(editableExtensions(options.readOnly)),
				options.placeholder ? placeholderExt(options.placeholder) : [],
				EditorView.contentAttributes.of({
					'aria-multiline': 'true',
					...(options.contentAttributes || {})
				}),
				EditorView.updateListener.of((update) => {
					if (update.docChanged) options.onChange(update.state.doc.toString())
				})
			]
		})
	})

	return {
		view,
		getValue: () => view.state.doc.toString(),
		setValue(value) {
			const current = view.state.doc.toString()
			if (value === current) return
			view.dispatch({ changes: { from: 0, to: current.length, insert: value } })
		},
		setReadOnly(readOnly) {
			view.dispatch({ effects: editableCompartment.reconfigure(editableExtensions(readOnly)) })
		},
		setDark(dark) {
			view.dispatch({ effects: themeCompartment.reconfigure(EditorView.theme({}, { dark: !!dark })) })
		},
		focus: () => view.focus(),
		destroy: () => view.destroy()
	}
}
