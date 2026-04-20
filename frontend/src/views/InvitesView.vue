<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import type { components } from '../api-types'

type Invite = components['schemas']['Invite']
type NewInvite = components['schemas']['NewInvite']
type UpdateInvite = components['schemas']['UpdateInvite']
type InvitePhase = components['schemas']['InvitePhase']
type NewInvitePhase = components['schemas']['NewInvitePhase']
type InviteStatusReport = components['schemas']['InviteStatusReport']
type InviteeStatus = components['schemas']['InviteeStatus']
type Person = components['schemas']['Person']
type Group = components['schemas']['Group']

const invites = ref<Invite[]>([])
const persons = ref<Person[]>([])
const groups = ref<Group[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Invite Modal
const isInviteModalOpen = ref(false)
const isSavingInvite = ref(false)
const editingInvite = ref<Invite | null>(null)
const inviteForm = reactive({
  title: '',
  description: '',
  from: new Date(Date.now() + 86400000).toISOString().slice(0, 16), // Tomorrow
  to: '',
  from_person_id: '',
})

// Phases Modal
const isPhasesModalOpen = ref(false)
const selectedInviteForPhases = ref<Invite | null>(null)
const phases = ref<InvitePhase[]>([])
const loadingPhases = ref(false)
const isAddingPhase = ref(false)
const phaseForm = reactive({
  order: 1,
  strategy_kind: 'ladder' as 'ladder' | 'sprint',
  timeout_minutes: 60,
  selectedRecipientIds: [] as string[], // Keep internal order for Ladder
})

// Status Modal
const isStatusModalOpen = ref(false)
const statusReport = ref<InviteStatusReport | null>(null)
const loadingStatus = ref(false)

async function fetchData() {
  loading.value = true
  try {
    const [invitesRes, personsRes, groupsRes] = await Promise.all([
      fetch('/api/invites'),
      fetch('/api/persons'),
      fetch('/api/groups')
    ])
    if (!invitesRes.ok) throw new Error('Failed to fetch invites')
    invites.value = await invitesRes.json()
    persons.value = await (personsRes.ok ? personsRes.json() : [])
    groups.value = await (groupsRes.ok ? groupsRes.json() : [])
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    loading.value = false
  }
}

// Helper to get name from ID
function getRecipientName(id: string) {
  const p = persons.value.find(p => p.id === id)
  if (p) return p.name + ' (Person)'
  const g = groups.value.find(g => g.id === id)
  if (g) return g.name + ' (Group)'
  return 'Unknown'
}

function moveRecipient(index: number, direction: number) {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= phaseForm.selectedRecipientIds.length) return
  const item = phaseForm.selectedRecipientIds.splice(index, 1)[0]
  phaseForm.selectedRecipientIds.splice(newIndex, 0, item)
}

function toggleRecipient(id: string) {
  const index = phaseForm.selectedRecipientIds.indexOf(id)
  if (index > -1) {
    phaseForm.selectedRecipientIds.splice(index, 1)
  } else {
    phaseForm.selectedRecipientIds.push(id)
  }
}

function openCreateInviteModal() {
  editingInvite.value = null
  inviteForm.title = ''
  inviteForm.description = ''
  inviteForm.from = new Date(Date.now() + 86400000).toISOString().slice(0, 16)
  inviteForm.to = ''
  // Default to Tom Cook if found
  const tom = persons.value.find(p => p.email === 'tom@example.com')
  inviteForm.from_person_id = tom ? tom.id : ''
  isInviteModalOpen.value = true
}

function openEditInviteModal(invite: Invite) {
  editingInvite.value = invite
  inviteForm.title = invite.title
  inviteForm.description = invite.description || ''
  inviteForm.from = new Date(invite.from).toISOString().slice(0, 16)
  inviteForm.to = invite.to ? new Date(invite.to).toISOString().slice(0, 16) : ''
  inviteForm.from_person_id = invite.from_person_id
  isInviteModalOpen.value = true
}

