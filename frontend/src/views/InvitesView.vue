<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import type { components } from '../api-types'
import { client } from '@/utils/api'
import { notify } from '@/utils/toast'
import { useConfirm } from '@/composables/useConfirm'
import TableSkeleton from '@/components/TableSkeleton.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'

type Invite = components['schemas']['Invite']
type InvitePhase = components['schemas']['InvitePhase']
type NewInvitePhase = components['schemas']['NewInvitePhase']
type InviteStatusReport = components['schemas']['InviteStatusReport']
type Person = components['schemas']['Person']
type Group = components['schemas']['Group']
type Tag = components['schemas']['Tag']

const { confirm } = useConfirm()

const invites = ref<Invite[]>([])
const persons = ref<Person[]>([])
const groups = ref<Group[]>([])
const tags = ref<Tag[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const startingInvites = reactive<Record<string, boolean>>({})

// Invite Modal (Wizard)
const isInviteModalOpen = ref(false)
const isSavingInvite = ref(false)
const editingInvite = ref<Invite | null>(null)
const wizardStep = ref(1)
const inviteForm = reactive({
  title: '',
  description: '',
  from: new Date(Date.now() + 86400000).toISOString().slice(0, 16), // Tomorrow
  to: '',
  from_person_id: '',
  tag_ids: [] as string[],
  phases: [] as any[], // Draft phases for new invites
})

// Current phase being added in Step 2
const draftPhaseForm = reactive({
  strategy_kind: 'ladder' as 'ladder' | 'sprint',
  timeout_minutes: 60,
  selectedRecipientIds: [] as string[],
})

// Phases Modal (for existing invites)
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

// Recipient Search & Filtering
const recipientSearchQuery = ref('')

const unifiedRecipients = computed(() => {
  const all: any[] = [
    ...persons.value.map(p => ({ ...p, type: 'person' })),
    ...groups.value.map(g => ({ ...g, type: 'group' }))
  ]
  
  if (!recipientSearchQuery.value) return all
  
  const query = recipientSearchQuery.value.toLowerCase()
  return all.filter(r => 
    r.name.toLowerCase().includes(query) || 
    (r.email && r.email.toLowerCase().includes(query)) ||
    r.type === query
  )
})

const selectedChips = computed(() => {
  // Use either the wizard draft form or the standalone phase modal form
  const currentIds = isInviteModalOpen.value 
    ? draftPhaseForm.selectedRecipientIds 
    : phaseForm.selectedRecipientIds
    
  return currentIds.map(id => {
    const p = persons.value.find(p => p.id === id)
    if (p) return { id, name: p.name, type: 'person' }
    const g = groups.value.find(g => g.id === id)
    if (g) return { id, name: g.name, type: 'group' }
    return { id, name: 'Unknown', type: 'unknown' }
  })
})

async function fetchData() {
  loading.value = true
  try {
    const [invitesRes, personsRes, groupsRes, tagsRes] = await Promise.all([
      client.GET('/invites'),
      client.GET('/persons'),
      client.GET('/groups'),
      client.GET('/tags')
    ])
    
    if (invitesRes.error) throw new Error('Failed to fetch invites')
    invites.value = invitesRes.data || []
    persons.value = personsRes.data || []
    groups.value = groupsRes.data || []
    tags.value = tagsRes.data || []
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
  if (item) {
    phaseForm.selectedRecipientIds.splice(newIndex, 0, item)
  }
}

function toggleRecipient(id: string) {
  const currentIds = isInviteModalOpen.value 
    ? draftPhaseForm.selectedRecipientIds 
    : phaseForm.selectedRecipientIds
    
  const index = currentIds.indexOf(id)
  if (index > -1) {
    currentIds.splice(index, 1)
  } else {
    currentIds.push(id)
  }
}

function toggleTag(id: string) {
  const index = inviteForm.tag_ids.indexOf(id)
  if (index > -1) {
    inviteForm.tag_ids.splice(index, 1)
  } else {
    inviteForm.tag_ids.push(id)
  }
}

function openCreateInviteModal() {
  editingInvite.value = null
  wizardStep.value = 1
  inviteForm.title = ''
  inviteForm.description = ''
  inviteForm.from = new Date(Date.now() + 86400000).toISOString().slice(0, 16)
  inviteForm.to = ''
  inviteForm.tag_ids = []
  inviteForm.phases = []
  // Default to first person if found
  const firstPerson = persons.value[0]
  inviteForm.from_person_id = firstPerson ? firstPerson.id : ''
  isInviteModalOpen.value = true
}

function openEditInviteModal(invite: Invite) {
  editingInvite.value = invite
  wizardStep.value = 1 // Basic edit is just Step 1
  inviteForm.title = invite.title
  inviteForm.description = invite.description || ''
  inviteForm.from = new Date(invite.from).toISOString().slice(0, 16)
  inviteForm.to = invite.to ? new Date(invite.to).toISOString().slice(0, 16) : ''
  inviteForm.from_person_id = invite.from_person_id
  inviteForm.tag_ids = (invite.tags || []).map(t => t.id)
  inviteForm.phases = []
  isInviteModalOpen.value = true
}

function addDraftPhase() {
  if (draftPhaseForm.selectedRecipientIds.length === 0) {
    notify.error('Please select at least one recipient')
    return
  }

  let strategy_config: Record<string, any> = {}
  if (draftPhaseForm.strategy_kind === 'ladder') {
    const selectedPersons = draftPhaseForm.selectedRecipientIds
      .map(id => persons.value.find(p => p.id === id))
      .filter((p): p is Person => !!p)

    strategy_config = {
      List: selectedPersons,
      Timeout: draftPhaseForm.timeout_minutes * 60 * 1000000000
    }
  } else {
    strategy_config = {
      Recipients: draftPhaseForm.selectedRecipientIds,
      Deadline: new Date(Date.now() + 3600000).toISOString()
    }
  }

  inviteForm.phases.push({
    order: inviteForm.phases.length + 1,
    strategy_kind: draftPhaseForm.strategy_kind,
    strategy_config,
    _display: formatStrategyConfig({ 
      strategy_kind: draftPhaseForm.strategy_kind, 
      strategy_config 
    } as any)
  })

  // Reset draft form
  draftPhaseForm.selectedRecipientIds = []
}

function removeDraftPhase(index: number) {
  inviteForm.phases.splice(index, 1)
  // Re-order
  inviteForm.phases.forEach((p, i) => p.order = i + 1)
}

async function saveInvite() {
  if (wizardStep.value === 1 && !editingInvite.value) {
    if (!inviteForm.title || !inviteForm.from || !inviteForm.from_person_id) {
      notify.error('Please fill in all required fields')
      return
    }
    wizardStep.value = 2
    return
  }
  if (wizardStep.value === 2 && !editingInvite.value) {
    if (inviteForm.phases.length === 0) {
      notify.error('Please add at least one phase')
      return
    }
    wizardStep.value = 3
    return
  }

  isSavingInvite.value = true
  try {
    const body: any = {
      title: inviteForm.title,
      description: inviteForm.description,
      from: new Date(inviteForm.from).toISOString(),
      from_person_id: inviteForm.from_person_id,
      tag_ids: inviteForm.tag_ids,
      to: inviteForm.to ? new Date(inviteForm.to).toISOString() : undefined,
    }

    // Include phases for deep create
    if (!editingInvite.value) {
      body.phases = inviteForm.phases.map(p => ({
        order: p.order,
        strategy_kind: p.strategy_kind,
        strategy_config: p.strategy_config
      }))
    }

    let error: any
    if (editingInvite.value) {
      const res = await client.PATCH('/invites/{id}', {
        params: { path: { id: editingInvite.value.id } },
        body
      })
      error = res.error
    } else {
      const res = await client.POST('/invites', {
        body
      })
      error = res.error
    }

    if (error) throw error
    await fetchData()
    isInviteModalOpen.value = false
    notify.success(editingInvite.value ? 'Invite updated' : 'Invite created')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
  } finally {
    isSavingInvite.value = false
  }
}

async function deleteInvite(invite: Invite) {
  const message = invite.status === 'active'
    ? `Warning: This invite is currently ACTIVE. Deleting it will stop all pending notifications and invalidate all magic links. Proceed?`
    : `Delete invite "${invite.title}"?`
    
  if (!await confirm({
    title: 'Delete Invite',
    message,
    variant: 'danger',
    confirmLabel: 'Delete'
  })) return
  try {
    const { error } = await client.DELETE('/invites/{id}', {
      params: { path: { id: invite.id } }
    })
    if (error) throw error
    await fetchData()
    notify.success('Invite deleted')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
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
    const { data, error } = await client.GET('/invites/{id}/phases', {
      params: { path: { id: inviteId } }
    })
    if (error) throw error
    phases.value = data || []
    phaseForm.order = phases.value.length + 1
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
  } finally {
    loadingPhases.value = false
  }
}

async function addPhase() {
  if (!selectedInviteForPhases.value) return
  isAddingPhase.value = true
  try {
    let strategy_config: Record<string, unknown> = {}
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

    const { error } = await client.POST('/invites/{id}/phases', {
      params: { path: { id: selectedInviteForPhases.value.id } },
      body: {
        order: phaseForm.order,
        strategy_kind: phaseForm.strategy_kind,
        strategy_config: strategy_config as any
      }
    })

    if (error) throw error
    await fetchPhases(selectedInviteForPhases.value.id)
    phaseForm.selectedRecipientIds = []
    notify.success('Phase added')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
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

  if (!await confirm({
    title: 'Remove Phase',
    message,
    variant: 'danger',
    confirmLabel: 'Remove'
  })) return

  try {
    const { error } = await client.DELETE('/invites/{id}/phases/{phase_id}', {
      params: { path: { id: selectedInviteForPhases.value.id, phase_id: phase.id } }
    })
    if (error) throw error
    await fetchPhases(selectedInviteForPhases.value.id)
    await fetchData() // Update invite status in main table
    notify.success('Phase removed')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
  }
}

async function startInvite(invite: Invite) {
  startingInvites[invite.id] = true
  try {
    const { error } = await client.POST('/invites/{id}/start', {
      params: { path: { id: invite.id } }
    })
    if (error) throw error
    notify.success('Invite process started!')
    await fetchData()
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
  } finally {
    delete startingInvites[invite.id]
  }
}

async function openStatusModal(invite: Invite) {
  isStatusModalOpen.value = true
  loadingStatus.value = true
  try {
    const { data, error } = await client.GET('/invites/{id}/status', {
      params: { path: { id: invite.id } }
    })
    if (error) throw error
    statusReport.value = data || null
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
    isStatusModalOpen.value = false
  } finally {
    loadingStatus.value = false
  }
}

function copyLink(token?: string) {
  if (!token) return
  const url = `${window.location.origin}/respond/${token}`
  navigator.clipboard.writeText(url).then(() => {
    notify.success('Link copied to clipboard!')
  })
}

function formatStrategyConfig(phase: InvitePhase) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const config = phase.strategy_config as any
  if (phase.strategy_kind === 'ladder') {
    const count = (config.List || []).length
    const timeout = Math.round((config.Timeout || 0) / 60000000000)
    return `Ladder: ${count} persons, ${timeout}m timeout`
  } else if (phase.strategy_kind === 'sprint') {
    const count = (config.Recipients || []).length
    const deadline = config.Deadline ? new Date(config.Deadline).toLocaleString() : 'N/A'
    return `Sprint: ${count} recipients, deadline ${deadline}`
  }
  return JSON.stringify(phase.strategy_config)
}

async function retryEmail(emailId: string) {
  try {
    const response = await fetch(`/api/emails/${emailId}/retry`, { method: 'POST' })
    if (!response.ok) throw new Error('Failed to retry email')
    
    // Refresh status report
    const inviteId = statusReport.value?.invite_id
    if (inviteId) {
      const res = await fetch(`/api/invites/${inviteId}/status`)
      if (res.ok) statusReport.value = await res.json()
    }
    notify.success('Email retry queued')
  } catch (err) {
    notify.error(err instanceof Error ? err.message : String(err))
  }
}

const statusCounts = computed(() => {
  if (!statusReport.value?.invitees) return { pending: 0, accepted: 0, declined: 0 }
  return statusReport.value.invitees.reduce((acc, i) => {
    acc[i.status as keyof typeof acc] = (acc[i.status as keyof typeof acc] || 0) + 1
    return acc
  }, { pending: 0, accepted: 0, declined: 0 })
})

const groupedInvitees = computed(() => {
  if (!statusReport.value?.invitees) return {}
  const groups: Record<string, typeof statusReport.value.invitees> = {}
  
  for (const invitee of statusReport.value.invitees) {
    const key = invitee.phase_order !== undefined && invitee.phase_order !== null 
      ? String(invitee.phase_order) 
      : 'unassigned'
      
    if (!groups[key]) groups[key] = []
    groups[key].push(invitee)
  }
  
  return groups
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

    <!-- Invites List -->
    <div class="mt-8">
      <div v-if="loading && invites.length === 0" class="space-y-4">
        <TableSkeleton :columns="3" class="hidden sm:block" />
        <div v-for="i in 3" :key="i" class="h-32 bg-gray-100 dark:bg-white/5 animate-pulse rounded-xl sm:hidden"></div>
      </div>
      
      <div v-else-if="invites.length === 0" class="text-center py-12 bg-white dark:bg-white/5 rounded-xl border border-dashed border-gray-300 dark:border-white/10">
        <p class="text-gray-500 italic">No invitations found. Start by creating one!</p>
      </div>

      <div v-else>
        <!-- Desktop Table -->
        <div class="hidden sm:block overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-300 dark:divide-white/10">
            <thead>
              <tr>
                <th scope="col" class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-0">Title</th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Tags</th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status</th>
                <th scope="col" class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Starts At</th>
                <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-0"><span class="sr-only">Actions</span></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-white/5">
              <tr v-for="invite in invites" :key="invite.id">
                <td class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-0">{{ invite.title }}</td>
                <td class="whitespace-nowrap px-3 py-4 text-sm">
                  <div class="flex flex-wrap gap-1">
                    <span v-for="tag in invite.tags" :key="tag.id" 
                      :style="{ backgroundColor: tag.color + '20', color: tag.color, borderColor: tag.color + '40' }"
                      class="px-2 py-0.5 rounded text-[10px] font-medium border uppercase tracking-wider">
                      {{ tag.name }}
                    </span>
                  </div>
                </td>
                <td class="whitespace-nowrap px-3 py-4 text-sm">
                  <div class="flex flex-col gap-1.5">
                    <span :class="{
                      'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300': invite.status === 'pending',
                      'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': invite.status === 'active',
                      'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400': invite.status === 'completed'
                    }" class="inline-block px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wide w-fit">
                      {{ invite.status }}
                    </span>
                    
                    <!-- Micro-Timeline -->
                    <div v-if="invite.progress && invite.progress.total_phases > 0" class="w-24">
                      <div class="flex justify-between items-center mb-0.5">
                        <span class="text-[8px] font-bold text-gray-400 uppercase">
                          Ph {{ invite.progress.active_phase_order || (invite.status === 'completed' ? invite.progress.total_phases : 0) }}/{{ invite.progress.total_phases }}
                        </span>
                        <span class="text-[8px] font-bold text-indigo-500">
                          {{ Math.round((invite.progress.total_accepted / (invite.progress.total_invitees || 1)) * 100) }}%
                        </span>
                      </div>
                      <div class="h-1 w-full bg-gray-100 dark:bg-white/5 rounded-full flex overflow-hidden border dark:border-white/5">
                        <div v-for="pOrder in invite.progress.total_phases" :key="pOrder"
                             class="h-full border-r last:border-r-0 border-white/20"
                             :style="{ width: `${100 / invite.progress.total_phases}%` }"
                             :class="{
                               'bg-emerald-500': pOrder < invite.progress.active_phase_order || (invite.status === 'completed'),
                               'bg-indigo-500 animate-pulse': pOrder === invite.progress.active_phase_order && invite.status === 'active',
                               'bg-gray-200 dark:bg-gray-800': (pOrder > invite.progress.active_phase_order && invite.status === 'active') || invite.status === 'pending'
                             }"
                        ></div>
                      </div>
                    </div>
                  </div>
                </td>
                <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-400">{{ new Date(invite.from).toLocaleString() }}</td>
                <td class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-0 space-x-3">
                  <button v-if="invite.status !== 'pending'" @click="openStatusModal(invite)" class="text-green-600 hover:text-green-900 dark:text-green-400">Status</button>
                  <button v-if="invite.status === 'pending'" @click="startInvite(invite)" :disabled="!!startingInvites[invite.id]" class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 font-bold inline-flex items-center gap-1">
                    <LoadingSpinner v-if="startingInvites[invite.id]" size="sm" />
                    <span>Start</span>
                  </button>
                  <button @click="openPhasesModal(invite)" class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400">Phases</button>
                  <button @click="openEditInviteModal(invite)" class="text-gray-600 hover:text-gray-900 dark:text-gray-400">Edit</button>
                  <button @click="deleteInvite(invite)" class="text-red-600 hover:text-red-900 dark:text-red-400">Delete</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Mobile Cards -->
        <div class="sm:hidden space-y-4">
          <div v-for="invite in invites" :key="invite.id" class="bg-white dark:bg-white/5 rounded-xl border border-gray-200 dark:border-white/10 p-4 shadow-sm">
            <div class="flex justify-between items-start mb-3">
              <h4 class="text-base font-bold text-gray-900 dark:text-white leading-tight pr-2">{{ invite.title }}</h4>
              <div class="flex flex-col items-end gap-1.5">
                <span :class="{
                  'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300': invite.status === 'pending',
                  'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': invite.status === 'active',
                  'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400': invite.status === 'completed'
                }" class="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider shrink-0">
                  {{ invite.status }}
                </span>
                
                <!-- Micro-Timeline -->
                <div v-if="invite.progress && invite.progress.total_phases > 0" class="w-20">
                  <div class="h-1 w-full bg-gray-100 dark:bg-white/5 rounded-full flex overflow-hidden border dark:border-white/5">
                    <div v-for="pOrder in invite.progress.total_phases" :key="pOrder"
                         class="h-full border-r last:border-r-0 border-white/20"
                         :style="{ width: `${100 / invite.progress.total_phases}%` }"
                         :class="{
                           'bg-emerald-500': pOrder < invite.progress.active_phase_order || (invite.status === 'completed'),
                           'bg-indigo-500 animate-pulse': pOrder === invite.progress.active_phase_order && invite.status === 'active',
                           'bg-gray-200 dark:bg-gray-800': (pOrder > invite.progress.active_phase_order && invite.status === 'active') || invite.status === 'pending'
                         }"
                    ></div>
                  </div>
                </div>
              </div>
            </div>
            
            <div class="flex flex-wrap gap-1 mb-4">
              <span v-for="tag in invite.tags" :key="tag.id" 
                :style="{ backgroundColor: tag.color + '15', color: tag.color, borderColor: tag.color + '30' }"
                class="px-2 py-0.5 rounded text-[10px] font-medium border uppercase tracking-wider">
                {{ tag.name }}
              </span>
            </div>

            <p class="text-xs text-gray-500 mb-5">
              Starts: {{ new Date(invite.from).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' }) }}
            </p>

            <div class="grid grid-cols-2 gap-2">
              <button v-if="invite.status !== 'pending'" @click="openStatusModal(invite)" class="py-2.5 rounded-lg bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400 text-xs font-bold uppercase">Status</button>
              <button v-if="invite.status === 'pending'" @click="startInvite(invite)" :disabled="!!startingInvites[invite.id]" class="py-2.5 rounded-lg bg-indigo-600 text-white text-xs font-bold uppercase disabled:opacity-50 flex items-center justify-center gap-2">
                <LoadingSpinner v-if="startingInvites[invite.id]" size="sm" />
                <span>Start</span>
              </button>
              <button @click="openPhasesModal(invite)" class="py-2.5 rounded-lg bg-gray-50 dark:bg-white/5 text-gray-700 dark:text-gray-300 border dark:border-white/5 text-xs font-bold uppercase">Phases</button>
              <button @click="openEditInviteModal(invite)" class="py-2.5 rounded-lg bg-gray-50 dark:bg-white/5 text-gray-700 dark:text-gray-300 border dark:border-white/5 text-xs font-bold uppercase">Edit</button>
              <button @click="deleteInvite(invite)" class="py-2.5 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-xs font-bold uppercase col-span-2">Delete</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Invite Create/Edit Modal (Wizard) -->
    <div v-if="isInviteModalOpen" class="relative z-10" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500/40 backdrop-blur-sm transition-opacity"></div>
      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div class="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 px-4 pt-5 pb-4 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-xl sm:p-6">
            
            <div class="flex justify-between items-center mb-6">
              <h3 class="text-xl font-bold text-gray-900 dark:text-white">
                {{ editingInvite ? 'Edit Invite' : 'Create Invitation' }}
              </h3>
              <div v-if="!editingInvite" class="flex items-center gap-2">
                <div v-for="step in 3" :key="step" 
                  class="h-2 w-12 rounded-full transition-colors"
                  :class="wizardStep >= step ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-700'"
                ></div>
              </div>
            </div>

            <!-- Step 1: Details -->
            <div v-if="wizardStep === 1" class="space-y-4">
              <div>
                <label for="invite-title" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Title</label>
                <input id="invite-title" v-model="inviteForm.title" type="text" placeholder="e.g. Annual Design Conference 2026" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border" />
              </div>
              <div>
                <label for="invite-description" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Description</label>
                <textarea id="invite-description" v-model="inviteForm.description" rows="2" placeholder="Tell recipients about the event..." class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border"></textarea>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tags</label>
                <div class="flex flex-wrap gap-2">
                  <button v-for="tag in tags" :key="tag.id"
                    @click="toggleTag(tag.id)"
                    type="button"
                    :style="{ 
                      backgroundColor: inviteForm.tag_ids.includes(tag.id) ? tag.color + '20' : 'transparent',
                      color: inviteForm.tag_ids.includes(tag.id) ? tag.color : 'inherit',
                      borderColor: inviteForm.tag_ids.includes(tag.id) ? tag.color : '#d1d5db'
                    }"
                    class="px-3 py-1 rounded text-xs font-medium border transition-colors hover:bg-gray-50 dark:hover:bg-gray-700"
                    :class="inviteForm.tag_ids.includes(tag.id) ? '' : 'text-gray-600 dark:text-gray-400 border-gray-300 dark:border-gray-600'">
                    {{ tag.name }}
                  </button>
                </div>
              </div>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label for="invite-from-person" class="block text-sm font-medium text-gray-700 dark:text-gray-300">From</label>
                  <select id="invite-from-person" v-model="inviteForm.from_person_id" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border">
                    <option v-for="p in persons" :key="p.id" :value="p.id">{{ p.name }}</option>
                  </select>
                </div>
                <div>
                  <label for="invite-from-at" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Starts At</label>
                  <input id="invite-from-at" v-model="inviteForm.from" type="datetime-local" class="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm p-2 border" />
                </div>
              </div>
            </div>

            <!-- Step 2: Phases -->
            <div v-else-if="wizardStep === 2" class="space-y-6">
              <div v-if="inviteForm.phases.length > 0" class="space-y-2">
                <p class="text-xs font-bold text-gray-500 uppercase">Workflow Sequence</p>
                <div v-for="(p, idx) in inviteForm.phases" :key="idx" class="flex items-center justify-between p-3 bg-gray-50 dark:bg-white/5 rounded-lg border dark:border-white/10">
                  <div class="flex items-center">
                    <span class="size-6 rounded-full bg-indigo-600 text-white text-[10px] flex items-center justify-center font-bold mr-3">{{ p.order }}</span>
                    <div>
                      <p class="text-sm font-bold text-gray-900 dark:text-white uppercase">{{ p.strategy_kind }}</p>
                      <p class="text-xs text-gray-500">{{ p._display }}</p>
                    </div>
                  </div>
                  <button @click="removeDraftPhase(idx)" class="text-gray-400 hover:text-red-500 transition-colors">
                    <svg class="size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path d="M6 18L18 6M6 6l12 12" stroke-width="2" /></svg>
                  </button>
                </div>
              </div>

              <div class="bg-indigo-50 dark:bg-indigo-900/20 p-4 rounded-xl border border-indigo-100 dark:border-indigo-800">
                <h4 class="text-sm font-bold text-indigo-900 dark:text-indigo-300 mb-4">Add Process Step</h4>
                <div class="grid grid-cols-2 gap-4 mb-4">
                  <div>
                    <label class="block text-xs font-medium text-indigo-700 dark:text-indigo-400 uppercase mb-1">Strategy</label>
                    <select v-model="draftPhaseForm.strategy_kind" class="block w-full rounded-md border-gray-300 dark:bg-gray-800 dark:text-white text-sm p-2 border">
                      <option value="ladder">Ladder (One by One)</option>
                      <option value="sprint">Sprint (All at once)</option>
                    </select>
                  </div>
                  <div>
                    <label class="block text-xs font-medium text-indigo-700 dark:text-indigo-400 uppercase mb-1">Timeout (min)</label>
                    <input v-model="draftPhaseForm.timeout_minutes" type="number" class="block w-full rounded-md border-gray-300 dark:bg-gray-800 dark:text-white text-sm p-2 border" />
                  </div>
                </div>

                <div class="mb-4">
                  <label class="block text-xs font-medium text-indigo-700 dark:text-indigo-400 uppercase mb-2">Recipients Selection</label>
                  
                  <!-- Search Bar -->
                  <div class="relative mb-3">
                    <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <svg class="size-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" stroke-width="2" /></svg>
                    </div>
                    <input type="text" v-model="recipientSearchQuery" placeholder="Search by name or email..." 
                      class="block w-full pl-10 pr-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md leading-5 bg-white dark:bg-gray-900 text-sm placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500" />
                  </div>

                  <!-- Chips Tray -->
                  <div v-if="selectedChips.length > 0" class="flex flex-wrap gap-2 mb-3 max-h-20 overflow-y-auto p-1">
                    <span v-for="chip in selectedChips" :key="chip.id" 
                      class="inline-flex items-center gap-1 px-2 py-1 rounded-md text-[10px] font-bold uppercase transition-all shadow-sm"
                      :class="chip.type === 'group' ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300' : 'bg-gray-100 text-gray-700 dark:bg-white/10 dark:text-gray-300'">
                      {{ chip.name }}
                      <button @click="toggleRecipient(chip.id)" class="hover:text-red-500 ml-1">×</button>
                    </span>
                  </div>

                  <!-- Filtered List -->
                  <div class="max-h-40 overflow-y-auto border dark:border-white/10 rounded-lg bg-white dark:bg-gray-900 divide-y dark:divide-white/5">
                    <div v-for="r in unifiedRecipients" :key="r.id" 
                      @click="toggleRecipient(r.id)"
                      class="flex items-center justify-between p-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
                      :class="draftPhaseForm.selectedRecipientIds.includes(r.id) ? 'bg-indigo-50 dark:bg-indigo-900/20' : ''">
                      <div class="flex flex-col">
                        <span class="text-sm font-medium" :class="draftPhaseForm.selectedRecipientIds.includes(r.id) ? 'text-indigo-600 dark:text-indigo-400' : ''">{{ r.name }}</span>
                        <span class="text-[10px] text-gray-500">{{ r.type === 'group' ? 'Group' : r.email }}</span>
                      </div>
                      <div v-if="draftPhaseForm.selectedRecipientIds.includes(r.id)" class="text-indigo-600">
                        <svg class="size-4" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                      </div>
                    </div>
                    <div v-if="unifiedRecipients.length === 0" class="p-4 text-center text-xs text-gray-500 italic">
                      No recipients match your search.
                    </div>
                  </div>
                </div>

                <button @click="addDraftPhase" class="w-full bg-indigo-600 text-white py-2 rounded-md font-bold text-sm hover:bg-indigo-700 transition-colors">
                  Include this Phase
                </button>
              </div>
            </div>

            <!-- Step 3: Review -->
            <div v-else-if="wizardStep === 3" class="space-y-6">
              <div class="bg-gray-50 dark:bg-white/5 rounded-xl p-6 border dark:border-white/10">
                <h4 class="text-sm font-bold uppercase text-gray-500 mb-4 tracking-wider">Final Preview</h4>
                <div class="space-y-4">
                  <div>
                    <p class="text-xs text-gray-500 uppercase">Invitation</p>
                    <p class="text-lg font-bold text-gray-900 dark:text-white">{{ inviteForm.title }}</p>
                  </div>
                  <div class="flex justify-between border-t dark:border-white/5 pt-4">
                    <div>
                      <p class="text-xs text-gray-500 uppercase">Process</p>
                      <p class="text-sm font-medium">{{ inviteForm.phases.length }} Phases Total</p>
                    </div>
                    <div class="text-right">
                      <p class="text-xs text-gray-500 uppercase">Starts On</p>
                      <p class="text-sm font-medium">{{ new Date(inviteForm.from).toLocaleDateString() }}</p>
                    </div>
                  </div>
                </div>
              </div>
              <div class="p-4 bg-yellow-50 dark:bg-yellow-900/10 rounded-lg border border-yellow-100 dark:border-yellow-900/30">
                <p class="text-xs text-yellow-800 dark:text-yellow-400">
                  <strong>Notice:</strong> This invitation will remain in <span class="uppercase font-bold">pending</span> status until you manually click "Start" from the main list.
                </p>
              </div>
            </div>

            <!-- Navigation -->
            <div class="mt-8 flex justify-between gap-3">
              <button @click="isInviteModalOpen = false" class="px-4 py-2 text-sm font-medium text-gray-500 hover:text-gray-700">Cancel</button>
              <div class="flex gap-3">
                <button v-if="wizardStep > 1 && !editingInvite" @click="wizardStep--" class="px-6 py-2 rounded-md border border-gray-300 dark:border-white/10 text-gray-700 dark:text-gray-200 font-bold text-sm">Back</button>
                <button @click="saveInvite" :disabled="isSavingInvite" class="inline-flex justify-center items-center gap-2 rounded-md bg-indigo-600 px-8 py-2 text-white font-bold text-sm shadow-lg hover:bg-indigo-700 disabled:opacity-50 transition-all">
                  <LoadingSpinner v-if="isSavingInvite" size="sm" />
                  {{ editingInvite ? 'Save Changes' : (wizardStep === 3 ? 'Create Process' : 'Continue') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Phases Management Modal -->
    <div v-if="isPhasesModalOpen" class="relative z-10" role="dialog" aria-modal="true">
      <div class="fixed inset-0 bg-gray-500/40 backdrop-blur-sm transition-opacity"></div>
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
                    <span class="text-sm text-gray-500">{{ formatStrategyConfig(p) }}</span>
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
                <label class="block text-xs font-medium text-gray-500 uppercase mb-2">Recipients Selection</label>
                
                <!-- Search Bar -->
                <div class="relative mb-3">
                  <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <svg class="size-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" stroke-width="2" /></svg>
                  </div>
                  <input type="text" v-model="recipientSearchQuery" placeholder="Search by name or email..." 
                    class="block w-full pl-10 pr-3 py-2 border border-gray-300 dark:border-gray-700 rounded-md leading-5 bg-white dark:bg-gray-900 text-sm placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500" />
                </div>

                <!-- Chips Tray -->
                <div v-if="selectedChips.length > 0" class="flex flex-wrap gap-2 mb-3 max-h-20 overflow-y-auto p-1">
                  <span v-for="chip in selectedChips" :key="chip.id" 
                    class="inline-flex items-center gap-1 px-2 py-1 rounded-md text-[10px] font-bold uppercase transition-all shadow-sm"
                    :class="chip.type === 'group' ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300' : 'bg-gray-100 text-gray-700 dark:bg-white/10 dark:text-gray-300'">
                    {{ chip.name }}
                    <button @click="toggleRecipient(chip.id)" class="hover:text-red-500 ml-1">×</button>
                  </span>
                </div>

                <!-- Filtered List -->
                <div class="max-h-32 overflow-y-auto border dark:border-white/10 rounded-lg bg-white dark:bg-gray-900 divide-y dark:divide-white/5">
                  <div v-for="r in unifiedRecipients" :key="r.id" 
                    @click="toggleRecipient(r.id)"
                    class="flex items-center justify-between p-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
                    :class="phaseForm.selectedRecipientIds.includes(r.id) ? 'bg-indigo-50 dark:bg-indigo-900/20' : ''">
                    <div class="flex flex-col">
                      <span class="text-sm font-medium" :class="phaseForm.selectedRecipientIds.includes(r.id) ? 'text-indigo-600 dark:text-indigo-400' : ''">{{ r.name }}</span>
                      <span class="text-[10px] text-gray-500">{{ r.type === 'group' ? 'Group' : r.email }}</span>
                    </div>
                    <div v-if="phaseForm.selectedRecipientIds.includes(r.id)" class="text-indigo-600">
                      <svg class="size-4" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                    </div>
                  </div>
                  <div v-if="unifiedRecipients.length === 0" class="p-4 text-center text-xs text-gray-500 italic">
                    No recipients match your search.
                  </div>
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
      <div class="fixed inset-0 bg-gray-500/40 backdrop-blur-sm transition-opacity"></div>
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
                <div class="max-h-64 overflow-y-auto pr-2 space-y-6">
                  <div v-for="(group, phaseKey) in groupedInvitees" :key="phaseKey">
                    <h5 class="text-xs font-bold uppercase text-gray-700 dark:text-gray-300 mb-2 bg-gray-100 dark:bg-gray-700/50 p-2 rounded">
                      {{ phaseKey === 'unassigned' ? 'Unassigned' : `Phase #${phaseKey}` }}
                    </h5>
                    <table class="min-w-full divide-y divide-gray-200 dark:divide-white/5">
                      <thead>
                        <tr class="text-left text-xs text-gray-400 uppercase">
                          <th class="pb-2 pl-2">Name</th>
                          <th class="pb-2 text-center">Invite Status</th>
                          <th class="pb-2 text-center">Email Status</th>
                          <th class="pb-2 text-right pr-2">Actions</th>
                        </tr>
                      </thead>
                      <tbody class="divide-y divide-gray-100 dark:divide-white/5">
                        <tr v-for="p in group" :key="p.id" class="py-3">
                          <td class="py-3 pl-2">
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
                          <td class="py-3 text-center">
                            <div v-if="p.email_status" class="flex flex-col items-center">
                              <span :class="{
                                'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400': p.email_status === 'sent',
                                'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400': p.email_status === 'failed' && (p.email_attempts || 0) < 3,
                                'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400': p.email_status === 'failed' && (p.email_attempts || 0) >= 3,
                                'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300': p.email_status === 'pending'
                              }" class="px-2 py-0.5 rounded-full text-[10px] font-medium uppercase tracking-wide">
                                {{ p.email_status === 'failed' && (p.email_attempts || 0) < 3 ? 'Retrying' : p.email_status }}
                              </span>
                              <span v-if="p.email_error" class="text-[9px] text-red-500 mt-1 max-w-[100px] truncate" :title="p.email_error">{{ p.email_error }}</span>
                            </div>
                            <span v-else class="text-xs text-gray-400">-</span>
                          </td>
                          <td class="py-3 text-right space-x-2 pr-2">
                            <button v-if="p.email_status === 'failed' && p.email_id" @click="retryEmail(p.email_id)" class="text-xs bg-orange-50 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400 px-2 py-1 rounded">Retry</button>
                            <button v-if="p.status === 'pending'" @click="copyLink(p.magic_token)" class="text-xs bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400 px-2 py-1 rounded">Copy Link</button>
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
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
