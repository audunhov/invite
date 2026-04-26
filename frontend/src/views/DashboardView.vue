<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import type { components } from '../api-types'
import { client } from '../utils/api'
import { notify } from '../utils/toast'
import TableSkeleton from '../components/TableSkeleton.vue'

type DashboardStats = components['schemas']['DashboardStats']

const router = useRouter()
const stats = ref<DashboardStats | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
let refreshInterval: number | null = null

async function fetchStats() {
  try {
    const { data, error: apiError, response } = await client.GET('/dashboard/stats')
    if (response.status === 401) {
      router.push('/login')
      return
    }
    if (apiError || !data) {
      error.value = 'Failed to fetch dashboard stats'
      return
    }
    stats.value = data
  } catch (err) {
    console.error(err)
    error.value = err instanceof Error ? err.message : 'Unknown error'
    notify.error(error.value)
  } finally {
    loading.value = false
  }
}

function formatTimeAgo(dateString: string) {
  const date = new Date(dateString)
  const now = new Date()
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000)
  
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return date.toLocaleDateString()
}

onMounted(() => {
  fetchStats()
  refreshInterval = window.setInterval(fetchStats, 30000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

function goToInvites() {
  router.push('/invites')
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h2>
      <div v-if="loading && stats" class="animate-spin h-5 w-5 border-2 border-indigo-500 border-t-transparent rounded-full"></div>
    </div>

    <!-- Top Row (Stats) -->
    <div v-if="loading && !stats" class="grid grid-cols-1 gap-5 sm:grid-cols-3">
      <div v-for="i in 3" :key="i" class="h-24 bg-gray-100 dark:bg-white/5 animate-pulse rounded-lg border border-gray-200 dark:border-white/10"></div>
    </div>
    <div v-else-if="stats" class="grid grid-cols-1 gap-5 sm:grid-cols-3">
      <!-- Active Invites -->
      <div class="bg-white dark:bg-white/5 overflow-hidden shadow rounded-lg border border-gray-200 dark:border-white/10">
        <div class="px-4 py-5 sm:p-6">
          <dt class="text-sm font-medium text-gray-500 dark:text-gray-400 truncate">Active Invites</dt>
          <dd class="mt-1 text-3xl font-semibold text-gray-900 dark:text-white">{{ stats.stats.active_invites }}</dd>
        </div>
      </div>
      <!-- Success Rate -->
      <div class="bg-white dark:bg-white/5 overflow-hidden shadow rounded-lg border border-gray-200 dark:border-white/10">
        <div class="px-4 py-5 sm:p-6">
          <dt class="text-sm font-medium text-gray-500 dark:text-gray-400 truncate">Success Rate</dt>
          <dd class="mt-1 text-3xl font-semibold text-gray-900 dark:text-white">{{ stats.stats.success_rate?.toFixed(1) || '0.0' }}%</dd>
        </div>
      </div>
      <!-- Failed Emails -->
      <div class="bg-white dark:bg-white/5 overflow-hidden shadow rounded-lg border border-gray-200 dark:border-white/10">
        <div class="px-4 py-5 sm:p-6">
          <dt class="text-sm font-medium text-gray-500 dark:text-gray-400 truncate">Failed Emails</dt>
          <dd class="mt-1 text-3xl font-semibold text-red-600 dark:text-red-400">{{ stats.stats.failed_emails }}</dd>
        </div>
      </div>
    </div>

    <!-- Process Timeline -->
    <div v-if="stats && stats.timeline && stats.timeline.length > 0" class="space-y-4">
      <h3 class="text-lg font-medium text-gray-900 dark:text-white">Process Timeline</h3>
      <div class="bg-white dark:bg-white/5 shadow rounded-lg border border-gray-200 dark:border-white/10 overflow-hidden">
        <div class="p-6 overflow-x-auto">
          <div class="min-w-[600px] space-y-6">
            <div v-for="invite in stats.timeline" :key="invite.id" class="group">
              <div class="flex justify-between items-end mb-2">
                <span class="text-sm font-bold text-gray-700 dark:text-gray-300 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 cursor-pointer" @click="goToInvites">
                  {{ invite.title }}
                </span>
                <span class="text-[10px] uppercase font-medium px-2 py-0.5 rounded-full" 
                  :class="{
                    'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': invite.status === 'completed',
                    'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400': invite.status === 'active'
                  }">
                  {{ invite.status }}
                </span>
              </div>
              <div class="h-4 w-full bg-gray-100 dark:bg-gray-800 rounded-full flex overflow-hidden border dark:border-white/5">
                <div v-for="phase in invite.phases" :key="phase.order" 
                     class="h-full border-r last:border-r-0 border-white/20 relative"
                     :style="{ width: `${100 / invite.phases.length}%` }"
                     :title="`Phase #${phase.order}: ${phase.accepted_count}/${phase.total_invitees} Accepted`"
                >
                  <!-- Base status color -->
                  <div class="absolute inset-0 transition-all"
                       :class="{
                         'bg-emerald-500': phase.status === 'completed',
                         'bg-indigo-500 animate-pulse': phase.status === 'active',
                         'bg-gray-300 dark:bg-gray-700': phase.status === 'pending'
                       }"
                  ></div>
                  <!-- Progress overlay (accepted) -->
                  <div v-if="phase.total_invitees > 0" 
                       class="absolute inset-y-0 left-0 bg-emerald-600 opacity-40"
                       :style="{ width: `${(phase.accepted_count / phase.total_invitees) * 100}%` }"
                  ></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <!-- Main Area (Bottlenecks) -->
      <div class="lg:col-span-2 space-y-4">
        <h3 class="text-lg font-medium text-gray-900 dark:text-white">Active Bottlenecks</h3>
        
        <div v-if="loading && !stats" class="space-y-4">
          <div v-for="i in 2" :key="i" class="h-32 bg-gray-100 dark:bg-white/5 animate-pulse rounded-lg border border-gray-200 dark:border-white/10"></div>
        </div>
        <div v-else-if="stats?.bottlenecks.length === 0" class="bg-white dark:bg-white/5 rounded-lg border border-dashed border-gray-300 dark:border-white/10 p-12 text-center">
          <p class="text-sm text-gray-500 dark:text-gray-400">No active bottlenecks detected.</p>
        </div>
        <div v-else class="grid grid-cols-1 gap-4">
          <div v-for="bn in stats?.bottlenecks" :key="bn.invite_id" 
               class="bg-white dark:bg-white/5 shadow rounded-lg border border-gray-200 dark:border-white/10 p-5 hover:border-indigo-500/50 transition-colors">
            <div class="flex justify-between items-start">
              <div class="space-y-2">
                <div class="flex flex-wrap gap-1">
                  <span v-for="tag in bn.tags" :key="tag.id" 
                        class="px-1.5 py-0.5 rounded-md text-[10px] font-bold uppercase tracking-wider"
                        :style="{ backgroundColor: tag.color + '20', color: tag.color, border: '1px solid ' + tag.color + '40' }">
                    {{ tag.name }}
                  </span>
                </div>
                <h4 class="font-bold text-gray-900 dark:text-white">{{ bn.title }}</h4>
                <div class="flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400">
                  <span class="px-2 py-0.5 bg-gray-100 dark:bg-white/10 rounded">Phase #{{ bn.phase_order }}</span>
                  <span class="capitalize">{{ bn.strategy_kind }}</span>
                  <span v-if="bn.active_since">Active for {{ formatTimeAgo(bn.active_since) }}</span>
                </div>
              </div>
              <button @click="goToInvites" class="text-sm font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                View Details
              </button>
            </div>
            <div class="mt-4 flex items-center">
              <div class="flex-shrink-0">
                <div class="h-10 w-10 rounded-full bg-indigo-100 dark:bg-indigo-900/30 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                  <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z" />
                  </svg>
                </div>
              </div>
              <div class="ml-4">
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Waiting for</p>
                <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ bn.waiting_for }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Sidebar (Activity) -->
      <div class="space-y-4">
        <h3 class="text-lg font-medium text-gray-900 dark:text-white">Recent Activity</h3>
        
        <div v-if="loading && !stats">
          <TableSkeleton :columns="1" :rows="6" />
        </div>
        <div v-else-if="stats?.activity.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
          No recent activity.
        </div>
        <div v-else class="flow-root">
          <ul role="list" class="-mb-8">
            <li v-for="(event, eventIdx) in stats?.activity" :key="eventIdx">
              <div class="relative pb-8">
                <span v-if="stats && eventIdx !== stats.activity.length - 1" class="absolute left-4 top-4 -ml-px h-full w-0.5 bg-gray-200 dark:bg-white/5" aria-hidden="true"></span>
                <div class="relative flex space-x-3">
                  <div>
                    <span :class="[
                      event.type === 'email_error' ? 'bg-red-500' : 'bg-green-500',
                      'h-8 w-8 rounded-full flex items-center justify-center ring-8 ring-white dark:ring-gray-900'
                    ]">
                      <svg v-if="event.type === 'email_error'" class="h-5 w-5 text-white" viewBox="0 0 20 20" fill="currentColor">
                        <path fill-rule="evenodd" d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0Zm-8-5a.75.75 0 0 1 .75.75v4.5a.75.75 0 0 1-1.5 0v-4.5A.75.75 0 0 1 10 5Zm0 10a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z" clip-rule="evenodd" />
                      </svg>
                      <svg v-else class="h-5 w-5 text-white" viewBox="0 0 20 20" fill="currentColor">
                        <path fill-rule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm3.857-9.809a.75.75 0 0 0-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 1 0-1.06 1.061l2.5 2.5a.75.75 0 0 0 1.137-.089l4-5.5Z" clip-rule="evenodd" />
                      </svg>
                    </span>
                  </div>
                  <div class="flex min-w-0 flex-1 justify-between space-x-4 pt-1.5">
                    <div>
                      <p class="text-sm text-gray-500 dark:text-gray-300">{{ event.message }}</p>
                    </div>
                    <div class="whitespace-nowrap text-right text-xs text-gray-500 dark:text-gray-400">
                      <time v-if="event.timestamp" :datetime="event.timestamp">{{ formatTimeAgo(event.timestamp) }}</time>
                    </div>
                  </div>
                </div>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>