async function saveInvite() {
  isSavingInvite.value = true
  try {
    const body: Record<string, any> = {
      title: inviteForm.title,
      description: inviteForm.description,
      from: new Date(inviteForm.from).toISOString(),
      from_person_id: inviteForm.from_person_id,
    }
    if (inviteForm.to) {
      body.to = new Date(inviteForm.to).toISOString()
    }
    const method = editingInvite.value ? 'PATCH' : 'POST'
    const url = editingInvite.value ? `/api/invites/${editingInvite.value.id}` : '/api/invites'
    
    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })

    if (!response.ok) throw new Error('Failed to save invite')
    await fetchData()
    isInviteModalOpen.value = false
  } catch (err) {
    alert(err)
  } finally {
    isSavingInvite.value = false
  }
}

async function deleteInvite(invite: Invite) {
  const message = invite.status === 'active'
    ? `Warning: This invite is currently ACTIVE. Deleting it will stop all pending notifications and invalidate all magic links. Proceed?`
    : `Delete invite "${invite.title}"?`
    
  if (!confirm(message)) return
  try {
    const response = await fetch(`/api/invites/${invite.id}`, { method: 'DELETE' })
    if (!response.ok) throw new Error('Failed to delete')
    await fetchData()
  } catch (err) {
    alert(err)
  }
}

async function openPhasesModal(invite: Invite) {
  selectedInviteForPhases.value = invite
  isPhasesModalOpen.value = true
  await fetchPhases(invite.id)
}

async function fetchPhases(inviteId: string) {
  loadingPhases.value = true
  try {
    const response = await fetch(`/api/invites/${inviteId}/phases`)
    phases.value = await response.json()
    phaseForm.order = phases.value.length + 1
  } catch (err) {
    alert(err)
  } finally {
    loadingPhases.value = false
  }
}

