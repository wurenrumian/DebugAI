<template>
  <div id="app">
    <header>
      <nav>
        <h1>AI 教学辅助平台</h1>
        <router-link v-if="$route.path !== '/profile' && !isLoggedIn" to="/login">登录</router-link>
        <router-link v-if="$route.path !== '/register' && !isLoggedIn" to="/register">注册</router-link>
        <router-link v-if="$route.path !== '/profile' && isLoggedIn" to="/profile">个人主页</router-link>
      </nav>
    </header>
    <main>
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { computed } from 'vue';

const isLoggedIn = computed(() => !!localStorage.getItem('user'));
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
}

header {
  background: white;
  box-shadow: 0 2px 4px rgba(0,0,0,.1);
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60px; /* Slightly taller header */
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: space-between; /* Space out items */
  padding: 0 20px; /* Add horizontal padding */
}

header h1 {
  margin: 0; /* Remove default margin from h1 */
  color: #007bff; /* Highlight title color */
  font-size: 24px;
}

nav {
  display: flex;
  gap: 25px; /* Slightly more space between links */
}

nav a {
  text-decoration: none;
  color: #2c3e50;
  padding: 10px 15px;
  font-weight: bold; /* Make links bolder */
  transition: color 0.3s ease; /* Smooth transition for hover */
}

nav a:hover {
  color: #0056b3; /* Darker blue on hover */
}

main {
  padding: 20px;
}
</style>
