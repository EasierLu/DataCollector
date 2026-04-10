import { createApp } from 'vue'
import { createPinia } from 'pinia'
import 'element-plus/dist/index.css'
import router from './router'
import App from './App.vue'
import './styles/index.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)


app.config.errorHandler = (err, _vm, info) => {
  console.error(`[Global Error] ${info}:`, err)
}

app.mount('#app')
