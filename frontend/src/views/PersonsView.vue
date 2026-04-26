<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import type { components } from '../api-types'
import { client } from '../utils/api'
import { notify } from '../utils/toast'
import { useConfirm } from '../composables/useConfirm'
import TableSkeleton from '../components/TableSkeleton.vue'
import LoadingSpinner from '../components/LoadingSpinner.vue'

type Person = components['schemas']['Person']
type NewPerson = components['schemas']['NewPerson']
type UpdatePerson = components['schemas']['UpdatePerson']

const { confirm } = useConfirm()
const persons = ref<Person[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Modal State
const isModalOpen = ref(false)
const isSaving = ref(false)
const editingPerson = ref<Person | null>(null)
const form = reactive({
  name: '',
  email: '',
})

async function fetchPersons() {
  loading.value = true
  try {
    const { data, error: err } = await client.GET('/persons')
    if (err) throw err
    persons.value = data || []
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  editingPerson.value = null
  form.name = ''
  form.email = ''
  isModalOpen.value = true
}

function openEditModal(person: Person) {
  editingPerson.value = person
  form.name = person.name
  form.email = person.email
  isModalOpen.value = true
}

function closeModal() {
  isModalOpen.value = false
  editingPerson.value = null
}

async function savePerson() {
  isSaving.value = true
  try {
    if (editingPerson.value) {
      // Update
      const { error: err } = await client.PATCH('/persons/{id}', {
        params: { path: { id: editingPerson.value.id } },
        body: {
          name: form.name,
          email: form.email,
        },
      })
      if (err) throw err
    } else {
      // Create
      const { error: err } = await client.POST('/persons', {
        body: {
          name: form.name,
          email: form.email,
        },
      })
      if (err) throw err
    }

    await fetchPersons()
    closeModal()
    notify.success('Person saved successfully')
  } catch (err) {
    // Error is handled by global client middleware, but we catch it here to stop the flow
    console.error('Save failed:', err)
  } finally {
    isSaving.value = false
  }
}

async function deletePerson(person: Person) {
  const isConfirmed = await confirm({
    title: 'Remove Person',
    message: `Are you sure you want to remove ${person.name}? This will also remove them from any groups they are in.`,
    variant: 'danger',
    confirmLabel: 'Remove'
  })

  if (!isConfirmed) return

  try {
    const { error: err } = await client.DELETE('/persons/{id}', {
      params: { path: { id: person.id } },
    })
    if (err) throw err
    await fetchPersons()
    notify.success('Person removed')
  } catch (err) {
    console.error('Delete failed:', err)
  }
}

onMounted(() => {
  fetchPersons()
})
</script>

<template>
  <div>
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">Persons</h2>
        <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">
          A list of all the persons in your account including their name and email.
        </p>
      </div>
      <div class="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
        <button
          @click="openCreateModal"
          type="button"
          class="block rounded-md bg-indigo-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
        >
          Add person
        </button>
      </div>
    </div>

    <div class="mt-8">
      <div v-if="loading && persons.length === 0" class="space-y-4">
        <TableSkeleton :columns="2" class="hidden sm:block" />
        <div v-for="i in 3" :key="i" class="h-24 bg-gray-100 dark:bg-white/5 animate-pulse rounded-xl sm:hidden"></div>
      </div>
      
      <div v-else-if="error" class="text-center py-12 bg-red-50 dark:bg-red-900/10 rounded-xl border border-red-100 dark:border-red-900/30">
        <p class="text-red-600 dark:text-red-400 font-medium">Error: {{ error }}</p>
      </div>

      <div v-else-if="persons.length === 0" class="text-center py-12 bg-white dark:bg-white/5 rounded-xl border border-dashed border-gray-300 dark:border-white/10">
        <p class="text-gray-500 italic">No persons found.</p>
      </div>

      <div v-else>
        <!-- Desktop Table -->
        <div class="hidden sm:block overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-300 dark:divide-white/10">
            <thead>
              <tr>
                <th scope="col" class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-0">Name</th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Email</th>
                <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-0"><span class="sr-only">Actions</span></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-white/5">
              <tr v-for="person in persons" :key="person.id">
                <td class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-0">{{ person.name }}</td>
                <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-400">{{ person.email }}</td>
                <td class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-0 space-x-3">
                  <button @click="openEditModal(person)" class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400">Edit</button>
                  <button @click="deletePerson(person)" class="text-red-600 hover:text-red-900 dark:text-red-400">Delete</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Mobile Cards -->
        <div class="sm:hidden space-y-4">
          <div v-for="person in persons" :key="person.id" class="bg-white dark:bg-white/5 rounded-xl border border-gray-200 dark:border-white/10 p-4 shadow-sm">
            <div class="flex items-center gap-3 mb-4">
              <div class="size-10 rounded-full bg-indigo-100 dark:bg-indigo-900/30 flex items-center justify-center text-indigo-600 dark:text-indigo-400 font-bold">
                {{ person.name.charAt(0) }}
              </div>
              <div class="min-w-0">
                <h4 class="text-sm font-bold text-gray-900 dark:text-white truncate">{{ person.name }}</h4>
                <p class="text-xs text-gray-500 truncate">{{ person.email }}</p>
              </div>
            </div>
            
            <div class="grid grid-cols-2 gap-2">
              <button @click="openEditModal(person)" class="py-2 rounded-lg bg-gray-50 dark:bg-white/5 text-gray-700 dark:text-gray-300 border dark:border-white/5 text-xs font-bold uppercase">Edit</button>
              <button @click="deletePerson(person)" class="py-2 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-xs font-bold uppercase">Delete</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit/Create Modal -->
    <div v-if="isModalOpen" class="relative z-10" aria-labelledby="modal-title" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500/40 backdrop-blur-sm transition-opacity"></div>

      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg sm:p-6">
            <div>
              <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white" id="modal-title">
                {{ editingPerson ? 'Edit Person' : 'Add Person' }}
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
                    placeholder="Jane Doe"
                  />
                </div>
                <div>
                  <label for="email" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Email</label>
                  <input
                    v-model="form.email"
                    type="email"
                    name="email"
                    id="email"
                    class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                    placeholder="jane@example.com"
                  />
                </div>
              </div>
            </div>
            <div class="mt-5 sm:mt-6 sm:grid sm:grid-flow-row-dense sm:grid-cols-2 sm:gap-3">
              <button
                @click="savePerson"
                :disabled="isSaving"
                type="button"
                class="inline-flex w-full justify-center items-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-base font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 sm:col-start-2 sm:text-sm disabled:opacity-50 gap-2"
              >
                <LoadingSpinner v-if="isSaving" size="sm" />
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
