<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useAuthStore } from '../stores/auth'
import type { components } from '../api-types'

type UpdatePerson = components['schemas']['UpdatePerson']

const auth = useAuthStore()
const isSaving = ref(false)
const message = ref<{ text: string; type: 'success' | 'error' } | null>(null)

const form = reactive({
  password: '',
  confirmPassword: '',
})

async function updatePassword() {
  if (!form.password) {
    message.value = { text: 'Password cannot be empty', type: 'error' }
    return
  }

  if (form.password !== form.confirmPassword) {
    message.value = { text: 'Passwords do not match', type: 'error' }
    return
  }

  if (!auth.user?.id) {
    message.value = { text: 'User session not found', type: 'error' }
    return
  }

  isSaving.value = true
  message.value = null
  try {
    const body: UpdatePerson = {
      password: form.password,
    }
    const response = await fetch(`/api/persons/${auth.user.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })

    if (!response.ok) {
      const errData = await response.json().catch(() => ({}))
      throw new Error(errData.message || 'Failed to update password')
    }

    message.value = { text: 'Password updated successfully', type: 'success' }
    form.password = ''
    form.confirmPassword = ''
  } catch (err) {
    message.value = { text: err instanceof Error ? err.message : 'Failed to update password', type: 'error' }
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <div class="max-w-2xl mx-auto">
    <div class="md:grid md:grid-cols-3 md:gap-6">
      <div class="md:col-span-1">
        <div class="px-4 sm:px-0">
          <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white">Security</h3>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Update your account password to stay secure.
          </p>
        </div>
      </div>
      <div class="mt-5 md:mt-0 md:col-span-2">
        <form @submit.prevent="updatePassword">
          <div class="shadow sm:rounded-md sm:overflow-hidden border border-gray-200 dark:border-white/10">
            <div class="px-4 py-5 bg-white dark:bg-gray-800 space-y-6 sm:p-6">
              <div v-if="message" :class="message.type === 'success' ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800' : 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-800'" class="p-4 rounded-md border text-sm">
                {{ message.text }}
              </div>

              <div class="grid grid-cols-6 gap-6">
                <div class="col-span-6 sm:col-span-4">
                  <label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">New Password</label>
                  <input
                    v-model="form.password"
                    type="password"
                    id="password"
                    class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                    required
                  />
                </div>

                <div class="col-span-6 sm:col-span-4">
                  <label for="confirm-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Confirm Password</label>
                  <input
                    v-model="form.confirmPassword"
                    type="password"
                    id="confirm-password"
                    class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                    required
                  />
                </div>
              </div>
            </div>
            <div class="px-4 py-3 bg-gray-50 dark:bg-gray-900/50 text-right sm:px-6 border-t dark:border-white/10">
              <button
                type="submit"
                :disabled="isSaving"
                class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
              >
                {{ isSaving ? 'Updating...' : 'Update Password' }}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
