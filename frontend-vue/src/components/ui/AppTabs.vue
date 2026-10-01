<template>
  <div class="tabs">
    <div class="tab-list" role="tablist" :aria-label="label">
      <button
        v-for="(tab, index) in visibleTabs"
        :id="tabId(tab.key)"
        :key="tab.key"
        type="button"
        role="tab"
        :class="['tab', { active: tab.key === modelValue }]"
        :aria-selected="tab.key === modelValue ? 'true' : 'false'"
        :aria-controls="panelId"
        :tabindex="tab.key === modelValue ? 0 : -1"
        @click="select(tab.key)"
        @keydown="onKeydown($event, index)"
      >
        <AppIcon v-if="tab.icon" :name="tab.icon" :size="16" />
        <span>{{ tab.label }}</span>
        <kbd v-if="tab.shortcut" class="tab-shortcut">{{ tab.shortcut }}</kbd>
      </button>
    </div>
    <div
      :id="panelId"
      class="tab-panel"
      role="tabpanel"
      :aria-labelledby="tabId(modelValue)"
    >
      <slot />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AppIcon from './AppIcon.vue'
import { uid } from '../../utils/uid'

/**
 * 标签页（WAI-ARIA Tabs）：左右方向键 / Home / End 切换
 * tabs: [{ key, label, icon?, shortcut?, hidden? }]
 */
const props = defineProps({
  tabs: { type: Array, required: true },
  modelValue: { type: String, required: true },
  label: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue'])

const prefix = uid('tabs')
const panelId = `${prefix}-panel`

const visibleTabs = computed(() => props.tabs.filter(t => !t.hidden))

const tabId = (key) => `${prefix}-tab-${key}`

const select = (key) => {
  if (key !== props.modelValue) emit('update:modelValue', key)
}

const onKeydown = (event, index) => {
  const count = visibleTabs.value.length
  let next = -1
  if (event.key === 'ArrowRight') next = (index + 1) % count
  else if (event.key === 'ArrowLeft') next = (index - 1 + count) % count
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = count - 1
  if (next === -1) return
  event.preventDefault()
  const key = visibleTabs.value[next].key
  select(key)
  const el = document.getElementById(tabId(key))
  if (el) el.focus()
}
</script>

<style scoped>
.tab-list {
  display: flex;
  gap: var(--space-1);
  overflow-x: auto;
  scrollbar-width: none;
  border-bottom: 1px solid var(--color-border);
}

.tab-list::-webkit-scrollbar {
  display: none;
}

.tab {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  padding: 10px 14px;
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--color-text-muted);
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  transition: color var(--duration-fast) var(--ease), background-color var(--duration-fast) var(--ease);
}

.tab::after {
  content: '';
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: -1px;
  height: 2px;
  border-radius: 2px;
  background: transparent;
}

.tab:hover {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

.tab.active {
  color: var(--color-primary-text);
}

.tab.active::after {
  background: var(--color-primary);
}

.tab:focus-visible {
  outline-offset: -2px;
}

.tab-shortcut {
  margin-left: 2px;
}

.tab-panel {
  padding-top: var(--space-5);
}

.tab-panel:focus-visible {
  outline-offset: 4px;
}

@media (max-width: 900px) {
  .tab-shortcut {
    display: none;
  }
}

@media (max-width: 640px) {
  .tab {
    padding: 10px 12px;
  }

  .tab-panel {
    padding-top: var(--space-4);
  }
}
</style>
