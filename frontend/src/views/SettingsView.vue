<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import type { components } from '../api-types'
import { client } from '../utils/api'
import { notify } from '../utils/toast'
import { useConfirm } from '../composables/useConfirm'

type UpdatePerson = components['schemas']['UpdatePerson']
type Tag = components['schemas']['Tag']
type NewTag = components['schemas']['NewTag']

const auth = useAuthStore()
const { confirm } = useConfirm()
const isSaving = ref(false)

const form = reactive({
  password: '',
  confirmPassword: '',
})

// Tag Management
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
        body: tagForm
      })
      error = res.error
    } else {
      const res = await client.POST('/tags', {
        body: tagForm
      })
      error = res.error
    }

    if (error) throw error

    notify.success(editingTag.value ? 'Tag updated' : 'Tag created')
    cancelEdit()
    await fetchTags()
  } catch (err) {
    // Middleware handles toasts for non-ok responses
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
    // Check usage
    const { data, error: usageError } = await client.GET('/tags/{id}', {
      params: { path: { id: tag.id } }
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
      params: { path: { id: tag.id } }
    })

    if (error) throw error

    notify.success('Tag deleted')
    await fetchTags()
  } catch (err) {
    // Middleware handles toasts
  }
}

onMounted(() => {
  fetchTags()
})

async function updatePassword() {
  if (!form.password) {
    notify.error('Password cannot be empty')
    return
  }

  if (form.password !== form.confirmPassword) {
    notify.error('Passwords do not match')
    return
  }

  if (!auth.user?.id) {
    notify.error('User session not found')
    return
  }

  isSaving.value = true
  try {
    const { error } = await client.PATCH('/persons/{id}', {
      params: { path: { id: auth.user.id } },
      body: { password: form.password }
    })

    if (error) throw error

    notify.success('Password updated successfully')
    form.password = ''
    form.confirmPassword = ''
  } catch (err) {
    // Middleware handles toasts
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

    <div class="hidden sm:block" aria-hidden="true">
      <div class="py-5">
        <div class="border-t border-gray-200 dark:border-white/10"></div>
      </div>
    </div>

    <div class="mt-10 sm:mt-0 md:grid md:grid-cols-3 md:gap-6">
      <div class="md:col-span-1">
        <div class="px-4 sm:px-0">
          <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white">Tags</h3>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Manage tags to organize your invitations.
          </p>
        </div>
      </div>
      <div class="mt-5 md:mt-0 md:col-span-2">
        <div class="shadow sm:rounded-md border border-gray-200 dark:border-white/10 overflow-hidden">
          <div class="px-4 py-5 bg-white dark:bg-gray-800 space-y-6 sm:p-6">
            <!-- Tag Form -->
            <form @submit.prevent="saveTag" class="flex flex-col sm:flex-row gap-4 items-end">
              <div class="flex-1 w-full">
                <label for="tag-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Tag Name</label>
                <input
                  v-model="tagForm.name"
                  type="text"
                  id="tag-name"
                  placeholder="e.g. Work, Family, VIP"
                  class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
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
                  class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
                >
                  {{ editingTag ? 'Update' : 'Add' }}
                </button>
                <button
                  v-if="editingTag"
                  type="button"
                  @click="cancelEdit"
                  class="inline-flex justify-center py-2 px-4 border border-gray-300 dark:border-gray-600 shadow-sm text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                >
                  Cancel
                </button>
              </div>
            </form>

            <!-- Tags List -->
            <div class="mt-6">
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
                      class="text-sm text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300 font-medium"
                    >
                      Edit
                    </button>
                    <button 
                      @click="deleteTag(tag)"
                      class="text-sm text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 font-medium"
                    >
                      Delete
                    </button>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
