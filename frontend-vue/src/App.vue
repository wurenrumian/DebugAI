<template>
  <component :is="layout" v-if="layout">
    <router-view />
  </component>
  <ToastContainer />
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ToastContainer from './components/ToastContainer.vue'
import AppLayout from './components/layout/AppLayout.vue'
import AuthLayout from './components/layout/AuthLayout.vue'

const route = useRoute()

// 需要登录的页面使用带导航的应用外壳，登录 / 注册等页面使用居中卡片布局
const layout = computed(() => {
  if (route.matched.length === 0) return null
  return route.meta && route.meta.requiresAuth ? AppLayout : AuthLayout
})
</script>
