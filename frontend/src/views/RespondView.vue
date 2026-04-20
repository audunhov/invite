<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import type { components } from '../api-types'
import { notify } from '../utils/toast'

type PublicInviteDetails = components['schemas']['PublicInviteDetails']

const route = useRoute()
const token = route.params.token as string

const invite = ref<PublicInviteDetails | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const responded = ref(false)
const actionTaken = ref<'accepted' | 'declined' | null>(null)

async function fetchInvite() {
  try {
    const response = await fetch(`/api/respond/${token}`)
    if (!response.ok) {
      if (response.status === 404) throw new Error('Invite not found or link expired')
      throw new Error('Failed to load invite details')
    }
    invite.value = await response.json()
    if (invite.value?.current_state !== 'pending') {
      responded.value = true
      actionTaken.value = invite.value?.current_state === 'accepted' ? 'accepted' : 'declined'
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    loading.value = false
  }
}

async function respond(action: 'accept' | 'decline') {
  loading.value = true
  try {
    const response = await fetch(`/api/respond/${token}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action }),
    })

    if (!response.ok) throw new Error('Failed to record response')
    
    responded.value = true
    actionTaken.value = action === 'accept' ? 'accepted' : 'declined'
    notify.success(`Response recorded: ${action}`)
  } catch (err) {
    notify.error(err instanceof Error ? err.message : 'An unexpected error occurred')
  } finally {
    loading.value = false
  }
}

onMounted(fetchInvite)
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4">
    <div class="max-w-md w-full bg-white dark:bg-gray-800 rounded-xl shadow-2xl overflow-hidden border dark:border-white/10">
      <div v-if="loading && !invite" class="p-8 text-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4"></div>
        <p class="text-gray-500">Loading invite details...</p>
      </div>

      <div v-else-if="error" class="p-8 text-center">
        <div class="text-red-500 text-5xl mb-4">⚠️</div>
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-2">Oops!</h2>
        <p class="text-gray-600 dark:text-gray-400">{{ error }}</p>
      </div>

      <div v-else-if="invite" class="p-8">
        <div class="text-center mb-8">
          <div class="inline-flex items-center justify-center size-16 rounded-full bg-indigo-100 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400 mb-4">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="size-8">
              <path d="M21.75 6.75v10.5a2.25 2.25 0 0 1-2.25 2.25h-15a2.25 2.25 0 0 1-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0 0 19.5 4.5h-15a2.25 2.25 0 0 0-2.25 2.25m19.5 0v.243a2.25 2.25 0 0 1-1.07 1.916l-7.5 4.615a2.25 2.25 0 0 1-2.36 0L5.32 8.91a2.25 2.25 0 0 1-1.07-1.916V6.75" />
            </svg>
          </div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">You're Invited!</h1>
          <p class="text-indigo-600 dark:text-indigo-400 font-medium mt-1">{{ invite.title }}</p>
        </div>

        <div class="space-y-6">
          <div class="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 text-sm">
            <p class="text-gray-600 dark:text-gray-400 italic mb-4" v-if="invite.description">
              "{{ invite.description }}"
            </p>
            <div class="flex items-start gap-3">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="size-5 text-gray-400 shrink-0">
                <path d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5A2.25 2.25 0 0 1 5.25 9h13.5A2.25 2.25 0 0 1 21 11.25v7.5" />
              </svg>
              <div>
                <p class="font-bold text-gray-900 dark:text-white">When</p>
                <p class="text-gray-600 dark:text-gray-400">{{ new Date(invite.from).toLocaleString([], { dateStyle: 'full', timeStyle: 'short' }) }}</p>
              </div>
            </div>
          </div>

          <div v-if="responded" class="text-center p-6 bg-green-50 dark:bg-green-900/20 rounded-xl border border-green-100 dark:border-green-800">
            <div class="text-green-500 text-3xl mb-2">✨</div>
            <h3 class="text-lg font-bold text-green-800 dark:text-green-400">
              Response Recorded: {{ actionTaken }}
            </h3>
            <p class="text-sm text-green-700 dark:text-green-500/80 mt-1">Thank you for letting us know!</p>
          </div>

          <div v-else class="grid grid-cols-2 gap-4">
            <button
              @click="respond('decline')"
              :disabled="loading"
              class="px-6 py-3 rounded-lg border-2 border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 font-bold hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
            >
              Decline
            </button>
            <button
              @click="respond('accept')"
              :disabled="loading"
              class="px-6 py-3 rounded-lg bg-indigo-600 text-white font-bold hover:bg-indigo-700 shadow-lg shadow-indigo-200 dark:shadow-none transition-all transform hover:-translate-y-0.5 active:translate-y-0 disabled:opacity-50"
            >
              Accept
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
