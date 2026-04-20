# Design Spec: UX Improvements (Toasts, Skeletons, and Confirmations)

## Overview
Elevate the application's user experience by replacing browser-native dialogs (`alert`, `confirm`) with professional UI components. Implement animated skeleton screens to provide better visual feedback during data fetching and add contextual loading states for interactive elements.

## Goals
- Implement a modern toast notification system using `vue-sonner`.
- Replace all `window.confirm()` calls with a custom, themed confirmation modal.
- Add animated skeleton loading states for all primary data tables.
- Ensure all new components are fully accessible and support dark mode.

## Architecture

### 1. Notification System (`vue-sonner`)
- **Integration**: Add `<Toaster />` to the root `App.vue`.
- **Styling**: Configure `vue-sonner` to use Tailwind-compatible styles.
- **Behavior**:
  - Success: Auto-dismiss after 3 seconds.
  - Error: Persistent until dismissed by the user.
  - Info/Warning: Auto-dismiss after 5 seconds.

### 2. Custom Confirmation Modal
- **Component**: `src/components/ConfirmModal.vue`.
- **API**: A global singleton or a simple utility function using a Promise-based approach.
- **Variants**:
  - `Default`: Indigo primary button.
  - `Danger`: Red primary button (for deletions/invalidations).

### 3. Skeleton Loading States
- **Component**: `src/components/SkeletonLoader.vue` or direct utility classes.
- **Implementation**:
  - **Tables**: Replace "Loading..." text with 5-10 shimmering rows matching the table's layout.
  - **Cards/Modals**: Shimmering blocks for content areas.

### 4. Interactive Loading
- **Buttons**: Enhance `isSaving` states to include a consistent SVG spinner alongside the text.
- **Form Fields**: Disable inputs while a parent action is "Saving".

## Implementation Strategy

### Phase 1: Infrastructure
- Install `vue-sonner`.
- Create `ConfirmModal.vue` and the `useConfirm` composable.
- Add `<Toaster />` to `App.vue`.

### Phase 2: Logic Replacement
- Global search for `alert()` and replace with `toast.error()` or `toast.success()`.
- Global search for `confirm()` and replace with `await confirm()`.

### Phase 3: Visual Polish
- Implement skeleton loaders in `PersonsView.vue`, `GroupsView.vue`, and `InvitesView.vue`.
- Standardize button spinners.

## Success Criteria
- [ ] No native `alert()` or `confirm()` calls remain in the codebase.
- [ ] Error notifications remain visible until dismissed.
- [ ] Users see clear, shimmering visual feedback while data is loading.
- [ ] Deleting an item shows a styled, red "Danger" confirmation modal.
