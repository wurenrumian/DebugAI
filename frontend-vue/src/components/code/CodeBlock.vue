<template>
  <!-- 经 renderMarkdown（DOMPurify 过滤）后输出，代码中的 HTML 会被转义为文本 -->
  <div
    class="code-block markdown-body"
    :style="maxHeight ? { '--code-max-height': maxHeight } : undefined"
    v-html="html"
  ></div>
</template>

<script setup>
import { computed } from 'vue'
import { renderCodeBlock } from '../../utils/codeBlock'

/** 只读代码展示（highlight.js 高亮，与 Markdown 代码块同一套配色） */
const props = defineProps({
  code: { type: String, default: '' },
  lang: { type: String, default: 'cpp' },
  maxHeight: { type: String, default: '360px' }
})

const html = computed(() => renderCodeBlock(props.code, props.lang))
</script>

<style scoped>
.code-block :deep(pre) {
  margin: 0;
  max-height: var(--code-max-height, 360px);
  overflow: auto;
}
</style>
