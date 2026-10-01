// 首屏渲染前应用已保存的主题偏好，避免深色 / 浅色闪烁（CSP 不允许内联脚本，故单独成文件）
// 与 src/utils/theme.js 中的 THEME_STORAGE_KEY 保持一致
;(function () {
	try {
		var value = localStorage.getItem('theme')
		if (value === 'light' || value === 'dark') {
			document.documentElement.setAttribute('data-theme', value)
		}
	} catch (e) {
		// 无法读取 localStorage 时跟随系统
	}
})()
