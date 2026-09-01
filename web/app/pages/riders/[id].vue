<script setup lang="ts">
import type { RiderResponse, ChartPoint, AnomalyFlag } from '~/composables/useCytoApi'

const route = useRoute()
const api = useCytoApi()
const req = useCytoRequest<RiderResponse>()
const rider = ref<RiderResponse | null>(null)

const score = computed(() => rider.value?.latest_score ?? null)
const factors = computed(() => rider.value?.factors)
const flags = computed<AnomalyFlag[]>(() => rider.value?.anomaly_flags ?? [])

async function load() {
  const res = await req.run(() => api.rider(route.params.id as string))
  if (res) rider.value = res
}

onMounted(load)

const points = computed<ChartPoint[]>(() => {
  if (!score.value) return []
  return [
    {
      t: score.value.scored_at || 'latest',
      bhi: score.value.battery_health_index ?? 0,
      rri: score.value.repayment_risk_index ?? 0,
    },
  ]
})

// Simple, documented MVP thresholds for the qualitative checklist lines.
const onTimePct = computed(() => Math.round((factors.value?.repayment.on_time_ratio ?? 0) * 100))
const cadence = computed(() => factors.value?.repayment.telemetry_cadence_proxy ?? 0)
const onTimeOk = computed(() => (factors.value?.repayment.on_time_ratio ?? 0) >= 0.8)
const daysLateOk = computed(() => (factors.value?.repayment.avg_days_late ?? 0) <= 2)
const cadenceOk = computed(() => cadence.value <= 0.5)
const tempOk = computed(() => {
  const t = factors.value?.battery.avg_temperature_c ?? 0
  return t >= 18 && t <= 32
})
</script>

