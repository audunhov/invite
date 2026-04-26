<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import type { components } from '../api-types'
import { client } from '../utils/api'
import { notify } from '../utils/toast'
import { useConfirm } from '../composables/useConfirm'
import { useTheme } from '../composables/useTheme'
import {
  UserIcon,
  PaintBrushIcon,
  ShieldCheckIcon,
  TagIcon,
  ComputerDesktopIcon,
  SunIcon,
  MoonIcon,
} from '@heroicons/vue/24/outline'

type Tag = components['schemas']['Tag']
type NewTag = components['schemas']['NewTag']
type Session = components['schemas']['Session']

const auth = useAuthStore()
const { confirm } = useConfirm()
const { theme } = useTheme()

const currentTab = ref('profile')
const tabs = [
  { id: 'profile', name: 'Profile', icon: UserIcon },
  { id: 'appearance', name: 'Appearance', icon: PaintBrushIcon },
  { id: 'security', name: 'Security', icon: ShieldCheckIcon },
  { id: 'tags', name: 'Tags', icon: TagIcon },
]

// Profile Logic
// ... (unchanged)
const isSavingProfile = ref(false)
const profileForm = reactive({
  name: auth.user?.name || '',
  email: auth.user?.email || '',
})

async function updateProfile() {
  if (!auth.user?.id) return
  isSavingProfile.value = true
  try {
    const { error } = await client.PATCH('/persons/{id}', {
      params: { path: { id: auth.user.id } },
      body: {
        name: profileForm.name,
        email: profileForm.email,
      },
    })
    if (error) throw error
    notify.success('Profile updated')
    await auth.checkAuth()
  } catch (err) {
    // Error handled by middleware
  } finally {
    isSavingProfile.value = false
  }
}

// Security Logic
const isSavingPassword = ref(false)
const passwordForm = reactive({
  password: '',
  confirmPassword: '',
})

async function updatePassword() {
  if (!passwordForm.password) {
    notify.error('Password cannot be empty')
    return
  }
  if (passwordForm.password !== passwordForm.confirmPassword) {
    notify.error('Passwords do not match')
    return
  }
  if (!auth.user?.id) return

  isSavingPassword.value = true
  try {
    const { error } = await client.PATCH('/persons/{id}', {
      params: { path: { id: auth.user.id } },
      body: { password: passwordForm.password },
    })
    if (error) throw error
    notify.success('Password updated')
    passwordForm.password = ''
    passwordForm.confirmPassword = ''
  } catch (err) {
    // Error handled by middleware
  } finally {
    isSavingPassword.value = false
  }
}

// Active Sessions Logic
const sessions = ref<Session[]>([])
const isLoadingSessions = ref(false)

async function fetchSessions() {
  isLoadingSessions.value = true
  try {
    const { data, error } = await client.GET('/auth/sessions')
    if (error) throw error
    if (data) sessions.value = data
  } catch (err) {
    // Error handled by middleware
  } finally {
    isLoadingSessions.value = false
  }
}

async function revokeSession(session: Session) {
  const confirmed = await confirm({
    title: 'Revoke Session',
    message: 'Are you sure you want to revoke this session? You will be logged out on that device.',
    variant: 'danger',
    confirmLabel: 'Revoke',
  })
  if (!confirmed) return

  try {
    const { error } = await client.DELETE('/auth/sessions/{id}', {
      params: { path: { id: session.id } },
    })
    if (error) throw error
    notify.success('Session revoked')
    await fetchSessions()
  } catch (err) {
    // Error handled by middleware
  }
}

// Tag Management Logic
const tags = ref<Tag[]>([])
const isLoadingTags = ref(false)
const isSavingTag = ref(false)
const editingTag = ref<Tag | null>(null)
const tagForm = reactive<NewTag>({
  name: '',
  color: '#6366f1',
})

async function fetchTags() {
  isLoadingTags.value = true
  try {
    const { data, error } = await client.GET('/tags')
    if (error) throw error
    if (data) tags.value = data
  } catch (err) {
    notify.error('Failed to load tags')
  } finally {
    isLoadingTags.value = false
  }
}

