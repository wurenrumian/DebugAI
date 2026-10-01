<template>
  <div ref="rootRef" class="user-menu">
    <button
      type="button"
      class="user-trigger"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-controls="menuId"
      :aria-label="`用户菜单：${displayName}`"
      @click="open = !open"
    >
      <span class="avatar" aria-hidden="true">{{ initial }}</span>
      <span class="user-name">{{ displayName }}</span>
      <AppIcon name="chevron-down" :size="16" class="chevron" />
    </button>

    <Transition name="pop">
      <div v-if="open" :id="menuId" class="user-dropdown">
        <div class="dropdown-head">
          <span class="avatar avatar-lg" aria-hidden="true">{{ initial }}</span>
          <div class="dropdown-identity">
            <p class="dropdown-name">{{ displayName }}</p>
            <p class="dropdown-meta">{{ roleLabel }}<template v-if="studentId"> · {{ studentId }}</template></p>
          </div>
        </div>
        <ul class="dropdown-list">
          <li>
            <RouterLink to="/profile" class="dropdown-item" @click="open = false">
              <AppIcon name="user" :size="16" />个人主页
            </RouterLink>
          </li>
          <li v-if="themeState.preference !== 'system'">
            <button type="button" class="dropdown-item" @click="followSystem">
              <AppIcon name="monitor" :size="16" />主题跟随系统
            </button>
          </li>
          <li>
            <button type="button" class="dropdown-item danger" @click="logout">
              <AppIcon name="logout" :size="16" />退出登录
            </button>
          </li>
        </ul>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { themeState, setThemePreference } from '../../utils/theme'
import AppIcon from '../ui/AppIcon.vue'
import { uid } from '../../utils/uid'

const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()

const open = ref(false)
const rootRef = ref(null)
const menuId = uid('user-menu')

const displayName = computed(() => authStore.user?.username || '用户')
const studentId = computed(() => authStore.user?.student_id || '')
const initial = computed(() => displayName.value.charAt(0).toUpperCase())
const roleLabel = computed(() => (authStore.role === 'admin' ? '管理员' : '学生'))

const followSystem = () => {
  setThemePreference('system')
  open.value = false
}

const logout = async () => {
  open.value = false
  await authStore.logout()
  router.push('/login')
}

const onDocumentClick = (event) => {
  if (open.value && rootRef.value && !rootRef.value.contains(event.target)) {
    open.value = false
  }
}

const onKeydown = (event) => {
  if (open.value && event.key === 'Escape') {
    open.value = false
    const trigger = rootRef.value && rootRef.value.querySelector('.user-trigger')
    if (trigger) trigger.focus()
  }
}

watch(() => route.fullPath, () => {
  open.value = false
})

onMounted(() => {
  document.addEventListener('click', onDocumentClick)
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.user-menu {
  position: relative;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  height: 36px;
  padding: 0 6px 0 4px;
  border-radius: var(--radius-full);
  color: var(--color-text-secondary);
  transition: background-color var(--duration-fast) var(--ease);
}

.user-trigger:hover,
.user-trigger[aria-expanded='true'] {
  background: var(--color-surface-hover);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border-radius: var(--radius-full);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
  border: 1px solid var(--color-primary-soft-border);
}

.avatar-lg {
  width: 40px;
  height: 40px;
  font-size: var(--fs-lg);
}

.user-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--color-text);
}

.chevron {
  color: var(--color-text-muted);
}

.user-dropdown {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  z-index: var(--z-dropdown);
  width: 240px;
  padding: var(--space-2);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
}

.dropdown-head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-2) var(--space-3);
  margin-bottom: var(--space-1);
  border-bottom: 1px solid var(--color-border);
}

.dropdown-identity {
  min-width: 0;
}

.dropdown-name {
  font-weight: var(--fw-semibold);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-meta {
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  font-size: var(--fs-base);
  color: var(--color-text-secondary);
  text-align: left;
  border-radius: var(--radius-sm);
}

.dropdown-item:hover {
  color: var(--color-text);
  text-decoration: none;
  background: var(--color-surface-hover);
}

.dropdown-item.danger {
  color: var(--color-danger);
}

.dropdown-item.danger:hover {
  background: var(--color-danger-soft);
}

.pop-enter-active,
.pop-leave-active {
  transition: opacity var(--duration-fast) var(--ease), transform var(--duration-fast) var(--ease);
}

.pop-enter-from,
.pop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

@media (max-width: 640px) {
  .user-name,
  .chevron {
    display: none;
  }

  .user-trigger {
    padding: 0 4px;
  }
}
</style>
