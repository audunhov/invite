<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { components } from '../api-types'

type Person = components['schemas']['Person']

const persons = ref<Person[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

async function fetchPersons() {
  try {
    const response = await fetch('/api/persons')
    if (!response.ok) throw new Error('Failed to fetch persons')
    persons.value = await response.json()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    loading.value = false
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
          type="button"
          class="block rounded-md bg-indigo-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
        >
          Add person
        </button>
      </div>
    </div>

    <div class="mt-8 flow-root">
      <div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
          <div v-if="loading" class="text-center py-4 text-gray-500">Loading persons...</div>
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
                  Email
                </th>
                <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-0">
                  <span class="sr-only">Edit</span>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-white/5">
              <tr v-for="person in persons" :key="person.id">
                <td
                  class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-0"
                >
                  {{ person.name }}
                </td>
                <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-400">
                  {{ person.email }}
                </td>
                <td
                  class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-0"
                >
                  <a href="#" class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300">
                    Edit<span class="sr-only">, {{ person.name }}</span>
                  </a>
                </td>
              </tr>
              <tr v-if="persons.length === 0">
                <td colspan="3" class="text-center py-4 text-gray-500 italic">No persons found.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>
