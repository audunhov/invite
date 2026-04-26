<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import type { components } from '../api-types'
import { client } from '../utils/api'
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
    const { data, error: err, response } = await client.GET('/respond/{token}', {
      params: { path: { token } }
    })
    
    if (err) {
      if (response.status === 404) throw new Error('Invite not found or link expired')
      throw new Error('Failed to load invite details')
    }
    
    if (data) {
      invite.value = data
      if (invite.value.current_state !== 'pending') {
        responded.value = true
        actionTaken.value = invite.value.current_state === 'accepted' ? 'accepted' : 'declined'
      }
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
    const { error: err } = await client.POST('/respond/{token}', {
      params: { path: { token } },
      body: { action }
    })

    if (err) throw err
    
    responded.value = true
    actionTaken.value = action === 'accept' ? 'accepted' : 'declined'
    notify.success(`Response recorded: ${action}`)
  } catch (err) {
    // Middleware handles toasts
  } finally {
    loading.value = false
  }
}

onMounted(fetchInvite)
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4 bg-gray-50 dark:bg-gray-950 transition-colors">
    <!-- Background Decoration -->
    <div class="fixed inset-0 overflow-hidden pointer-events-none opacity-20 dark:opacity-10">
      <div class="absolute -top-[10%] -left-[10%] size-[50%] bg-indigo-500 rounded-full blur-[120px]"></div>
      <div class="absolute -bottom-[10%] -right-[10%] size-[50%] bg-emerald-500 rounded-full blur-[120px]"></div>
    </div>

    <div class="max-w-2xl w-full bg-white dark:bg-gray-900 rounded-2xl shadow-[0_20px_50px_rgba(0,0,0,0.1)] dark:shadow-[0_20px_50px_rgba(0,0,0,0.3)] overflow-hidden border dark:border-white/5 relative z-10">
      <div v-if="loading && !invite" class="p-20 text-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4"></div>
        <p class="text-gray-500 font-medium animate-pulse">Opening your invitation...</p>
      </div>

      <div v-else-if="error" class="p-16 text-center">
        <div class="inline-flex items-center justify-center size-20 rounded-full bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 mb-6">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="size-10"><path d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" /></svg>
        </div>
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Invitation Error</h2>
        <p class="text-gray-600 dark:text-gray-400">{{ error }}</p>
      </div>

      <div v-else-if="invite" class="relative">
        <!-- Top Accent Bar -->
        <div class="h-2 bg-gradient-to-r from-indigo-500 via-purple-500 to-emerald-500"></div>
        
        <div class="p-8 sm:p-12">
          <div class="text-center mb-10">
            <p class="text-indigo-600 dark:text-indigo-400 font-bold uppercase tracking-[0.2em] text-xs mb-3">You are invited by</p>
            <h1 class="text-3xl sm:text-4xl font-extrabold text-gray-900 dark:text-white tracking-tight">{{ invite.sender_name }}</h1>
            <div class="mt-6 inline-block h-px w-20 bg-gray-200 dark:bg-gray-800"></div>
          </div>

          <div class="space-y-10">
            <!-- Event Title & Message -->
            <div class="text-center space-y-4">
              <h2 class="text-xl sm:text-2xl font-medium text-gray-800 dark:text-gray-200 italic font-serif leading-relaxed">
                "{{ invite.title }}"
              </h2>
              <p v-if="invite.description" class="text-gray-600 dark:text-gray-400 max-w-lg mx-auto leading-relaxed">
                {{ invite.description }}
              </p>
            </div>

            <!-- Details Grid -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 bg-gray-50 dark:bg-white/5 rounded-2xl p-6 border dark:border-white/5">
              <div class="flex items-center gap-4">
                <div class="size-10 rounded-full bg-white dark:bg-gray-800 shadow-sm flex items-center justify-center text-gray-400">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="size-5"><path d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5" /></svg>
                </div>
                <div>
                  <p class="text-[10px] font-bold text-gray-400 uppercase tracking-wider">Date</p>
                  <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ new Date(invite.from).toLocaleDateString([], { dateStyle: 'full' }) }}</p>
                </div>
              </div>
              <div class="flex items-center gap-4">
                <div class="size-10 rounded-full bg-white dark:bg-gray-800 shadow-sm flex items-center justify-center text-gray-400">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="size-5"><path d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" /></svg>
                </div>
                <div>
                  <p class="text-[10px] font-bold text-gray-400 uppercase tracking-wider">Time</p>
                  <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ new Date(invite.from).toLocaleTimeString([], { timeStyle: 'short' }) }}</p>
                </div>
              </div>
            </div>

            <!-- Response Section -->
            <div v-if="responded" class="text-center animate-in fade-in zoom-in duration-500">
              <div class="inline-flex items-center justify-center size-16 rounded-full" 
                :class="actionTaken === 'accepted' ? 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600' : 'bg-gray-100 dark:bg-gray-800 text-gray-500'">
                <svg v-if="actionTaken === 'accepted'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" class="size-8"><path d="m4.5 12.75 6 6 9-13.5" /></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" class="size-8"><path d="M6 18 18 6M6 6l12 12" /></svg>
              </div>
              <h3 class="mt-6 text-2xl font-bold text-gray-900 dark:text-white">
                {{ actionTaken === 'accepted' ? "See you there!" : "Maybe next time." }}
              </h3>
              <p class="text-gray-500 dark:text-gray-400 mt-2">Your response has been recorded.</p>
            </div>

            <div v-else class="flex flex-col sm:flex-row gap-4 pt-4">
              <button
                @click="respond('decline')"
                :disabled="loading"
                class="flex-1 px-8 py-4 rounded-xl border-2 border-gray-100 dark:border-gray-800 text-gray-500 dark:text-gray-400 font-bold hover:bg-gray-50 dark:hover:bg-gray-800 transition-all disabled:opacity-50"
              >
                Decline
              </button>
              <button
                @click="respond('accept')"
                :disabled="loading"
                class="flex-[2] px-8 py-4 rounded-xl bg-indigo-600 text-white font-bold shadow-[0_10px_20px_rgba(79,70,229,0.3)] hover:bg-indigo-700 hover:shadow-[0_15px_25px_rgba(79,70,229,0.4)] transition-all transform hover:-translate-y-1 active:translate-y-0 disabled:opacity-50"
              >
                Accept Invitation
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
