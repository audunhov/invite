<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import type { components } from '../api-types'
import { notify } from '../utils/toast'
import { useConfirm } from '../composables/useConfirm'
import TableSkeleton from '../components/TableSkeleton.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'

type Group = components['schemas']['Group']
type NewGroup = components['schemas']['NewGroup']
type UpdateGroup = components['schemas']['UpdateGroup']
type Person = components['schemas']['Person']

const { confirm } = useConfirm()
const groups = ref<Group[]>([])
const persons = ref<Person[]>([]) // For member assignment
const currentGroupMembers = ref<Person[]>([])
const loading = ref(true)
const loadingMembers = ref(false)
const error = ref<string | null>(null)

// Modal State
const isModalOpen = ref(false)
const isMembersModalOpen = ref(false)
const isSaving = ref(false)
const isAddingMember = ref(false)

const editingGroup = ref<Group | null>(null)
const selectedGroupId = ref<string | null>(null)

const form = reactive({
  name: '',
  description: '',
})

const memberForm = reactive({
  personId: '',
})

// Filter persons who are NOT in the current group
const availablePersons = computed(() => {
  const members = currentGroupMembers.value || []
  const memberIds = new Set(members.map(m => m.id))
  return persons.value.filter(p => !memberIds.has(p.id))
})

async function fetchData() {
  loading.value = true
  try {
    const [groupsRes, personsRes] = await Promise.all([
      fetch('/api/groups'),
      fetch('/api/persons')
    ])
    
    if (!groupsRes.ok || !personsRes.ok) throw new Error('Failed to fetch data')
    
    groups.value = await groupsRes.json()
    persons.value = await personsRes.json()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    loading.value = false
  }
}