<template>
  <div>
    <NuxtLink to="/" class="inline-flex items-center gap-1 text-sm text-slate-400 hover:text-indigo-400 transition-colors">
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
        <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18" />
      </svg>
      Back to Portfolio
    </NuxtLink>

    <div v-if="req.pending" class="mt-8 text-slate-400">Loading appraisal metrics…</div>
    <div v-else-if="req.error" class="mt-8">
      <p class="text-rose-400">{{ req.error.message }}</p>
      <button
        class="mt-3 inline-flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
        @click="load"
      >
        Retry
      </button>
    </div>

    <template v-else-if="rider">
      <div class="mt-4 mb-6 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold tracking-tight text-slate-100">
            Rider Appraisal Profile
          </h1>
          <p class="text-sm text-slate-400 mt-1">
            Reference ID: <span class="font-mono text-indigo-400 bg-indigo-950/40 px-2 py-0.5 rounded border border-indigo-900/30 text-xs">{{ route.params.id }}</span>
          </p>
        </div>
        <div class="flex items-center gap-2 text-xs text-slate-400 bg-slate-900/60 border border-slate-800 rounded-lg px-3 py-1.5 self-start">
          <span class="inline-block h-2 w-2 rounded-full bg-emerald-500"></span>
          <span>Model Version: {{ score?.model_version || 'N/A' }}</span>
        </div>
      </div>

      <template v-if="score">
      <!-- Score gauges row -->
      <div class="grid gap-6 md:grid-cols-3">
        <div class="glass-card flex flex-col items-center justify-center"><ScoreGauge :value="score.cyto_score ?? 0" label="Combined CytoScore" /></div>
        <div class="glass-card flex flex-col items-center justify-center"><ScoreGauge :value="score.battery_health_index ?? 0" label="Battery Health Index" /></div>
        <div class="glass-card flex flex-col items-center justify-center"><ScoreGauge :value="100 - (score.repayment_risk_index ?? 0)" label="Repayment Safety Index" /></div>
      </div>

      <!-- History and Details section -->
      <div class="grid gap-6 md:grid-cols-3 mt-6">
        <div class="glass-card md:col-span-2">
          <h2 class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider">Risk Trend History</h2>
          <DegradationChart :points="points" />
        </div>

        <div class="glass-card flex flex-col justify-between">
          <div>
            <h2 class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider">Evaluation Details</h2>
            <div class="divide-y divide-slate-800 text-xs">
              <div class="py-3 flex justify-between">
                <span class="text-slate-400">Battery Health (BHI)</span>
                <span class="font-mono font-semibold text-slate-200">{{ (score.battery_health_index ?? 0).toFixed(1) }}</span>
              </div>
              <div class="py-3 flex justify-between">
                <span class="text-slate-400">Repayment Risk (RRI)</span>
                <span class="font-mono font-semibold text-slate-200">{{ (score.repayment_risk_index ?? 0).toFixed(1) }}</span>
              </div>
              <div class="py-3 flex justify-between">
                <span class="text-slate-400">Combined CytoScore</span>
                <span class="font-mono font-semibold text-indigo-400">{{ (score.cyto_score ?? 0).toFixed(1) }}</span>
              </div>
              <div class="py-3 flex justify-between">
                <span class="text-slate-400">Scored At</span>
                <span class="font-mono text-slate-200">{{ score.scored_at || '—' }}</span>
              </div>
            </div>
          </div>

          <div class="mt-6 pt-4 border-t border-slate-800 text-[11px] text-slate-500 leading-normal">
            This risk evaluation is generated via green EV telematics paired with repayment data models.
          </div>
        </div>
      </div>

      <!-- Contributing factors checklist (mvp.md §5) -->
      <div class="glass-card mt-6 p-5">
        <h2 class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider">Contributing Factors</h2>
        <div v-if="factors" class="grid gap-6 md:grid-cols-3">
          <div>
            <h3 class="mb-3 text-xs font-semibold text-indigo-400 uppercase tracking-wider">Repayment</h3>
            <ul class="space-y-2 text-xs text-slate-300">
              <li class="flex items-center gap-2">
                <span v-if="onTimeOk" class="font-bold text-emerald-400">✓</span><span v-else class="font-bold text-amber-400">!</span>
                <span>{{ onTimePct }}% payments on time</span>
              </li>
              <li class="flex items-center gap-2">
                <span v-if="daysLateOk" class="font-bold text-emerald-400">✓</span><span v-else class="font-bold text-amber-400">!</span>
                <span>{{ (factors.repayment.avg_days_late ?? 0).toFixed(1) }} days late (avg)</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="font-bold text-slate-600">•</span>
                <span>Payment cadence {{ (factors.repayment.payment_cadence_proxy ?? 0).toFixed(2) }} std days</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="font-bold text-slate-600">•</span>
                <span>Loan-to-battery {{ (factors.repayment.loan_to_battery_value_ratio ?? 0).toFixed(2) }}</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="font-bold text-slate-600">•</span>
                <span>{{ (factors.repayment.tenure_days ?? 0).toFixed(0) }} days tenure</span>
              </li>
            </ul>
          </div>

          <div>
            <h3 class="mb-3 text-xs font-semibold text-indigo-400 uppercase tracking-wider">Activity</h3>
            <ul class="space-y-2 text-xs text-slate-300">
              <li class="flex items-center gap-2">
                <span v-if="cadenceOk" class="font-bold text-emerald-400">✓</span><span v-else class="font-bold text-amber-400">!</span>
                <span>Battery swap/usage cadence {{ cadence.toFixed(3) }} (regularity)</span>
              </li>
            </ul>
          </div>

          <div>
            <h3 class="mb-3 text-xs font-semibold text-indigo-400 uppercase tracking-wider">Battery</h3>
            <ul class="space-y-2 text-xs text-slate-300">
              <li class="flex items-center gap-2">
                <span v-if="tempOk" class="font-bold text-emerald-400">✓</span><span v-else class="font-bold text-amber-400">!</span>
                <span>{{ (factors.battery.avg_temperature_c ?? 0).toFixed(1) }}°C average temperature</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="font-bold text-slate-600">•</span>
                <span>{{ (factors.battery.cycle_count ?? 0).toFixed(0) }} charge cycles</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="font-bold text-slate-600">•</span>
                <span>{{ (factors.battery.avg_depth_of_discharge ?? 0).toFixed(1) }}% average depth of discharge</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="font-bold text-slate-600">•</span>
                <span>{{ (factors.battery.age_days ?? 0).toFixed(0) }} days age</span>
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Anomaly review -->
      <div class="glass-card mt-6 p-5">
        <h2 class="mb-3 text-sm font-semibold text-slate-200 uppercase tracking-wider">Anomaly Review</h2>
        <p v-if="flags.length === 0" class="text-xs text-slate-500">No anomalies detected for this rider.</p>
        <ul v-else class="space-y-2">
          <li v-for="(f, i) in flags" :key="i" class="flex items-start gap-2 text-xs">
            <AnomalyBadge :flags="[f]" />
            <span class="text-slate-300">{{ f.reason }}</span>
          </li>
        </ul>
        <p class="mt-3 text-[11px] text-slate-500">Anomalies are review signals, not rejections (spec.md §9).</p>
      </div>
      </template>

      <div v-else class="mt-8 text-slate-400">
        No score yet — upload telemetry and repayment data to score this rider.
      </div>
    </template>
  </div>
</template>
