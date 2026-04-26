import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { components } from '../api-types'
import { client } from '../utils/api'

type Person = components['schemas']['Person']

export const useAuthStore = defineStore('auth', () => {
  const user = ref<Person | null>(null)
  const isAuthenticated = ref(false)
  const isInitialized = ref(false)

  async function checkAuth() {
    try {
      const { data, error } = await client.GET('/auth/me')
      if (!error && data) {
        user.value = data
        isAuthenticated.value = true
      } else {
        user.value = null
        isAuthenticated.value = false
      }
    } catch {
      user.value = null
      isAuthenticated.value = false
    } finally {
      isInitialized.value = true
    }
    return isAuthenticated.value
  }

  async function login(credentials: Record<string, string>) {
    if (!credentials.email || !credentials.password) {
      throw new Error('Email and password are required')
    }

    const { error, response } = await client.POST('/auth/login', {
      body: {
        email: credentials.email,
        password: credentials.password,
      },
    })

    if (error) {
      if (response.status === 401) {
        throw new Error('Invalid email or password')
      }
      const errData = error as any
      throw new Error(errData.message || 'Login failed')
    }

    // Success - update state
    await checkAuth()
  }

  async function logout() {
    try {
      await client.POST('/auth/logout')
    } finally {
      user.value = null
      isAuthenticated.value = false
    }
  }

  return { user, isAuthenticated, isInitialized, checkAuth, login, logout }
})
