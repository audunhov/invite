import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
      meta: { requiresAuth: true },
    },
    {
      path: '/persons',
      name: 'persons',
      component: () => import('../views/PersonsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/invites',
      name: 'invites',
      component: () => import('../views/InvitesView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/groups',
      name: 'groups',
      component: () => import('../views/GroupsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/SettingsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('../views/ForgotPasswordView.vue'),
    },
    {
      path: '/reset-password',
      name: 'reset-password',
      component: () => import('../views/ResetPasswordView.vue'),
    },
    {
      path: '/respond/:token',
      name: 'respond',
      component: () => import('../views/RespondView.vue'),
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  
  // Only check auth if we haven't already and we're not logged in
  if (!auth.isAuthenticated && !auth.isInitialized) {
    await auth.checkAuth()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login' }
  }

  // Redirect to dashboard if trying to access login/forgot/reset while already authenticated
  const guestRoutes = ['login', 'forgot-password', 'reset-password']
  if (guestRoutes.includes(to.name as string) && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
})

export default router
