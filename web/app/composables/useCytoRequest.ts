import { classifyApiError, type ApiError } from './useCytoApi'

const RETRY_DELAY_MS = 7000
const MAX_NETWORK_ATTEMPTS = 3
const MAX_SERVER_ATTEMPTS = 3

// useCytoRequest is the single shared request/error layer for the dashboard.
// Every API call flows through `run`, which classifies failures, shows the
// correct toast, and auto-retries network/server errors with a cap. Callers
// keep their own data state and only mutate it on a successful result, so a
// failed refresh never clobbers the last good data or fabricates an empty
// state.
export function useCytoRequest<T>() {
  const toast = useToast()

  const pending = ref(false)
  const error = ref<ApiError | null>(null)
  const data = ref<T | null>(null)

  let retryToastId: number | null = null
  let retryTimer: ReturnType<typeof setTimeout> | undefined

  function clearRetryToast() {
    if (retryToastId != null) {
      toast.dismiss(retryToastId)
      retryToastId = null
    }
  }

  function setRetryToast(type: 'warning' | 'error', text: string) {
    if (retryToastId == null) {
      retryToastId = toast.push(type, text, 0)
    } else {
      toast.update(retryToastId, text)
    }
  }

  function sleep(ms: number): Promise<void> {
    return new Promise((resolve) => {
      retryTimer = setTimeout(resolve, ms)
    })
  }

  async function run(fn: () => Promise<T>): Promise<T | null> {
    pending.value = true
    error.value = null
    let attempts = 0

    while (true) {
      try {
        const result = await fn()
        data.value = result
        error.value = null
        pending.value = false
        clearRetryToast()
        return result
      } catch (e) {
        const info = classifyApiError(e)

        if (info.kind === 'network') {
          attempts++
          if (attempts < MAX_NETWORK_ATTEMPTS) {
            error.value = info
            setRetryToast('warning', `Can't reach the server. Retrying (${attempts}/${MAX_NETWORK_ATTEMPTS})…`)
            await sleep(RETRY_DELAY_MS)
            continue
          }
          pending.value = false
          error.value = info
          clearRetryToast()
          toast.error("Can't reach the server.")
          return null
        }

        if (info.kind === 'auth') {
          pending.value = false
          error.value = info
          toast.error('Check your partner API key.')
          return null
        }

        if (info.kind === 'validation') {
          pending.value = false
          error.value = info
          return null
        }

        if (info.kind === 'server') {
          attempts++
          if (attempts < MAX_SERVER_ATTEMPTS) {
            error.value = info
            setRetryToast('error', `Something went wrong on our end. Retrying (${attempts}/${MAX_SERVER_ATTEMPTS})…`)
            await sleep(RETRY_DELAY_MS)
            continue
          }
          pending.value = false
          error.value = info
          clearRetryToast()
          toast.error('Something went wrong on our end.')
          return null
        }

        // unknown
        pending.value = false
        error.value = info
        return null
      }
    }
  }

  onBeforeUnmount(() => {
    if (retryTimer) clearTimeout(retryTimer)
    clearRetryToast()
  })

  return { pending, error, data, run }
}