async function fetchMembers(groupId: string) {
  loadingMembers.value = true
  try {
    const response = await fetch(`/api/groups/${groupId}/members`)
    if (!response.ok) throw new Error('Failed to fetch members')
    currentGroupMembers.value = (await response.json()) || []
  } catch (err) {
    notify.error(err instanceof Error ? err.message : 'Error fetching members')
  } finally {
    loadingMembers.value = false
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

async function openMembersModal(group: Group) {
  selectedGroupId.value = group.id
  currentGroupMembers.value = []
  memberForm.personId = ''
  isMembersModalOpen.value = true
  await fetchMembers(group.id)
}

function closeModal() {
  isModalOpen.value = false
  isMembersModalOpen.value = false
  editingGroup.value = null
  selectedGroupId.value = null
}

async function saveGroup() {
  isSaving.value = true
  try {
    let response: Response
    if (editingGroup.value) {
      const body: UpdateGroup = { name: form.name, description: form.description }
      response = await fetch(`/api/groups/${editingGroup.value.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    } else {
      const body: NewGroup = { name: form.name, description: form.description }
      response = await fetch('/api/groups', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    }

    if (!response.ok) throw new Error('Failed to save group')
    await fetchData()
    closeModal()
    notify.success('Group saved successfully')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : 'Failed to save group')
  } finally {
    isSaving.value = false
  }
}

async function deleteGroup(group: Group) {
  const isConfirmed = await confirm({
    title: 'Delete Group',
    message: `Are you sure you want to delete group "${group.name}"?`,
    variant: 'danger',
    confirmLabel: 'Delete'
  })

  if (!isConfirmed) return

  try {
    const response = await fetch(`/api/groups/${group.id}`, { method: 'DELETE' })
    if (!response.ok) throw new Error('Failed to delete group')
    await fetchData()
    notify.success('Group deleted')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : 'Failed to delete group')
  }
}

async function addMember() {
  if (!selectedGroupId.value || !memberForm.personId) return
  isAddingMember.value = true
  try {
    const response = await fetch(`/api/groups/${selectedGroupId.value}/members`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ person_id: memberForm.personId }),
    })
    if (!response.ok) throw new Error('Failed to add member')
    await fetchMembers(selectedGroupId.value)
    memberForm.personId = ''
    notify.success('Member added')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : 'Failed to add member')
  } finally {
    isAddingMember.value = false
  }
}

async function removeMember(person: Person) {
  if (!selectedGroupId.value) return
  
  const isConfirmed = await confirm({
    title: 'Remove Member',
    message: `Are you sure you want to remove ${person.name} from this group?`,
    variant: 'danger',
    confirmLabel: 'Remove'
  })

  if (!isConfirmed) return

  try {
    const response = await fetch(`/api/groups/${selectedGroupId.value}/members/${person.id}`, {
      method: 'DELETE',
    })
    if (!response.ok) throw new Error('Failed to remove member')
    await fetchMembers(selectedGroupId.value)
    notify.success('Member removed')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : 'Failed to remove member')
  }
}

onMounted(fetchData)
</script>

<template>
  <div>
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">Groups</h2>
        <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">
          Manage your contact groups and their members.
        </p>
      </div>
      <div class="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
        <button
          @click="openCreateModal"
          type="button"
          class="block rounded-md bg-indigo-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
        >
          Add group
        </button>
      </div>
    </div>

    <!-- Groups Table -->
    <div class="mt-8 flow-root">
      <div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
          <TableSkeleton v-if="loading && groups.length === 0" :columns="2" />
          <table v-else class="min-w-full divide-y divide-gray-300 dark:divide-white/10">
            <thead>
              <tr>
                <th scope="col" class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-0">Name</th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Description</th>
                <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-0"><span class="sr-only">Actions</span></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-white/5">
              <tr v-for="group in groups" :key="group.id">
                <td class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-0">{{ group.name }}</td>
                <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-400">{{ group.description || '-' }}</td>
                <td class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-0 space-x-4">
                  <button @click="openMembersModal(group)" class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400">Members</button>
                  <button @click="openEditModal(group)" class="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-300">Edit</button>
                  <button @click="deleteGroup(group)" class="text-red-600 hover:text-red-900 dark:text-red-400">Delete</button>
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

    <!-- Edit/Create Group Modal -->
    <div v-if="isModalOpen" class="relative z-10" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"></div>
      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg sm:p-6">
            <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white">{{ editingGroup ? 'Edit Group' : 'Add Group' }}</h3>
            <div class="mt-4 space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Name</label>
                <input v-model="form.name" type="text" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm sm:text-sm p-2 border" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Description</label>
                <textarea v-model="form.description" rows="3" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm sm:text-sm p-2 border"></textarea>
              </div>
            </div>
            <div class="mt-5 sm:mt-6 sm:grid sm:grid-cols-2 sm:gap-3">
              <button @click="saveGroup" :disabled="isSaving" class="inline-flex w-full justify-center items-center rounded-md bg-indigo-600 px-4 py-2 text-white shadow-sm hover:bg-indigo-700 disabled:opacity-50 gap-2">
                <LoadingSpinner v-if="isSaving" size="sm" />
                {{ isSaving ? 'Saving...' : 'Save' }}
              </button>
              <button @click="closeModal" class="mt-3 inline-flex w-full justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-gray-700 dark:bg-gray-700 dark:text-gray-200">Cancel</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Members Management Modal -->
    <div v-if="isMembersModalOpen" class="relative z-10" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"></div>
      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-xl sm:p-6">
            <div class="flex justify-between items-center mb-4">
              <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white">Group Members</h3>
              <button @click="closeModal" class="text-gray-400 hover:text-gray-500">✕</button>
            </div>

            <!-- Add Member Form -->
            <div class="bg-gray-50 dark:bg-gray-900/50 p-4 rounded-lg mb-6 border dark:border-white/10">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Add New Member</label>
              <div class="flex gap-3">
                <select 
                  v-model="memberForm.personId" 
                  class="block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm sm:text-sm p-2 border"
                >
                  <option value="">Select a person...</option>
                  <option v-for="person in availablePersons" :key="person.id" :value="person.id">
                    {{ person.name }} ({{ person.email }})
                  </option>
                </select>
                <button 
                  @click="addMember" 
                  :disabled="!memberForm.personId || isAddingMember"
                  class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 gap-2"
                >
                  <LoadingSpinner v-if="isAddingMember" size="sm" />
                  {{ isAddingMember ? 'Adding...' : 'Add' }}
                </button>
              </div>
              <p v-if="availablePersons.length === 0 && !loading" class="mt-2 text-xs text-gray-500 italic">
                All available persons are already in this group.
              </p>
            </div>

            <!-- Current Members List -->
            <div class="max-h-64 overflow-y-auto">
              <div v-if="loadingMembers" class="text-center py-4 text-gray-500 text-sm">Loading members...</div>
              <ul v-else-if="currentGroupMembers.length > 0" class="divide-y divide-gray-200 dark:divide-white/5">
                <li v-for="person in currentGroupMembers" :key="person.id" class="py-3 flex justify-between items-center">
                  <div>
                    <p class="text-sm font-medium text-gray-900 dark:text-white">{{ person.name }}</p>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ person.email }}</p>
                  </div>
                  <button @click="removeMember(person)" class="text-red-600 hover:text-red-900 text-sm font-medium">Remove</button>
                </li>
              </ul>
              <p v-else class="text-center py-4 text-gray-500 italic text-sm">No members in this group yet.</p>
            </div>
            
            <div class="mt-6">
              <button @click="closeModal" class="w-full inline-flex justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-gray-700 dark:bg-gray-700 dark:text-gray-200 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-600">
                Close
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
