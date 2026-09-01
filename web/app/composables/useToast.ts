import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: number
  type: ToastType
  message: string
}

const toasts = ref<Toast[]>([])
let nextId = 1

function dismiss(id: number) {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

export function useToast() {
  // `duration` <= 0 means the toast stays until manually dismissed.
  function push(type: ToastType, message: string, duration = 5000): number {
    const id = nextId++
    toasts.value.push({ id, type, message })
    if (import.meta.client && duration > 0) {
      window.setTimeout(() => dismiss(id), duration)
    }
    return id
  }

  function update(id: number, message: string) {
    const toast = toasts.value.find((t) => t.id === id)
    if (toast) toast.message = message
  }

  return {
    toasts,
    success: (message: string, duration = 5000) => push('success', message, duration),
    error: (message: string, duration = 5000) => push('error', message, duration),
    info: (message: string, duration = 5000) => push('info', message, duration),
    warning: (message: string, duration = 5000) => push('warning', message, duration),
    push,
    update,
    dismiss,
  }
}
