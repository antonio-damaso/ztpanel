import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useSetupStore } from '../stores/setup'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/setup', name: 'setup', component: () => import('../views/SetupView.vue'), meta: { public: true } },
    { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
    { path: '/', name: 'networks', component: () => import('../views/NetworksView.vue') },
    { path: '/networks/:id', name: 'network-detail', component: () => import('../views/NetworkDetailView.vue'), props: true },
    { path: '/organizations/:id', name: 'organization', component: () => import('../views/OrganizationView.vue'), props: true },
    { path: '/users', name: 'users', component: () => import('../views/UsersView.vue') },
    { path: '/activity', name: 'activity', component: () => import('../views/ActivityView.vue') },
    { path: '/account', name: 'account', component: () => import('../views/AccountView.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach(async (to) => {
  const setup = useSetupStore()
  if (!setup.checked) {
    await setup.check()
  }
  if (setup.needed && to.name !== 'setup') {
    return { name: 'setup' }
  }
  if (!setup.needed && to.name === 'setup') {
    return { name: 'login' }
  }

  const auth = useAuthStore()
  if (!auth.checked) {
    await auth.checkSession()
  }

  if (!auth.isAuthenticated && !to.meta.public) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (auth.isAuthenticated && to.name === 'login') {
    return { name: 'networks' }
  }
  return true
})

export default router
