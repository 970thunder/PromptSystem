import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'

import './styles/main.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

// naive-ui 不做全局注册（app.use(naive) 会把整个组件库打进首屏包）：
// 各视图已在 <script setup> 中按需 import 所需组件，消息/对话框走
// NConfigProvider 下的 provider 与 request.ts 的 discrete API。
app.mount('#app')
