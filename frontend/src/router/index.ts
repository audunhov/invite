import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
    },
    {
      path: '/persons',
      name: 'persons',
      component: () => import('../views/PersonsView.vue'),
    },
    {
      path: '/invites',
      name: 'invites',
      component: () => import('../views/InvitesView.vue'),
    },
    {
      path: '/groups',
      name: 'groups',
      component: () => import('../views/GroupsView.vue'),
    },
  ],
})

export default router