async function saveTag() {
  if (!tagForm.name) {
    notify.error('Tag name is required')
    return
  }
  isSavingTag.value = true
  try {
    let error
    if (editingTag.value) {
      const res = await client.PATCH('/tags/{id}', {
        params: { path: { id: editingTag.value.id } },
        body: tagForm,
      })
      error = res.error
    } else {
      const res = await client.POST('/tags', {
        body: tagForm,
      })
      error = res.error
    }
    if (error) throw error
    notify.success(editingTag.value ? 'Tag updated' : 'Tag created')
    cancelEdit()
    await fetchTags()
  } catch (err) {
    // Error handled by middleware
  } finally {
    isSavingTag.value = false
  }
}

function editTag(tag: Tag) {
  editingTag.value = tag
  tagForm.name = tag.name
  tagForm.color = tag.color
}

function cancelEdit() {
  editingTag.value = null
  tagForm.name = ''
  tagForm.color = '#6366f1'
}

async function deleteTag(tag: Tag) {
  try {
    const { data, error: usageError } = await client.GET('/tags/{id}', {
      params: { path: { id: tag.id } },
    })
    if (usageError) throw usageError
    const count = data?.count || 0

    let message = `Are you sure you want to delete the tag "${tag.name}"?`
    if (count > 0) {
      message += ` This tag is currently assigned to ${count} invitation(s). Deleting it will remove it from all of them.`
    }

    const confirmed = await confirm({
      title: 'Delete Tag',
      message,
      variant: 'danger',
      confirmLabel: 'Delete',
    })
    if (!confirmed) return

    const { error } = await client.DELETE('/tags/{id}', {
      params: { path: { id: tag.id } },
    })
    if (error) throw error
    notify.success('Tag deleted')
    await fetchTags()
  } catch (err) {
    // Error handled by middleware
  }
}

function selectTab(tabId: string) {
  currentTab.value = tabId
  if (tabId === 'tags') {
    fetchTags()
  } else if (tabId === 'security') {
    fetchSessions()
  }
}

onMounted(() => {
  if (currentTab.value === 'tags') {
    fetchTags()
  } else if (currentTab.value === 'security') {
    fetchSessions()
  }
})
</script>

