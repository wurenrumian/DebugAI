<template>
  <div class="profile-container">
    <h2>个人主页</h2>
    <div class="user-info">
      <p><strong>学号:</strong> {{ user.student_id }}</p>
      <p><strong>用户名:</strong> {{ user.username }}</p>
      <p><strong>用户类型:</strong> {{ user.user_type }}</p>
    </div>
    <button @click="logout">退出登录</button>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const router = useRouter()
const user = ref({})
const loading = ref(true)
const error = ref('')

const logout = async () => {
  try {
    await axios.post('http://localhost:8080/auth/logout')
  } catch (err) {
    console.error('Logout error:', err)
  }
  localStorage.removeItem('user')
  router.push('/login')
}

const fetchProfile = async () => {
  try {
    const response = await axios.get('http://localhost:8080/api/v1/profile')
    user.value = response.data.data
    localStorage.setItem('user', JSON.stringify(user.value))
  } catch (err) {
    error.value = err.response?.data?.error || '获取信息失败'
    if (err.response?.status === 401) {
      localStorage.removeItem('user')
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchProfile()
})
</script>

<style scoped>
.profile-container {
  max-width: 400px;
  margin: 50px auto;
  padding: 20px;
  border: 1px solid #ddd;
  border-radius: 8px;
  text-align: center;
}
.user-info p {
  margin: 10px 0;
}
button {
  padding: 10px 20px;
  background-color: #dc3545;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
</style>
