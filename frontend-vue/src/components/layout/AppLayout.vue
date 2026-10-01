<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">跳到主要内容</a>

    <header class="app-header">
      <div class="app-header-inner">
        <button
          type="button"
          class="nav-toggle"
          :aria-expanded="mobileOpen ? 'true' : 'false'"
          aria-controls="mobile-nav"
          :aria-label="mobileOpen ? '关闭导航菜单' : '打开导航菜单'"
          @click="mobileOpen = !mobileOpen"
        >
          <AppIcon :name="mobileOpen ? 'close' : 'menu'" :size="20" />
        </button>

        <BrandMark to="/profile" class="app-brand" />

        <nav class="primary-nav" aria-label="主导航">
          <RouterLink
            v-for="item in NAV_ITEMS"
            :key="item.to"
            :to="item.to"
            class="nav-link"
            :class="{ active: isActive(item.to) }"
            :aria-current="isActive(item.to) ? 'page' : undefined"
          >
            <AppIcon :name="item.icon" :size="16" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </nav>

        <div class="header-actions">
          <ThemeToggle />
          <UserMenu />
        </div>
      </div>

      <Transition name="drawer">
        <nav
          v-if="mobileOpen"
          id="mobile-nav"
          class="mobile-nav"
          aria-label="主导航"
        >
          <RouterLink
            to="/profile"
            class="mobile-link"
            :class="{ active: route.path === '/profile' }"
            :aria-current="route.path === '/profile' ? 'page' : undefined"
          >
            <AppIcon name="home" :size="18" />
            <span>个人主页</span>
          </RouterLink>
          <RouterLink
            v-for="item in NAV_ITEMS"
            :key="item.to"
            :to="item.to"
            class="mobile-link"
            :class="{ active: isActive(item.to) }"
            :aria-current="isActive(item.to) ? 'page' : undefined"
          >
            <AppIcon :name="item.icon" :size="18" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </nav>
      </Transition>
    </header>

    <div v-if="mobileOpen" class="mobile-backdrop" aria-hidden="true" @click="mobileOpen = false"></div>

    <main id="main-content" class="app-main" tabindex="-1">
      <div class="app-main-inner">
        <slot />
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import AppIcon from '../ui/AppIcon.vue'
import ThemeToggle from '../ui/ThemeToggle.vue'
import BrandMark from './BrandMark.vue'
import UserMenu from './UserMenu.vue'
import { NAV_ITEMS } from './navigation'

const route = useRoute()
const mobileOpen = ref(false)

const isActive = (path) => route.path === path || route.path.startsWith(`${path}/`)

watch(() => route.fullPath, () => {
  mobileOpen.value = false
})

const onKeydown = (event) => {
  if (mobileOpen.value && event.key === 'Escape') {
    mobileOpen.value = false
  }
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.skip-link {
  position: absolute;
  top: -100px;
  left: var(--space-4);
  z-index: var(--z-toast);
  padding: var(--space-2) var(--space-4);
  color: var(--color-on-primary);
  background: var(--color-primary);
  border-radius: var(--radius-md);
}

.skip-link:focus {
  top: var(--space-2);
}

.app-header {
  position: sticky;
  top: 0;
  z-index: var(--z-nav);
  background: var(--color-surface);
  background: color-mix(in srgb, var(--color-surface) 92%, transparent);
  border-bottom: 1px solid var(--color-border);
  backdrop-filter: saturate(1.4) blur(10px);
  -webkit-backdrop-filter: saturate(1.4) blur(10px);
}

.app-header-inner {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  max-width: var(--layout-max);
  height: var(--nav-height);
  margin: 0 auto;
  padding: 0 var(--space-6);
}

.nav-toggle {
  display: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  margin-left: -6px;
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
}

.nav-toggle:hover {
  background: var(--color-surface-hover);
}

.primary-nav {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.nav-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 12px;
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  white-space: nowrap;
  transition: background-color var(--duration-fast) var(--ease), color var(--duration-fast) var(--ease);
}

.nav-link:hover {
  color: var(--color-text);
  text-decoration: none;
  background: var(--color-surface-hover);
}

.nav-link.active {
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  margin-left: auto;
}

.mobile-nav {
  display: none;
}

.mobile-backdrop {
  display: none;
}

.app-main {
  flex: 1;
  outline: none;
}

.app-main-inner {
  max-width: var(--layout-max);
  margin: 0 auto;
  padding: var(--space-8) var(--space-6) var(--space-12);
}

@media (max-width: 1080px) {
  .app-header-inner {
    gap: var(--space-4);
  }

  .nav-link {
    padding: 0 10px;
  }

  .nav-link .app-icon {
    display: none;
  }
}

@media (max-width: 900px) {
  .nav-toggle {
    display: inline-flex;
  }

  .primary-nav {
    display: none;
  }

  .app-header-inner {
    gap: var(--space-2);
    padding: 0 var(--space-4);
  }

  .mobile-nav {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    z-index: var(--z-drawer);
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--space-2) var(--space-3) var(--space-4);
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border);
    box-shadow: var(--shadow-md);
  }

  .mobile-link {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: 12px var(--space-3);
    font-size: var(--fs-md);
    font-weight: var(--fw-medium);
    color: var(--color-text-secondary);
    border-radius: var(--radius-md);
  }

  .mobile-link:hover {
    text-decoration: none;
    background: var(--color-surface-hover);
  }

  .mobile-link.active {
    color: var(--color-primary-text);
    background: var(--color-primary-soft);
  }

  .mobile-backdrop {
    display: block;
    position: fixed;
    inset: var(--nav-height) 0 0 0;
    z-index: calc(var(--z-nav) - 1);
    background: var(--color-overlay);
  }

  .app-main-inner {
    padding: var(--space-6) var(--space-4) var(--space-10);
  }
}

@media (max-width: 640px) {
  .app-main-inner {
    padding: var(--space-5) var(--space-4) var(--space-10);
  }
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity var(--duration) var(--ease), transform var(--duration) var(--ease);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