<template>
  <div class="lg:grid lg:grid-cols-12 lg:gap-x-5">
    <aside class="py-6 px-2 sm:px-6 lg:col-span-3 lg:py-0 lg:px-0">
      <nav class="space-y-1">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="selectTab(tab.id)"
          :class="[
            currentTab === tab.id
              ? 'bg-gray-50 text-indigo-700 dark:bg-white/10 dark:text-white'
              : 'text-gray-900 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-white/5 dark:hover:text-white',
            'group flex items-center rounded-md px-3 py-2 text-sm font-medium w-full text-left',
          ]"
        >
          <component
            :is="tab.icon"
            :class="[
              currentTab === tab.id
                ? 'text-indigo-500 dark:text-white'
                : 'text-gray-400 group-hover:text-gray-500 dark:text-gray-400 dark:group-hover:text-white',
              'mr-3 size-6 shrink-0',
            ]"
            aria-hidden="true"
          />
          <span class="truncate">{{ tab.name }}</span>
        </button>
      </nav>
    </aside>

    <div class="space-y-6 sm:px-6 lg:col-span-9 lg:px-0">
      <!-- Profile Section -->
      <section v-if="currentTab === 'profile'">
        <form @submit.prevent="updateProfile">
          <div class="shadow sm:overflow-hidden sm:rounded-md border border-gray-200 dark:border-white/10">
            <div class="bg-white px-4 py-5 dark:bg-gray-800 sm:p-6">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">Profile</h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Update your personal information.</p>
              </div>

              <div class="mt-6 grid grid-cols-4 gap-6">
                <div class="col-span-4 sm:col-span-2">
                  <label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Name</label>
                  <input
                    v-model="profileForm.name"
                    type="text"
                    id="name"
                    class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2 border"
                  />
                </div>

                <div class="col-span-4 sm:col-span-2">
                  <label for="email" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Email</label>
                  <input
                    v-model="profileForm.email"
                    type="email"
                    id="email"
                    class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2 border"
                  />
                </div>
              </div>
            </div>
            <div class="bg-gray-50 px-4 py-3 text-right dark:bg-gray-900/50 sm:px-6 border-t dark:border-white/10">
              <button
                type="submit"
                :disabled="isSavingProfile"
                class="inline-flex justify-center rounded-md bg-indigo-600 py-2 px-3 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-50"
              >
                {{ isSavingProfile ? 'Saving...' : 'Save' }}
              </button>
            </div>
          </div>
        </form>
      </section>

      <!-- Appearance Section -->
      <section v-if="currentTab === 'appearance'">
        <div class="shadow sm:overflow-hidden sm:rounded-md border border-gray-200 dark:border-white/10">
          <div class="bg-white px-4 py-5 dark:bg-gray-800 sm:p-6">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">Appearance</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Choose how Invite API looks to you.</p>
            </div>

            <div class="mt-6">
              <div class="grid grid-cols-3 gap-3">
                <button
                  @click="theme = 'light'"
                  :class="[
                    theme === 'light'
                      ? 'ring-2 ring-indigo-600 dark:ring-indigo-500'
                      : 'border-gray-200 dark:border-white/10 hover:bg-gray-50 dark:hover:bg-white/5',
                    'relative flex cursor-pointer flex-col rounded-lg border p-4 focus:outline-none',
                  ]"
                >
                  <SunIcon class="size-6 text-gray-400 dark:text-gray-500 mb-2" />
                  <span class="text-sm font-medium text-gray-900 dark:text-white">Light</span>
                </button>

                <button
                  @click="theme = 'dark'"
                  :class="[
                    theme === 'dark'
                      ? 'ring-2 ring-indigo-600 dark:ring-indigo-500'
                      : 'border-gray-200 dark:border-white/10 hover:bg-gray-50 dark:hover:bg-white/5',
                    'relative flex cursor-pointer flex-col rounded-lg border p-4 focus:outline-none',
                  ]"
                >
                  <MoonIcon class="size-6 text-gray-400 dark:text-gray-500 mb-2" />
                  <span class="text-sm font-medium text-gray-900 dark:text-white">Dark</span>
                </button>

                <button
                  @click="theme = 'system'"
                  :class="[
                    theme === 'system'
                      ? 'ring-2 ring-indigo-600 dark:ring-indigo-500'
                      : 'border-gray-200 dark:border-white/10 hover:bg-gray-50 dark:hover:bg-white/5',
                    'relative flex cursor-pointer flex-col rounded-lg border p-4 focus:outline-none',
                  ]"
                >
                  <ComputerDesktopIcon class="size-6 text-gray-400 dark:text-gray-500 mb-2" />
                  <span class="text-sm font-medium text-gray-900 dark:text-white">System</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Security Section -->
      <section v-if="currentTab === 'security'" class="space-y-6">
        <form @submit.prevent="updatePassword">
          <div class="shadow sm:overflow-hidden sm:rounded-md border border-gray-200 dark:border-white/10">
            <div class="bg-white px-4 py-5 dark:bg-gray-800 sm:p-6">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">Security</h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Update your password to stay secure.</p>
              </div>

              <div class="mt-6 grid grid-cols-4 gap-6">
                <div class="col-span-4 sm:col-span-2">
                  <label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">New Password</label>
                  <input
                    v-model="passwordForm.password"
                    type="password"
                    id="password"
                    class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2 border"
                  />
                </div>

                <div class="col-span-4 sm:col-span-2">
                  <label for="confirm-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Confirm Password</label>
                  <input
                    v-model="passwordForm.confirmPassword"
                    type="password"
                    id="confirm-password"
                    class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2 border"
                  />
                </div>
              </div>
            </div>
            <div class="bg-gray-50 px-4 py-3 text-right dark:bg-gray-900/50 sm:px-6 border-t dark:border-white/10">
              <button
                type="submit"
                :disabled="isSavingPassword"
                class="inline-flex justify-center rounded-md bg-indigo-600 py-2 px-3 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-50"
              >
                {{ isSavingPassword ? 'Updating...' : 'Update Password' }}
              </button>
            </div>
          </div>
        </form>

        <div class="shadow sm:overflow-hidden sm:rounded-md border border-gray-200 dark:border-white/10">
          <div class="bg-white px-4 py-5 dark:bg-gray-800 sm:p-6">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">Active Sessions</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Manage your active sessions and log out from other devices.</p>
            </div>

            <div class="mt-6">
              <div v-if="isLoadingSessions" class="flex justify-center py-4">
                <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-indigo-500"></div>
              </div>
              <ul v-else class="divide-y divide-gray-200 dark:divide-white/5 border-t dark:border-white/5">
                <li v-for="session in sessions" :key="session.id" class="py-4 flex items-center justify-between">
                  <div class="flex items-center">
                    <ComputerDesktopIcon class="size-6 text-gray-400 mr-3" />
                    <div>
                      <p class="text-sm font-medium text-gray-900 dark:text-white flex items-center">
                        {{ session.is_current ? 'Current Session' : 'Other Session' }}
                        <span v-if="session.is_current" class="ml-2 inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20 dark:bg-green-500/10 dark:text-green-400 dark:ring-green-500/20">Active</span>
                      </p>
                      <p class="text-xs text-gray-500 dark:text-gray-400">
                        Started on {{ new Date(session.created_at).toLocaleString() }}
                      </p>
                    </div>
                  </div>
                  <button
                    v-if="!session.is_current"
                    @click="revokeSession(session)"
                    class="text-sm font-medium text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
                  >
                    Revoke
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      <!-- Tags Section -->
      <section v-if="currentTab === 'tags'">
        <div class="shadow sm:overflow-hidden sm:rounded-md border border-gray-200 dark:border-white/10">
          <div class="bg-white px-4 py-5 dark:bg-gray-800 sm:p-6">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">Tags</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Manage tags to organize your invitations.</p>
            </div>

            <div class="mt-6">
              <!-- Tag Form -->
              <form @submit.prevent="saveTag" class="flex flex-col sm:flex-row gap-4 items-end mb-6">
                <div class="flex-1 w-full">
                  <label for="tag-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Tag Name</label>
                  <input
                    v-model="tagForm.name"
                    type="text"
                    id="tag-name"
                    placeholder="e.g. Work, Family, VIP"
                    class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2 border"
                    required
                  />
                </div>
                <div class="w-full sm:w-32">
                  <label for="tag-color" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Color</label>
                  <div class="mt-1 flex gap-2 items-center">
                    <input
                      v-model="tagForm.color"
                      type="color"
                      id="tag-color"
                      class="h-9 w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 cursor-pointer p-1 border"
                    />
                  </div>
                </div>
                <div class="flex gap-2">
                  <button
                    type="submit"
                    :disabled="isSavingTag"
                    class="inline-flex justify-center rounded-md bg-indigo-600 py-2 px-3 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-50"
                  >
                    {{ editingTag ? 'Update' : 'Add' }}
                  </button>
                  <button
                    v-if="editingTag"
                    type="button"
                    @click="cancelEdit"
                    class="inline-flex justify-center rounded-md bg-white py-2 px-3 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 dark:bg-white/5 dark:text-white dark:ring-white/10 dark:hover:bg-white/10"
                  >
                    Cancel
                  </button>
                </div>
              </form>

              <!-- Tags List -->
              <div v-if="isLoadingTags" class="flex justify-center py-4">
                <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-indigo-500"></div>
              </div>
              <div v-else-if="tags.length === 0" class="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
                No tags created yet.
              </div>
              <ul v-else class="divide-y divide-gray-200 dark:divide-white/5 border-t dark:border-white/5">
                <li v-for="tag in tags" :key="tag.id" class="py-4 flex items-center justify-between">
                  <div class="flex items-center">
                    <span
                      class="inline-block w-4 h-4 rounded-full mr-3 shadow-sm"
                      :style="{ backgroundColor: tag.color }"
                    ></span>
                    <span class="text-sm font-medium text-gray-900 dark:text-white">{{ tag.name }}</span>
                  </div>
                  <div class="flex gap-4">
                    <button
                      @click="editTag(tag)"
                      class="text-sm font-medium text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300"
                    >
                      Edit
                    </button>
                    <button
                      @click="deleteTag(tag)"
                      class="text-sm font-medium text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
                    >
                      Delete
                    </button>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
