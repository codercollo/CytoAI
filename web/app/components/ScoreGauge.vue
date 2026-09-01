<script setup lang="ts">
const props = withDefaults(defineProps<{ value: number; label?: string; size?: number }>(), {
  label: 'score',
  size: 220,
})

const cx = 100
const cy = 100
const radius = 80
const startDeg = 180
const sweepDeg = 180

function color(v: number): string {
  if (v >= 70) return '#10B981'
  if (v >= 40) return '#F59E0B'
  return '#F43F5E'
}

function arcPath(fromDeg: number, toDeg: number): string {
  const toRad = (d: number) => (d * Math.PI) / 180
  const x1 = cx + radius * Math.cos(toRad(fromDeg))
  const y1 = cy - radius * Math.sin(toRad(fromDeg))
  const x2 = cx + radius * Math.cos(toRad(toDeg))
  const y2 = cy - radius * Math.sin(toRad(toDeg))
  const large = Math.abs(toDeg - fromDeg) > 180 ? 1 : 0
  const sweep = toDeg < fromDeg ? 0 : 1
  return `M ${x1} ${y1} A ${radius} ${radius} 0 ${large} ${sweep} ${x2} ${y2}`
}

const clamped = computed(() => Math.min(Math.max(props.value, 0), 100))
const valueArc = computed(() => arcPath(startDeg, startDeg - (clamped.value / 100) * sweepDeg))
const stroke = computed(() => color(clamped.value))
</script>

<template>
  <div class="flex flex-col items-center">
    <svg :width="size" :height="size" viewBox="0 0 200 200">
      <path :d="arcPath(180, 0)" fill="none" stroke="rgba(255,255,255,0.08)" stroke-width="14" stroke-linecap="round" />
      <path :d="valueArc" fill="none" :stroke="stroke" stroke-width="14" stroke-linecap="round" />
      <text x="100" y="106" text-anchor="middle" class="fill-slate-100 font-sans" font-size="42" font-weight="800">
        {{ Math.round(value) }}
      </text>
      <text x="100" y="140" text-anchor="middle" class="fill-slate-400 font-sans uppercase tracking-wider" font-size="10" font-weight="600">{{ label }}</text>
    </svg>
  </div>
</template>
