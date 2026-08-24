import { createApp } from 'vue'
import PrimeVue from 'primevue/config'
import ConfirmationService from 'primevue/confirmationservice'
import Aura from '@primevue/themes/aura'
import 'primeicons/primeicons.css'

import App from './App.vue'
import router from './router'
import { loadMe } from './shared/store/auth'
import './styles/base.css'

async function bootstrap() {
  const app = createApp(App)
  app.use(router)
  app.use(PrimeVue, {
    theme: {
      preset: Aura,
      options: {
        darkModeSelector: '.dark',
      },
    },
  })
  app.use(ConfirmationService)
  await loadMe()
  app.mount('#app')
}

void bootstrap()
