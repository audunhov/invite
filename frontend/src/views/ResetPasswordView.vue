<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { notify } from '../utils/toast'

const route = useRoute()
const router = useRouter()

const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref<string | null>(null)
const success = ref(false)
const token = ref('')

onMounted(() => {
  const t = route.query.token
  if (typeof t === 'string') {
    token.value = t
  } else {
    error.value = 'Invalid or missing reset token.'
    notify.error(error.value)
  }
})

async function resetPassword() {
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match.'
    notify.error(error.value)
    return
  }

  loading.value = true
  error.value = null
  try {
    const response = await fetch('/api/auth/reset-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        token: token.value,
        password: password.value,
      }),
    })

    if (!response.ok) {
      const errData = await response.json().catch(() => ({}))
      throw new Error(errData.message || 'Reset failed')
    }

    success.value = true
    notify.success('Password reset successful')
    setTimeout(() => {
      router.push('/login')
    }, 3000)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'An unexpected error occurred'
    notify.error(error.value)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-full flex-col justify-center py-12 sm:px-6 lg:px-8">
    <div class="sm:mx-auto sm:w-full sm:max-w-md">
      <h2 class="mt-6 text-center text-3xl font-bold tracking-tight text-gray-900 dark:text-white">Reset your password</h2>
      <p class="mt-2 text-center text-sm text-gray-600 dark:text-gray-400">
        Enter your new password below.
      </p>
    </div>

    <div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
      <div class="bg-white dark:bg-gray-800 py-8 px-4 shadow sm:rounded-lg sm:px-10 border border-gray-200 dark:border-white/10">
        <div v-if="success" class="rounded-md bg-green-50 dark:bg-green-900/30 p-4">
          <div class="flex">
            <div class="shrink-0">
              <svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4.13-5.682z" clip-rule="evenodd" />
              </svg>
            </div>
            <div class="ml-3">
              <p class="text-sm font-medium text-green-800 dark:text-green-300">Password reset successful! Redirecting to login...</p>
            </div>
          </div>
        </div>

        <form v-else @submit.prevent="resetPassword" class="space-y-6">
          <div v-if="!token && error" class="text-sm text-red-600 dark:text-red-400 mb-4">
            {{ error }}
          </div>
          
          <div v-if="token">
            <div>
              <label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">New Password</label>
              <div class="mt-1">
                <input
                  v-model="password"
                  id="password"
                  name="password"
                  type="password"
                  required
                  minlength="8"
                  class="block w-full appearance-none rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
                />
              </div>
            </div>

            <div class="mt-6">
              <label for="confirm-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Confirm New Password</label>
              <div class="mt-1">
                <input
                  v-model="confirmPassword"
                  id="confirm-password"
                  name="confirm-password"
                  type="password"
                  required
                  minlength="8"
                  class="block w-full appearance-none rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
                />
              </div>
            </div>

            <div v-if="error" class="mt-4 text-sm text-red-600 dark:text-red-400">
              {{ error }}
            </div>

            <div class="mt-6">
              <button
                type="submit"
                :disabled="loading"
                class="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50"
              >
                {{ loading ? 'Resetting...' : 'Reset password' }}
              </button>
            </div>
          </div>

          <div class="text-center mt-6">
            <RouterLink to="/login" class="text-sm font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400">
              Back to login
            </RouterLink>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
