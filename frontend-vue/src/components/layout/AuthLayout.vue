<template>
  <div class="auth-shell">
    <header class="auth-topbar">
      <BrandMark tagline />
      <ThemeToggle />
    </header>
    <main id="main-content" class="auth-main">
      <div class="auth-card">
        <slot />
      </div>
    </main>
    <footer class="auth-bottom">AI 驱动的编程教学辅助 · 调试 · 评价 · 练习推荐</footer>
  </div>
</template>

<script setup>
import BrandMark from './BrandMark.vue'
import ThemeToggle from '../ui/ThemeToggle.vue'
</script>

<style scoped>
.auth-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background:
    radial-gradient(1200px 520px at 15% -10%, var(--color-primary-soft), transparent 60%),
    radial-gradient(900px 480px at 110% 110%, var(--color-primary-soft), transparent 60%),
    var(--color-bg);
}

.auth-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  max-width: var(--layout-max);
  margin: 0 auto;
  padding: var(--space-5) var(--space-6);
}

.auth-main {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-4) var(--space-4) var(--space-8);
}

.auth-card {
  width: 100%;
  max-width: 420px;
  padding: var(--space-8);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
}

.auth-bottom {
  padding: var(--space-5) var(--space-4);
  text-align: center;
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}

@media (max-width: 640px) {
  .auth-topbar {
    padding: var(--space-4);
  }

  .auth-main {
    align-items: flex-start;
    padding-top: var(--space-2);
  }

  .auth-card {
    padding: var(--space-6) var(--space-5);
  }
}
</style>

<style>
/* 认证页（登录 / 注册 / 找回密码 / 重置密码）共享的内容样式 */
.auth-heading {
  margin-bottom: var(--space-6);
}

.auth-title {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-semibold);
}

.auth-subtitle {
  margin-top: var(--space-1);
  font-size: var(--fs-base);
  color: var(--color-text-muted);
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.auth-form .field + .field {
  margin-top: 0;
}

.auth-row-end {
  display: flex;
  justify-content: flex-end;
  margin-top: calc(var(--space-2) * -1);
  font-size: var(--fs-sm);
}

.auth-footer {
  margin-top: var(--space-6);
  padding-top: var(--space-5);
  border-top: 1px solid var(--color-border);
  text-align: center;
  font-size: var(--fs-base);
  color: var(--color-text-muted);
}

.auth-footer a {
  margin-left: 4px;
  font-weight: var(--fw-medium);
}
</style>
