<template>
  <div
    :class="['code-editor', { 'is-disabled': disabled, 'is-ready': ready }]"
    :style="{ '--editor-min-height': minHeight, '--editor-max-height': maxHeight }"
  >
    <!-- CodeMirror 挂载点 -->
    <div v-show="ready" ref="hostRef" class="code-editor-host"></div>

    <!-- 编辑器加载完成前 / 加载失败时的纯文本回退，保证始终可输入 -->
    <textarea
      v-if="!ready"
      :id="id || undefined"
      class="code-editor-fallback"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :aria-labelledby="labelledby || undefined"
      :aria-describedby="describedby || undefined"
      spellcheck="false"
      autocapitalize="off"
      autocomplete="off"
      @input="onFallbackInput"
    ></textarea>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { themeState } from '../../utils/theme'

/**
 * C/C++ 代码编辑器（CodeMirror 6，按需加载）
 *  - v-model 双向绑定
 *  - 行号、Tab 缩进、括号匹配、语法高亮，配色随浅色 / 深色主题切换
 *  - 键盘用户：按 Esc 后再按 Tab 可把焦点移出编辑器
 */
const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  // 用于 <label for>（仅回退 textarea 使用）以及 aria 关联
  id: { type: String, default: '' },
  labelledby: { type: String, default: '' },
  describedby: { type: String, default: '' },
  minHeight: { type: String, default: '240px' },
  maxHeight: { type: String, default: '520px' }
})

const emit = defineEmits(['update:modelValue'])

const hostRef = ref(null)
const ready = ref(false)
let editor = null
let destroyed = false

const onFallbackInput = (event) => {
  emit('update:modelValue', event.target.value)
}

onMounted(async () => {
  try {
    const { createCodeEditor } = await import('./codemirror.js')
    if (destroyed || !hostRef.value) return
    const contentAttributes = {}
    if (props.labelledby) contentAttributes['aria-labelledby'] = props.labelledby
    if (props.describedby) contentAttributes['aria-describedby'] = props.describedby
    editor = createCodeEditor({
      parent: hostRef.value,
      doc: props.modelValue,
      placeholder: props.placeholder,
      readOnly: props.disabled,
      dark: themeState.resolved === 'dark',
      contentAttributes,
      onChange: (value) => {
        if (value !== props.modelValue) emit('update:modelValue', value)
      }
    })
    ready.value = true
  } catch (error) {
    // 加载失败（如网络中断）时保留 textarea 回退
    console.error('代码编辑器加载失败，已回退为普通文本框:', error)
  }
})

watch(() => props.modelValue, (value) => {
  if (editor) editor.setValue(value ?? '')
})

watch(() => props.disabled, (disabled) => {
  if (editor) editor.setReadOnly(disabled)
})

watch(() => themeState.resolved, (resolved) => {
  if (editor) editor.setDark(resolved === 'dark')
})

onBeforeUnmount(() => {
  destroyed = true
  if (editor) {
    editor.destroy()
    editor = null
  }
})

defineExpose({
  focus: () => editor && editor.focus()
})
</script>

<style scoped>
.code-editor {
  position: relative;
  overflow: hidden;
  background: var(--code-bg);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  transition: border-color var(--duration-fast) var(--ease), box-shadow var(--duration-fast) var(--ease);
}

.code-editor:hover:not(.is-disabled) {
  border-color: var(--color-text-muted);
}

.code-editor:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px var(--color-focus-ring);
}

.code-editor.is-disabled {
  border-color: var(--color-border);
  opacity: 0.85;
}

.code-editor-host {
  min-height: var(--editor-min-height);
  max-height: var(--editor-max-height);
  display: flex;
  flex-direction: column;
}

.code-editor-host :deep(.cm-editor) {
  flex: 1;
  min-height: var(--editor-min-height);
  max-height: var(--editor-max-height);
}

.code-editor-fallback {
  display: block;
  width: 100%;
  min-height: var(--editor-min-height);
  max-height: var(--editor-max-height);
  padding: 10px 12px;
  font-family: var(--font-mono);
  font-size: 13.5px;
  line-height: 1.65;
  color: var(--code-text);
  background: var(--code-bg);
  border: none;
  border-radius: 0;
  resize: vertical;
  tab-size: 4;
}

.code-editor-fallback:focus,
.code-editor-fallback:hover {
  border: none;
  box-shadow: none;
  outline: none;
}
</style>
