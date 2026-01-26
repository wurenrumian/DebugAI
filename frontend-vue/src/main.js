import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import axios from 'axios'

// Enable credentials for all requests
axios.defaults.withCredentials = true

// Add a request interceptor to include the JWT token
axios.interceptors.request.use(config => {
	const user = JSON.parse(localStorage.getItem('user'));
	if (user && user.token) {
		config.headers.Authorization = `Bearer ${user.token}`;
	}
	return config;
}, error => {
	return Promise.reject(error);
});

createApp(App).use(router).mount('#app')
