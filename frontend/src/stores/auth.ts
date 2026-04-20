import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { components } from '../api-types'

type Person = components['schemas']['Person']

export const useAuthStore = defineStore('auth', () => {
  const user = ref<Person | null>(null)
  const isAuthenticated = ref(false)
  const isInitialized = ref(false)

  async function checkAuth() {
    try {
      const response = await fetch('/api/auth/me')
      if (response.ok) {
        user.value = await response.json()
        isAuthenticated.value = true
      } else {
        user.value = null
        isAuthenticated.value = false
      }
    } catch (err) {
      user.value = null
      isAuthenticated.value = false
    } finally {
      isInitialized.value = true
    }
    return isAuthenticated.value
  }

  async function logout() {
    try {
      await fetch('/api/auth/logout', { method: 'POST' })
    } finally {
      user.value = null
      isAuthenticated.value = false
    }
  }

  return { user, isAuthenticated, isInitialized, checkAuth, logout }
})
