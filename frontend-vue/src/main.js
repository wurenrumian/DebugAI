import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import axios from 'axios'

// Enable credentials for all requests
axios.defaults.withCredentials = true

createApp(App).use(router).mount('#app')
