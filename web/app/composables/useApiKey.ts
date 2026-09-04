import { ref, computed } from 'vue'

const apiKey = ref('')
const initialized = ref(false)

export function useApiKey() {
  function init() {
    if (import.meta.client && !initialized.value) {
      apiKey.value = localStorage.getItem('cyto_api_key') || ''
      initialized.value = true
    }
  }

  function setApiKey(key: string) {
    apiKey.value = key
    if (import.meta.client) {
      if (key) {
        localStorage.setItem('cyto_api_key', key)
      } else {
        localStorage.removeItem('cyto_api_key')
      }
    }
  }

  const hasApiKey = computed(() => apiKey.value.trim().length > 0)

  // Auto-init on call
  init()

  return {
    apiKey,
    setApiKey,
    hasApiKey,
    init,
  }
}
