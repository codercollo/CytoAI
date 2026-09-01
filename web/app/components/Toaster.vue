<script setup lang="ts">
const { toasts, dismiss } = useToast()

const iconPath: Record<string, string> = {
  success: 'M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z',
  error: 'm9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z',
  warning:
    'M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z',
  info: 'm11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z',
}

const ring: Record<string, string> = {
  success: 'border-emerald-800/40',
  error: 'border-rose-800/40',
  warning: 'border-amber-800/40',
  info: 'border-indigo-800/40',
}

const badge: Record<string, string> = {
  success: 'bg-emerald-500/10 text-emerald-400',
  error: 'bg-rose-500/10 text-rose-400',
  warning: 'bg-amber-500/10 text-amber-400',
  info: 'bg-indigo-500/10 text-indigo-400',
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-[100] flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2">
      <TransitionGroup
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="opacity-0 translate-y-2"
        leave-active-class="transition duration-150 ease-in"
        leave-to-class="opacity-0 translate-y-2"
      >
        <div
          v-for="t in toasts"
          :key="t.id"
          class="glass-card flex items-start gap-3 border p-3"
          :class="ring[t.type]"
        >
          <span
            class="mt-0.5 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full"
            :class="badge[t.type]"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke-width="1.5"
              stroke="currentColor"
              class="h-4 w-4"
            >
              <path stroke-linecap="round" stroke-linejoin="round" :d="iconPath[t.type]" />
            </svg>
          </span>
          <p class="flex-1 text-sm leading-snug text-slate-200">{{ t.message }}</p>
          <button
            type="button"
            aria-label="Dismiss"
            class="shrink-0 rounded-md p-0.5 text-slate-500 transition-colors hover:text-slate-300"
            @click="dismiss(t.id)"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke-width="1.5"
              stroke="currentColor"
              class="h-4 w-4"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
