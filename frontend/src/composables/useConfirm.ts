import { reactive, readonly } from 'vue'

export interface ConfirmOptions {
  title: string
  message: string
  variant?: 'default' | 'danger'
  confirmLabel?: string
  cancelLabel?: string
}

interface ConfirmState {
  isOpen: boolean
  title: string
  message: string
  variant: 'default' | 'danger'
  confirmLabel: string
  cancelLabel: string
  resolve: ((value: boolean) => void) | null
}

const state = reactive<ConfirmState>({
  isOpen: false,
  title: '',
  message: '',
  variant: 'default',
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  resolve: null,
})

export function useConfirm() {
  const confirm = (options: ConfirmOptions): Promise<boolean> => {
    state.title = options.title
    state.message = options.message
    state.variant = options.variant || 'default'
    state.confirmLabel = options.confirmLabel || 'Confirm'
    state.cancelLabel = options.cancelLabel || 'Cancel'
    state.isOpen = true

    return new Promise((resolve) => {
      state.resolve = resolve
    })
  }

  const onConfirm = () => {
    state.isOpen = false
    if (state.resolve) state.resolve(true)
  }

  const onCancel = () => {
    state.isOpen = false
    if (state.resolve) state.resolve(false)
  }

  return {
    state: readonly(state),
    confirm,
    onConfirm,
    onCancel,
  }
}
