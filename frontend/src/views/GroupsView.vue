<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import type { components } from '../api-types'

type Group = components['schemas']['Group']
type NewGroup = components['schemas']['NewGroup']
type UpdateGroup = components['schemas']['UpdateGroup']

const groups = ref<Group[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Modal State
const isModalOpen = ref(false)
const isSaving = ref(false)
const editingGroup = ref<Group | null>(null)
const form = reactive({
  name: '',
  description: '',
})

async function fetchGroups() {
  loading.value = true
  try {
    const response = await fetch('/api/groups')
    if (!response.ok) throw new Error('Failed to fetch groups')
    groups.value = await response.json()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  editingGroup.value = null
  form.name = ''
  form.description = ''
  isModalOpen.value = true
}

function openEditModal(group: Group) {
  editingGroup.value = group
  form.name = group.name
  form.description = group.description || ''
  isModalOpen.value = true
}

function closeModal() {
  isModalOpen.value = false
  editingGroup.value = null
}

async function saveGroup() {
  isSaving.value = true
  try {
    let response: Response
    if (editingGroup.value) {
      // Update
      const body: UpdateGroup = {
        name: form.name,
        description: form.description,
      }
      response = await fetch(`/api/groups/${editingGroup.value.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    } else {
      // Create
      const body: NewGroup = {
        name: form.name,
        description: form.description,
      }
      response = await fetch('/api/groups', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    }

    if (!response.ok) {
      const errData = await response.json()
      throw new Error(errData.message || 'Failed to save group')
    }

    await fetchGroups()
    closeModal()
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Failed to save')
  } finally {
    isSaving.value = false
  }
}

async function deleteGroup(group: Group) {
  if (!confirm(`Are you sure you want to delete group "${group.name}"?`)) return

  try {
    const response = await fetch(`/api/groups/${group.id}`, {
      method: 'DELETE',
    })
    if (!response.ok) throw new Error('Failed to delete group')
    await fetchGroups()
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Failed to delete')
  }
}

onMounted(() => {
  fetchGroups()
})
</script>

<template>
  <div>
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">Groups</h2>
        <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">
          Manage your contact groups to easily organize and send bulk invites.
        </p>
      </div>
      <div class="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
        <button
          @click="openCreateModal"
          type="button"
          class="block rounded-md bg-indigo-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
        >
          Add group
        </button>
      </div>
    </div>

    <div class="mt-8 flow-root">
      <div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
          <div v-if="loading && groups.length === 0" class="text-center py-4 text-gray-500">Loading groups...</div>
          <div v-else-if="error" class="text-center py-4 text-red-500">Error: {{ error }}</div>
          <table v-else class="min-w-full divide-y divide-gray-300 dark:divide-white/10">
            <thead>
              <tr>
                <th
                  scope="col"
                  class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-0"
                >
                  Name
                </th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  Description
                </th>
                <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-0">
                  <span class="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-white/5">
              <tr v-for="group in groups" :key="group.id">
                <td
                  class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-0"
                >
                  {{ group.name }}
                </td>
                <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-400">
                  {{ group.description || '-' }}
                </td>
                <td
                  class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-0 space-x-4"
                >
                  <button
                    @click="openEditModal(group)"
                    class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300"
                  >
                    Edit
                  </button>
                  <button
                    @click="deleteGroup(group)"
                    class="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
                  >
                    Delete
                  </button>
                </td>
              </tr>
              <tr v-if="groups.length === 0 && !loading">
                <td colspan="3" class="text-center py-4 text-gray-500 italic">No groups found.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Edit/Create Modal -->
    <div v-if="isModalOpen" class="relative z-10" aria-labelledby="modal-title" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"></div>

      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg sm:p-6">
            <div>
              <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white" id="modal-title">
                {{ editingGroup ? 'Edit Group' : 'Add Group' }}
              </h3>
              <div class="mt-4 space-y-4">
                <div>
                  <label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Name</label>
                  <input
                    v-model="form.name"
                    type="text"
                    name="name"
                    id="name"
                    class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                    placeholder="Work Friends"
                  />
                </div>
                <div>
                  <label for="description" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Description</label>
                  <textarea
                    v-model="form.description"
                    name="description"
                    id="description"
                    rows="3"
                    class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                    placeholder="People from the office"
                  ></textarea>
                </div>
              </div>
            </div>
            <div class="mt-5 sm:mt-6 sm:grid sm:grid-flow-row-dense sm:grid-cols-2 sm:gap-3">
              <button
                @click="saveGroup"
                :disabled="isSaving"
                type="button"
                class="inline-flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-base font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 sm:col-start-2 sm:text-sm disabled:opacity-50"
              >
                {{ isSaving ? 'Saving...' : 'Save' }}
              </button>
              <button
                @click="closeModal"
                type="button"
                class="mt-3 inline-flex w-full justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-base font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 sm:col-start-1 sm:mt-0 sm:text-sm dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:hover:bg-gray-600"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
