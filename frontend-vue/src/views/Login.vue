<template>
  <div class="login-container">
    <h2>登录</h2>
    <form @submit.prevent="handleLogin">
      <div class="form-group">
        <label for="studentId">学号:</label>
        <input
          v-model="form.studentId"
          type="text"
          id="studentId"
          required
        />
      </div>
      <div class="form-group">
        <label for="password">密码:</label>
        <input
          v-model="form.password"
          type="password"
          id="password"
          required
        />
      </div>
      <button type="submit" :disabled="loading">登录</button>
    </form>
    <p v-if="error" class="error">{{ error }}</p>
    <p>没有账号？<router-link to="/register">注册</router-link></p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const router = useRouter()
const form = ref({
  studentId: '',
  password: ''
})
const error = ref('')
const loading = ref(false)

const handleLogin = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await axios.post('http://localhost:8080/auth/login', {
      student_id: form.value.studentId,
      password: form.value.password
    })
    const { token, username, user_type, student_id } = response.data.data
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify({ username, user_type, student_id }))
    router.push('/profile')
  } catch (err) {
    error.value = err.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (localStorage.getItem('user')) {
    router.push('/profile')
  }
})
</script>

<style scoped>
.login-container {
  max-width: 400px;
  margin: 50px auto;
  padding: 20px;
  border: 1px solid #ddd;
  border-radius: 8px;
}
.form-group {
  margin-bottom: 15px;
}
label {
  display: block;
  margin-bottom: 5px;
}
input {
  width: 100%;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
}
button {
  width: 100%;
  padding: 10px;
  background-color: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
button:disabled {
  background-color: #ccc;
}
.error {
  color: red;
}
</style>
