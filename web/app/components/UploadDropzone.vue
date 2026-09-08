<script setup lang="ts">
const props = defineProps<{ label: string; hint?: string }>()

const emit = defineEmits<{ (e: 'file', file: File): void }>()

const dragging = ref(false)
const inputRef = ref<HTMLInputElement>()

function onDrop(e: DragEvent) {
  dragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) emit('file', file)
}

function onPick(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) emit('file', file)
  input.value = ''
}
</script>

<template>
  <div
    class="cursor-pointer p-5 sm:p-8 text-center transition-all border-dashed border-2 rounded-xl hover:border-indigo-500/50 hover:bg-indigo-950/5"
    :class="dragging ? 'border-indigo-500 bg-indigo-950/20' : 'border-slate-800 bg-[#0e1626]'"
    @dragover.prevent="dragging = true"
    @dragleave.prevent="dragging = false"
    @drop.prevent="onDrop"
    @click="inputRef?.click()"
  >
    <input ref="inputRef" type="file" accept=".csv,text/csv" class="hidden" @change="onPick" />
    <div class="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-slate-900 border border-slate-800 mb-3 text-slate-400">
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 16.5V9.75m0 0 3 3m-3-3-3 3M6.75 19.5a4.5 4.5 0 0 1-1.41-8.775 5.25 5.25 0 0 1 10.233-2.33 3 3 0 0 1 3.758 3.848A3.752 3.752 0 0 1 18 19.5H6.75Z" />
      </svg>
    </div>
    <div class="text-sm font-semibold text-slate-200">{{ label }}</div>
    <div class="mt-1 text-xs text-slate-400 font-sans">{{ hint || 'Drop a CSV here or click to browse' }}</div>
  </div>
</template>
