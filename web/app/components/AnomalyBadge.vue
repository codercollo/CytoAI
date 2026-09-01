<script setup lang="ts">
import type { AnomalyFlag } from '~/composables/useCytoApi'

const props = defineProps<{ flags?: AnomalyFlag[] }>()

const count = computed(() => (props.flags || []).length)
</script>

<template>
  <span
    v-if="count > 0"
    class="inline-flex items-center gap-1 rounded-full bg-amber-950/40 border border-amber-800/40 px-2 py-0.5 text-[10px] font-semibold text-amber-300"
    :title="`${count} anomaly flag${count === 1 ? '' : 's'} — review, not a rejection`"
  >
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="h-3 w-3">
      <path
        fill-rule="evenodd"
        d="M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003ZM12 8.25a.75.75 0 0 1 .75.75v3.75a.75.75 0 0 1-1.5 0V9a.75.75 0 0 1 .75-.75Zm0 8.25a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z"
        clip-rule="evenodd"
      />
    </svg>
    {{ count }}
  </span>
  <span v-else class="text-xs text-slate-700">—</span>
</template>