async function addPhase() {
  if (!selectedInviteForPhases.value) return
  isAddingPhase.value = true
  try {
    let strategy_config: any = {}
    if (phaseForm.strategy_kind === 'ladder') {
      const selectedPersons = phaseForm.selectedRecipientIds
        .map(id => persons.value.find(p => p.id === id))
        .filter((p): p is Person => !!p)

      strategy_config = {
        List: selectedPersons,
        Timeout: phaseForm.timeout_minutes * 60 * 1000000000
      }
    } else {
      strategy_config = {
        Recipients: phaseForm.selectedRecipientIds,
        Deadline: new Date(Date.now() + 3600000).toISOString()
      }
    }

    const body: NewInvitePhase = {
      order: phaseForm.order,
      strategy_kind: phaseForm.strategy_kind,
      strategy_config: strategy_config
    }

    const response = await fetch(`/api/invites/${selectedInviteForPhases.value.id}/phases`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    if (!response.ok) throw new Error('Failed to add phase')
    await fetchPhases(selectedInviteForPhases.value.id)
    phaseForm.selectedRecipientIds = []
  } catch (err) {
    alert(err)
  } finally {
    isAddingPhase.value = false
  }
}

async function removePhase(phase: InvitePhase) {
  if (!selectedInviteForPhases.value) return
  
  const isActive = selectedInviteForPhases.value.status === 'active'
  const message = isActive 
    ? `Warning: This invite is currently ACTIVE. Deleting a phase might cause the process to immediately jump to the next phase or complete. Proceed?`
    : `Are you sure you want to remove this phase?`

  if (!confirm(message)) return

  try {
    const response = await fetch(`/api/invites/${selectedInviteForPhases.value.id}/phases/${phase.id}`, {
      method: 'DELETE'
    })
    if (!response.ok) throw new Error('Failed to remove phase')
    await fetchPhases(selectedInviteForPhases.value.id)
    await fetchData() // Update invite status in main table
  } catch (err) {
    alert(err)
  }
}

async function startInvite(invite: Invite) {
  try {
    const response = await fetch(`/api/invites/${invite.id}/start`, { method: 'POST' })
    if (!response.ok) {
      const err = await response.json()
      throw new Error(err.message || 'Failed to start')
    }
    alert('Invite process started!')
    await fetchData()
  } catch (err) {
    alert(err)
  }
}

async function openStatusModal(invite: Invite) {
  isStatusModalOpen.value = true
  loadingStatus.value = true
  try {
    const response = await fetch(`/api/invites/${invite.id}/status`)
    if (!response.ok) throw new Error('Failed to fetch status')
    statusReport.value = await response.json()
  } catch (err) {
    alert(err)
    isStatusModalOpen.value = false
  } finally {
    loadingStatus.value = false
  }
}

function copyLink(token?: string) {
  if (!token) return
  const url = `${window.location.origin}/respond/${token}`
  navigator.clipboard.writeText(url).then(() => {
    alert('Link copied to clipboard!')
  })
}

const statusCounts = computed(() => {
  if (!statusReport.value?.invitees) return { pending: 0, accepted: 0, declined: 0 }
  return statusReport.value.invitees.reduce((acc, i) => {
    acc[i.status as keyof typeof acc] = (acc[i.status as keyof typeof acc] || 0) + 1
    return acc
  }, { pending: 0, accepted: 0, declined: 0 })
})

onMounted(fetchData)
</script>

<template>
  <div>
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">Invites</h2>
        <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">Manage your multi-phase invite processes.</p>
      </div>
      <div class="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
        <button
          @click="openCreateInviteModal"
          type="button"
          class="block rounded-md bg-indigo-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
        >
          Create Invite
        </button>
      </div>
    </div>

    <!-- Invites Table -->
    <div class="mt-8 flow-root">
      <div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
          <div v-if="loading && invites.length === 0" class="text-center py-4 text-gray-500">Loading invites...</div>
          <table v-else class="min-w-full divide-y divide-gray-300 dark:divide-white/10">
            <thead>
              <tr>
                <th scope="col" class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-0">Title</th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status</th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Starts At</th>
                <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-0"><span class="sr-only">Actions</span></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-white/5">
              <tr v-for="invite in invites" :key="invite.id">
                <td class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-0">{{ invite.title }}</td>
                <td class="whitespace-nowrap px-3 py-4 text-sm">
                  <span :class="{
                    'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300': invite.status === 'pending',
                    'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': invite.status === 'active',
                    'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400': invite.status === 'completed'
                  }" class="px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wide">
                    {{ invite.status }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-400">{{ new Date(invite.from).toLocaleString() }}</td>
                <td class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-0 space-x-3">
                  <button v-if="invite.status !== 'pending'" @click="openStatusModal(invite)" class="text-green-600 hover:text-green-900 dark:text-green-400">Status</button>
                  <button v-if="invite.status === 'pending'" @click="startInvite(invite)" class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 font-bold">Start</button>
                  <button @click="openPhasesModal(invite)" class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400">Phases</button>
                  <button @click="openEditInviteModal(invite)" class="text-gray-600 hover:text-gray-900 dark:text-gray-400">Edit</button>
                  <button @click="deleteInvite(invite)" class="text-red-600 hover:text-red-900 dark:text-red-400">Delete</button>
                </td>
              </tr>
              <tr v-if="invites.length === 0 && !loading">
                <td colspan="4" class="text-center py-4 text-gray-500 italic">No invites found.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Invite Create/Edit Modal -->
    <div v-if="isInviteModalOpen" class="relative z-10" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"></div>
      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg sm:p-6">
            <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white">{{ editingInvite ? 'Edit Invite' : 'Create Invite' }}</h3>
            <div class="mt-4 space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Title</label>
                <input v-model="inviteForm.title" type="text" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Description</label>
                <textarea v-model="inviteForm.description" rows="2" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border"></textarea>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">From</label>
                <select v-model="inviteForm.from_person_id" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border">
                  <option v-for="p in persons" :key="p.id" :value="p.id">{{ p.name }} ({{ p.email }})</option>
                </select>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Starts At</label>
                <input v-model="inviteForm.from" type="datetime-local" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Ends At (Optional)</label>
                <input v-model="inviteForm.to" type="datetime-local" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border" />
              </div>
            </div>
            <div class="mt-5 sm:mt-6 sm:grid sm:grid-cols-2 sm:gap-3">
              <button @click="saveInvite" :disabled="isSavingInvite" class="inline-flex w-full justify-center rounded-md bg-indigo-600 px-4 py-2 text-white shadow-sm hover:bg-indigo-700">Save</button>
              <button @click="isInviteModalOpen = false" class="mt-3 inline-flex w-full justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-gray-700 dark:bg-gray-700 dark:text-gray-200">Cancel</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Phases Management Modal -->
    <div v-if="isPhasesModalOpen" class="relative z-10" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"></div>
      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl sm:p-6">
            <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white mb-4">Manage Phases: {{ selectedInviteForPhases?.title }}</h3>
            
            <div class="mb-8">
              <div v-if="loadingPhases" class="text-center py-4 text-gray-500">Loading...</div>
              <ul v-else-if="phases.length > 0" class="divide-y divide-gray-200 dark:divide-white/5 border dark:border-white/10 rounded-md">
                <li v-for="p in phases" :key="p.id" class="p-4 flex justify-between items-center">
                  <div>
                    <span class="font-bold mr-3">#{{ p.order }}</span>
                    <span class="uppercase text-xs bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded mr-3">{{ p.strategy_kind }}</span>
                    <span class="text-sm text-gray-500">Config: {{ JSON.stringify(p.strategy_config).slice(0, 50) }}...</span>
                  </div>
                  <button @click="removePhase(p)" class="text-red-600 hover:text-red-900 text-sm">Remove</button>
                </li>
              </ul>
              <p v-else class="text-center py-4 text-gray-500 italic">No phases added yet.</p>
            </div>

            <div class="bg-gray-50 dark:bg-gray-900/50 p-4 rounded-lg border dark:border-white/10">
              <h4 class="text-sm font-bold mb-4">Add New Phase</h4>
              <div class="grid grid-cols-2 gap-4 mb-4">
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase">Strategy</label>
                  <select v-model="phaseForm.strategy_kind" class="mt-1 block w-full rounded-md border-gray-300 dark:bg-gray-700 dark:text-white p-2 border">
                    <option value="ladder">Ladder (One by One)</option>
                    <option value="sprint">Sprint (All at once)</option>
                  </select>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 uppercase">Timeout (min)</label>
                  <input v-model="phaseForm.timeout_minutes" type="number" class="mt-1 block w-full rounded-md border-gray-300 dark:bg-gray-700 dark:text-white p-2 border" />
                </div>
              </div>

              <div class="mb-4">
                <label class="block text-xs font-medium text-gray-500 uppercase mb-2">Available Recipients</label>
                <div class="max-h-32 overflow-y-auto border dark:border-white/10 rounded p-2 bg-white dark:bg-gray-800">
                  <div v-for="p in persons" :key="p.id" class="flex items-center mb-1">
                    <input type="checkbox" :checked="phaseForm.selectedRecipientIds.includes(p.id)" @change="toggleRecipient(p.id)" class="mr-2" />
                    <span class="text-sm">{{ p.name }} (Person)</span>
                  </div>
                  <template v-if="phaseForm.strategy_kind === 'sprint'">
                    <div v-for="g in groups" :key="g.id" class="flex items-center mb-1">
                      <input type="checkbox" :checked="phaseForm.selectedRecipientIds.includes(g.id)" @change="toggleRecipient(g.id)" class="mr-2" />
                      <span class="text-sm">{{ g.name }} (Group)</span>
                    </div>
                  </template>
                </div>
              </div>

              <div v-if="phaseForm.strategy_kind === 'ladder' && phaseForm.selectedRecipientIds.length > 0" class="mb-4">
                <label class="block text-xs font-medium text-gray-500 uppercase mb-2">Recipient Priority</label>
                <ul class="border dark:border-white/10 rounded-md divide-y divide-gray-100 dark:divide-white/5 bg-white dark:bg-gray-800">
                  <li v-for="(id, idx) in phaseForm.selectedRecipientIds" :key="id" class="p-2 flex justify-between items-center text-sm">
                    <span><span class="font-bold mr-2">{{ idx + 1 }}.</span> {{ getRecipientName(id) }}</span>
                    <div class="space-x-2">
                      <button @click="moveRecipient(idx, -1)" :disabled="idx === 0" class="text-gray-400">↑</button>
                      <button @click="moveRecipient(idx, 1)" :disabled="idx === phaseForm.selectedRecipientIds.length - 1" class="text-gray-400">↓</button>
                    </div>
                  </li>
                </ul>
              </div>

              <button @click="addPhase" :disabled="isAddingPhase || phaseForm.selectedRecipientIds.length === 0" class="w-full bg-indigo-600 text-white py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50">
                Add Phase
              </button>
            </div>

            <div class="mt-6 flex justify-end">
              <button @click="isPhasesModalOpen = false" class="bg-white px-4 py-2 border rounded-md dark:bg-gray-700 dark:text-white">Close</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Status Modal -->
    <div v-if="isStatusModalOpen" class="relative z-10" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"></div>
      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl sm:p-6">
            <h3 class="text-lg font-medium leading-6 text-gray-900 dark:text-white mb-4">Invite Status</h3>
            
            <div v-if="loadingStatus" class="text-center py-4">Loading...</div>
            <div v-else-if="statusReport">
              <div class="grid grid-cols-3 gap-4 mb-8 text-center">
                <div class="bg-gray-50 dark:bg-gray-900/30 p-4 rounded-lg border dark:border-white/10">
                  <p class="text-2xl font-bold">{{ statusCounts.accepted }}</p>
                  <p class="text-xs uppercase text-green-600 font-bold">Accepted</p>
                </div>
                <div class="bg-gray-50 dark:bg-gray-900/30 p-4 rounded-lg border dark:border-white/10">
                  <p class="text-2xl font-bold">{{ statusCounts.pending }}</p>
                  <p class="text-xs uppercase text-indigo-600 font-bold">Pending</p>
                </div>
                <div class="bg-gray-50 dark:bg-gray-900/30 p-4 rounded-lg border dark:border-white/10">
                  <p class="text-2xl font-bold">{{ statusCounts.declined }}</p>
                  <p class="text-xs uppercase text-red-600 font-bold">Declined</p>
                </div>
              </div>

              <div v-if="statusReport.active_phase" class="mb-6 bg-indigo-50 dark:bg-indigo-900/20 p-3 rounded border border-indigo-100 dark:border-indigo-800">
                <label class="text-xs font-bold uppercase text-indigo-600 dark:text-indigo-400">Current Phase (#{{ statusReport.active_phase.order }})</label>
                <p class="font-medium">{{ statusReport.active_phase.progress_message }}</p>
                <p class="text-xs text-gray-500 mt-1">Strategy: {{ statusReport.active_phase.strategy_kind }}</p>
                <p v-if="statusReport.active_phase.next_check_at" class="text-xs text-gray-500">Next check: {{ new Date(statusReport.active_phase.next_check_at).toLocaleTimeString() }}</p>
              </div>

              <div class="mt-8">
                <h4 class="text-sm font-bold uppercase text-gray-500 mb-4">Recipient Details</h4>
                <div class="max-h-64 overflow-y-auto">
                  <table class="min-w-full divide-y divide-gray-200 dark:divide-white/5">
                    <thead>
                      <tr class="text-left text-xs text-gray-400 uppercase">
                        <th class="pb-2">Name</th>
                        <th class="pb-2 text-center">Status</th>
                        <th class="pb-2 text-right">Link</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-100 dark:divide-white/5">
                      <tr v-for="p in statusReport.invitees" :key="p.id" class="py-3">
                        <td class="py-3">
                          <p class="text-sm font-medium">{{ p.name }}</p>
                          <p class="text-xs text-gray-500">{{ p.email }}</p>
                        </td>
                        <td class="py-3 text-center">
                          <span :class="{
                            'text-green-500 font-bold': p.status === 'accepted',
                            'text-red-500 font-bold': p.status === 'declined',
                            'text-gray-400': p.status === 'pending'
                          }" class="text-xs uppercase px-2 py-1 rounded-full bg-gray-100 dark:bg-gray-700/50">
                            {{ p.status }}
                          </span>
                        </td>
                        <td class="py-3 text-right">
                          <button v-if="p.status === 'pending'" @click="copyLink(p.magic_token)" class="text-xs bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400 px-2 py-1 rounded">Copy Link</button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>

            <div class="mt-6 flex justify-between gap-4">
              <button @click="isStatusModalOpen = false" class="w-full bg-gray-100 dark:bg-gray-700 py-2 rounded-md">Close</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
