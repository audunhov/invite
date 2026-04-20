# UX Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement professional toast notifications, custom confirmation modals, and skeleton loading screens.

**Architecture:**
- **Toasts:** Integrate `vue-sonner` for non-blocking notifications.
- **Confirmations:** Implement a custom `ConfirmModal` component and a Promise-based `useConfirm` composable.
- **Loading UI:** Create `TableSkeleton` and `ButtonSpinner` components using Tailwind's `animate-pulse`.

**Tech Stack:** Vue 3, Tailwind CSS, `vue-sonner`.

---

### Task 1: Notification Infrastructure

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/src/App.vue`
- Create: `frontend/src/utils/toast.ts`

- [ ] **Step 1: Install vue-sonner**
Run: `npm install vue-sonner` in the `frontend` directory.

- [ ] **Step 2: Initialize Toaster in App.vue**
Import and add `<Toaster />` to the root template.
```vue
<script setup lang="ts">
import { Toaster } from 'vue-sonner'
// ...
</script>
<template>
  <Toaster position="top-right" richColors />
  <!-- ... -->
</template>
```

- [ ] **Step 3: Create Toast Utility**
Create `src/utils/toast.ts` as a clean wrapper.
```typescript
import { toast } from 'vue-sonner'
export const notify = {
  success: (msg: string) => toast.success(msg),
  error: (msg: string) => toast.error(msg, { duration: Infinity }),
  info: (msg: string) => toast.info(msg),
}
```

- [ ] **Step 4: Commit**
```bash
git add frontend/package.json frontend/src/App.vue frontend/src/utils/toast.ts
git commit -m "feat: integrate vue-sonner for notifications"
```

---

### Task 2: Custom Confirmation System

**Files:**
- Create: `frontend/src/components/ConfirmModal.vue`
- Create: `frontend/src/composables/useConfirm.ts`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Create ConfirmModal.vue**
A Tailwind-styled modal with title, message, and Confirm/Cancel buttons. Support a `variant="danger"` for red buttons.

- [ ] **Step 2: Create useConfirm Composable**
Implement a global state that controls the modal and returns a Promise.
```typescript
const state = reactive({ isOpen: false, resolve: null as any, ... })
export function useConfirm() {
  return (options) => new Promise(res => { state.resolve = res; ... })
}
```

- [ ] **Step 3: Register Modal in App.vue**
Add `<ConfirmModal />` to the template.

- [ ] **Step 4: Commit**
```bash
git add frontend/src/components/ConfirmModal.vue frontend/src/composables/useConfirm.ts frontend/src/App.vue
git commit -m "feat: implement custom confirmation modal"
```

---

### Task 3: Global Dialog Replacement

**Files:**
- Modify: `frontend/src/views/InvitesView.vue`
- Modify: `frontend/src/views/PersonsView.vue`
- Modify: `frontend/src/views/GroupsView.vue`
- Modify: `frontend/src/views/SettingsView.vue`

- [ ] **Step 1: Replace alert() calls**
Use `notify.success/error` in all catch blocks and success handlers.

- [ ] **Step 2: Replace confirm() calls**
Import `useConfirm` and update all deletion/invalidation logic to `await confirm(...)`.

- [ ] **Step 3: Commit**
```bash
git commit -a -m "refactor: replace browser-native dialogs with custom UI components"
```

---

### Task 4: Skeleton Loading States

**Files:**
- Create: `frontend/src/components/TableSkeleton.vue`
- Modify: `frontend/src/views/InvitesView.vue`
- Modify: `frontend/src/views/PersonsView.vue`
- Modify: `frontend/src/views/GroupsView.vue`

- [ ] **Step 1: Create TableSkeleton.vue**
An animated component that shows 5 rows of shimmering bars.
```vue
<template>
  <div v-for="i in 5" class="animate-pulse flex space-x-4 py-4 border-b dark:border-white/5">
    <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/4"></div>
    <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2"></div>
    <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/6 ml-auto"></div>
  </div>
</template>
```

- [ ] **Step 2: Implement in Views**
Replace `v-if="loading && items.length === 0"` text with `<TableSkeleton />`.

- [ ] **Step 3: Add consistent button spinners**
Create a `src/components/LoadingSpinner.vue` (SVG) and use it in all "Save" buttons when `isSaving` is true.

- [ ] **Step 4: Commit**
```bash
git add frontend/src/components/TableSkeleton.vue frontend/src/views/
git commit -m "feat: add skeleton loading states and button spinners"
```
