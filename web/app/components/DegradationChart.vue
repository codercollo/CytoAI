<script setup lang="ts">
import type { ChartPoint } from '~/composables/useCytoApi'

const props = withDefaults(
  defineProps<{
    points: ChartPoint[]
    bhiLabel?: string
  }>(),
  {
    bhiLabel: 'Battery Health Index (BHI)',
  }
)

const W = 600
const H = 240
const pad = { l: 44, r: 16, t: 16, b: 28 }

function x(i: number): number {
  if (props.points.length <= 1) return pad.l + (W - pad.l - pad.r) / 2
  return pad.l + (i / (props.points.length - 1)) * (W - pad.l - pad.r)
}

function y(v: number): number {
  const c = Math.min(Math.max(v, 0), 100)
  return pad.t + (1 - c / 100) * (H - pad.t - pad.b)
}

function line(key: 'bhi' | 'rri'): string {
  return props.points.map((p, i) => `${x(i)},${y(p[key])}`).join(' ')
}
</script>

<template>
  <div class="w-full">
    <svg :viewBox="`0 0 ${W} ${H}`" class="w-full">
      <line
        v-for="g in 5"
        :key="g"
        :x1="pad.l"
        :x2="W - pad.r"
        :y1="y((g - 1) * 25)"
        :y2="y((g - 1) * 25)"
        stroke="rgba(255,255,255,0.06)"
      />
      <text
        v-for="g in 5"
        :key="'l' + g"
        :x="pad.l - 8"
        :y="y((g - 1) * 25) + 4"
        text-anchor="end"
        font-size="10"
        class="fill-slate-500 font-mono"
      >
        {{ (g - 1) * 25 }}
      </text>

      <polyline :points="line('bhi')" fill="none" stroke="#6366f1" stroke-width="2.5" />
      <polyline :points="line('rri')" fill="none" stroke="#10b981" stroke-width="2.5" />

      <circle v-for="(p, i) in points" :key="'b' + i" :cx="x(i)" :cy="y(p.bhi)" r="4" fill="#6366f1" />
      <circle v-for="(p, i) in points" :key="'r' + i" :cx="x(i)" :cy="y(p.rri)" r="4" fill="#10b981" />
    </svg>

    <div class="mt-3 flex items-center gap-5 font-sans text-xs font-medium text-slate-400">
      <span class="flex items-center gap-2"><span class="inline-block h-2 w-2 rounded-full bg-indigo-500"></span> {{ bhiLabel }}</span>
      <span class="flex items-center gap-2"><span class="inline-block h-2 w-2 rounded-full bg-emerald-500"></span> Repayment Risk Index (RRI)</span>
    </div>

    <div v-if="points.length < 2" class="mt-3 font-mono text-xs text-slate-500">
      Historic series not yet available — showing the latest point.
    </div>
  </div>
</template>
