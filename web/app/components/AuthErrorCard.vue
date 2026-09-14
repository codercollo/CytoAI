<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    title?: string
    message?: string
  }>(),
  {
    title: 'Authentication Error',
    message: 'Unable to authenticate with the Cyto API server. Please retry.',
  }
)

const emit = defineEmits<{ (e: 'retry'): void }>()
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

        <div class="mt-3">
          <button
            type="button"
            class="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition-colors hover:bg-indigo-500 active:bg-indigo-700"
            @click="emit('retry')"
          >
            Retry Request
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
