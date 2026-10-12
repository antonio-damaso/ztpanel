import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import ConfirmationService from 'primevue/confirmationservice'
import 'primeicons/primeicons.css'

import './style.css'
import App from './App.vue'
import router from './router'
import { ztpanelTheme } from './theme'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(PrimeVue, {
  theme: {
    preset: ztpanelTheme,
    options: { darkModeSelector: '.zp-dark' },
  },
})
app.use(ToastService)
app.use(ConfirmationService)

app.mount('#app')
