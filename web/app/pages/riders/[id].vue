<script setup lang="ts">
import type { RiderResponse, ChartPoint, AnomalyFlag, ScoreResponse } from '~/composables/useCytoApi'

const route = useRoute()
const api = useCytoApi()
const toast = useToast()
const req = useCytoRequest<RiderResponse>()
const computeReq = useCytoRequest<ScoreResponse>()

const rider = ref<RiderResponse | null>(null)
const insufficientReason = ref<string | null>(null)

const score = computed(() => rider.value?.latest_score ?? null)
const factors = computed(() => rider.value?.factors)
const flags = computed<AnomalyFlag[]>(() => rider.value?.anomaly_flags ?? [])

// For swap_network loans the BHI describes the fleet/pool the rider draws
// from, not a single collateral battery — label it distinctly so a reader
// does not mistake fleet health for "this rider's battery".
const isSwapNetwork = computed(() => rider.value?.loan?.financing_model === 'swap_network')
const bhiLabel = computed(() => (isSwapNetwork.value ? 'Fleet/Pool Health (BHI)' : 'Battery Health (BHI)'))

async function load() {
  insufficientReason.value = null
  const res = await req.run(() => api.rider(route.params.id as string))
  if (res) rider.value = res
}

async function triggerComputeScore() {
  if (!rider.value) return
  insufficientReason.value = null
  toast.info('Computing score for rider…')
  const res = await computeReq.run(() =>
    api.computeScore(rider.value!.id, rider.value?.battery?.id)
  )
  if (res) {
    if (res.insufficient_data) {
      insufficientReason.value = res.reason || 'Insufficient history to compute a fair score.'
      toast.warning(`Not enough data to score: ${res.reason || 'Insufficient records'}`)
    } else {
      toast.success(`Score computed: CytoScore ${res.cyto_score.toFixed(1)}`)
      await load()
    }
  }
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

    <!-- Auth Error Re-entry Card -->
    <AuthErrorCard
      v-if="req.error?.kind === 'auth'"
      class="mt-6"
      title="Authentication Failed (401/403)"
      message="Your partner API key is missing or unauthorized to view this rider. Enter your key below."
      @retry="load"
    />

    <div v-else-if="req.pending" class="mt-8 text-slate-400">Loading appraisal metrics…</div>

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
          <div class="mt-1 flex flex-wrap items-center gap-2 text-sm text-slate-400">
            <span>Reference ID:</span>
            <span class="font-mono text-indigo-400 bg-indigo-950/40 px-2 py-0.5 rounded border border-indigo-900/30 text-xs">{{ route.params.id }}</span>
            <span v-if="rider.external_ref" class="text-xs text-slate-500">({{ rider.external_ref }})</span>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <!-- Financing model badge -->
          <span
            class="rounded px-2.5 py-1 text-xs font-semibold border"
            :class="isSwapNetwork ? 'bg-emerald-950/60 text-emerald-400 border-emerald-800/40' : 'bg-indigo-950/60 text-indigo-400 border-indigo-800/40'"
          >
            {{ isSwapNetwork ? 'Swap Network (Shared Fleet)' : 'Leased Fixed (Dedicated)' }}
          </span>

          <div class="flex items-center gap-2 text-xs text-slate-400 bg-slate-900/60 border border-slate-800 rounded-lg px-3 py-1.5">
            <span class="inline-block h-2 w-2 rounded-full bg-emerald-500"></span>
            <span>Model: {{ score?.model_version || 'v1.0' }}</span>
          </div>

          <button
            :disabled="computeReq.pending.value"
            class="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white transition-all hover:bg-indigo-500 active:bg-indigo-700 shadow-sm disabled:opacity-50"
            @click="triggerComputeScore"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-3.5 h-3.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
            </svg>
            {{ computeReq.pending.value ? 'Scoring…' : (score ? 'Re-compute Score' : 'Compute Score') }}
          </button>
        </div>
      </div>

      <!-- Insufficient Data Notification Banner -->
      <div
        v-if="insufficientReason"
        class="mb-6 rounded-xl border border-amber-800/40 bg-amber-950/20 p-4 text-xs text-amber-300"
      >
        <div class="flex items-center gap-2 font-semibold text-amber-200 mb-1">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 text-amber-400">
            <path fill-rule="evenodd" d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0Zm-7-4a1 1 0 1 1-2 0 1 1 0 0 1 2 0ZM9 9a.75.75 0 0 0 0 1.5h.253a.25.25 0 0 1 .244.304l-.459 2.066A1.75 1.75 0 0 0 10.747 15H11a.75.75 0 0 0 0-1.5h-.253a.25.25 0 0 1-.244-.304l.459-2.066A1.75 1.75 0 0 0 9.253 9H9Z" clip-rule="evenodd" />
          </svg>
          Thin-File Assessment: Insufficient History to Score Fairly
        </div>
        <p>{{ insufficientReason }}</p>
        <p class="mt-2 text-[11px] text-slate-400">
          Under bias-monitoring policy (spec.md §9), thin-file riders are protected from arbitrary low scores until sufficient operational history is ingested.
        </p>
      </div>

      <template v-if="score">
      <!-- Score gauges row -->
      <div class="grid gap-6 md:grid-cols-3">
        <div class="glass-card flex flex-col items-center justify-center"><ScoreGauge :value="score.cyto_score ?? 0" label="Combined CytoScore" /></div>
        <div class="glass-card flex flex-col items-center justify-center"><ScoreGauge :value="score.battery_health_index ?? 0" :label="bhiLabel" /></div>
        <div class="glass-card flex flex-col items-center justify-center"><ScoreGauge :value="100 - (score.repayment_risk_index ?? 0)" label="Repayment Safety Index" /></div>
      </div>

      <!-- History and Details section -->
      <div class="grid gap-6 md:grid-cols-3 mt-6">
        <div class="glass-card md:col-span-2">
          <h2 class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider">Risk Trend History</h2>
          <DegradationChart :points="points" :bhi-label="bhiLabel" />
        </div>

        <div class="glass-card flex flex-col justify-between">
          <div>
            <h2 class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider">Evaluation Details</h2>
            <div class="divide-y divide-slate-800 text-xs">
              <div class="py-3 flex justify-between">
                <span class="text-slate-400">Financing Model</span>
                <span class="font-semibold text-slate-200">{{ isSwapNetwork ? 'Swap Network' : 'Leased Fixed' }}</span>
              </div>
              <div class="py-3 flex justify-between">
                <span class="text-slate-400">BHI Scope</span>
                <span class="font-mono text-slate-200">{{ isSwapNetwork ? 'Shared Fleet Pool' : 'Dedicated Battery' }}</span>
              </div>
              <div class="py-3 flex justify-between">
                <span class="text-slate-400">{{ bhiLabel }}</span>
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
              <!-- Swap-network specific factor: battery stress profile vs fleet -->
              <li v-if="isSwapNetwork && factors.repayment.battery_stress_profile != null" class="flex items-center gap-2">
                <span class="font-bold text-emerald-400">•</span>
                <span>Fleet stress profile {{ (factors.repayment.battery_stress_profile ?? 0).toFixed(2) }} (relative to fleet)</span>
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
            <h3 class="mb-3 text-xs font-semibold text-indigo-400 uppercase tracking-wider">
              {{ isSwapNetwork ? 'Fleet Battery Pool' : 'Collateral Battery' }}
            </h3>
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

      <!-- Empty state when no score exists for rider yet -->
      <div v-else class="glass-card mt-8 p-8 text-center border border-slate-800">
        <div class="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-slate-900 border border-slate-800 text-slate-400 mb-3">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
          </svg>
        </div>
        <h3 class="text-sm font-semibold text-slate-200">No Score Appraisal Record Yet</h3>
        <p class="mt-1 text-xs text-slate-400 max-w-sm mx-auto">
          This rider has not been evaluated yet. You can trigger an on-demand score calculation or ingest telemetry / repayments.
        </p>
        <div class="mt-4 flex items-center justify-center gap-3">
          <button
            :disabled="computeReq.pending.value"
            class="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition-all hover:bg-indigo-500 shadow-sm disabled:opacity-50"
            @click="triggerComputeScore"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-3.5 h-3.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
            </svg>
            {{ computeReq.pending.value ? 'Computing Score…' : 'Compute Score Now' }}
          </button>
          <NuxtLink
            to="/upload"
            class="inline-flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-xs font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
          >
            Upload Data
          </NuxtLink>
        </div>
      </div>
    </template>
  </div>
</template>
