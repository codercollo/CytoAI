<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    title?: string
    message?: string
  }>(),
  {
    title: 'Authentication Required',
    message: 'Check your partner API key or enter a valid one below to retry.',
  }
)

const emit = defineEmits<{ (e: 'retry'): void }>()

const { apiKey, setApiKey } = useApiKey()
const toast = useToast()

const inputKey = ref(apiKey.value)
const showKey = ref(false)

watch(apiKey, (v) => {
  inputKey.value = v
})

function saveAndRetry() {
  const trimmed = inputKey.value.trim()
  setApiKey(trimmed)
  if (!trimmed) {
    toast.warning('Partner API key cleared')
  } else {
    toast.success('API key updated — retrying request')
  }
  emit('retry')
}
</script>

<template>
  <div class="rounded-xl border border-amber-500/30 bg-amber-950/20 p-5 text-left">
    <div class="flex items-start gap-3">
      <div class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-400">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z" />
        </svg>
      </div>
      <div class="flex-1 min-w-0">
        <h3 class="text-sm font-semibold text-amber-300">{{ title }}</h3>
        <p class="mt-1 text-xs text-slate-300">{{ message }}</p>

        <form class="mt-3 flex flex-col sm:flex-row items-stretch sm:items-center gap-2" @submit.prevent="saveAndRetry">
          <div class="relative flex-1">
            <input
              v-model="inputKey"
              :type="showKey ? 'text' : 'password'"
              placeholder="Paste partner API key here (Bearer ...)"
              autocomplete="off"
              class="w-full rounded-lg border border-slate-700 bg-slate-900/80 px-3 py-2 pr-10 text-xs font-mono text-slate-200 placeholder:text-slate-500 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500/40"
            />
            <button
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200 text-xs p-1"
              :title="showKey ? 'Hide key' : 'Show key'"
              @click="showKey = !showKey"
            >
              <svg v-if="!showKey" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
              </svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.451 10.451 0 0 1 12 4.5c4.756 0 8.773 3.162 10.065 7.498a10.522 10.522 0 0 1-4.293 5.774M6.228 6.228 3 3m3.228 3.228 3.65 3.65m7.894 7.894L21 21m-3.228-3.228-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.242 4.242L9.88 9.88" />
              </svg>
            </button>
          </div>
          <button
            type="submit"
            class="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition-colors hover:bg-indigo-500 active:bg-indigo-700"
          >
            Save &amp; Retry
          </button>
        </form>
      </div>
    </div>
  </div>
</template>
