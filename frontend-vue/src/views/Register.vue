<template>
  <div class="register-container">
    <h2>注册</h2>
    <form @submit.prevent="handleRegister">
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
        <label for="username">用户名:</label>
        <input
          v-model="form.username"
          type="text"
          id="username"
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
      <div class="form-group">
        <label for="userType">用户类型:</label>
        <select v-model="form.userType" id="userType">
          <option value="student">学生</option>
          <option value="admin">管理员</option>
        </select>
      </div>
      <button type="submit" :disabled="loading">注册</button>
    </form>
    <p v-if="error" class="error">{{ error }}</p>
    <p>已有账号？<router-link to="/login">登录</router-link></p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const router = useRouter()
const form = ref({
  studentId: '',
  username: '',
  password: '',
  userType: 'student'
})
const error = ref('')
const loading = ref(false)

const handleRegister = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await axios.post('http://localhost:8080/auth/register', {
      student_id: form.value.studentId,
      username: form.value.username,
      password: form.value.password,
      user_type: form.value.userType
    })
    if (response.data.message === '注册成功') {
      router.push('/login')
    }
  } catch (err) {
    error.value = err.response?.data?.error || '注册失败'
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
.register-container {
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
input, select {
  width: 100%;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
}
button {
  width: 100%;
  padding: 10px;
  background-color: #28a745;
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
