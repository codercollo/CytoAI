<script setup lang="ts">
import type { RowError, RescoreResponse } from '~/composables/useCytoApi'

const api = useCytoApi()
const toast = useToast()
const portfolioRefresh = useState<number>('portfolio_refresh', () => 0)

const telemetryReq = useCytoRequest<{ inserted: number }>()
const repaymentReq = useCytoRequest<{ inserted: number }>()
const swapsReq = useCytoRequest<{ inserted: number }>()
const rescoreReq = useCytoRequest<RescoreResponse>()

const telemetryStatus = ref('')
const telemetryErrors = ref<RowError[]>([])
const repaymentStatus = ref('')
const repaymentErrors = ref<RowError[]>([])
const swapsStatus = ref('')
const swapsErrors = ref<RowError[]>([])

const telemetrySettled = ref(false)
const repaymentSettled = ref(false)
const swapsSettled = ref(false)
const scoring = ref(false)

const authError = computed(() => {
  return (
    telemetryReq.error.value?.kind === 'auth' ||
    repaymentReq.error.value?.kind === 'auth' ||
    swapsReq.error.value?.kind === 'auth' ||
    rescoreReq.error.value?.kind === 'auth'
  )
})

async function runRescoreAll() {
  if (scoring.value) return
  scoring.value = true
  toast.info('Scoring riders…')
  const res = await rescoreReq.run(() => api.rescoreAll())
  scoring.value = false
  if (res) {
    const skipped = res.skipped?.length ?? 0
    if (skipped > 0) {
      toast.success(`Scored ${res.scored} riders, ${skipped} skipped (insufficient data)`)
    } else {
      toast.success(`Scored ${res.scored} riders`)
    }
    portfolioRefresh.value++
  }
}

async function maybeAutoRescore() {
  // Leased-fixed requires telematics + repayments.
  // Swap-network requires swaps + repayments.
  if ((telemetrySettled.value || swapsSettled.value) && repaymentSettled.value) {
    await runRescoreAll()
  }
}

async function uploadTelemetry(file: File) {
  telemetryStatus.value = 'Uploading…'
  telemetryErrors.value = []
  const res = await telemetryReq.run(() => api.uploadTelematics(file))
  if (res) {
    telemetryStatus.value = `Inserted ${res.inserted} rows`
    toast.success(`Telemetry uploaded — ${res.inserted} rows`)
  } else {
    telemetryStatus.value = 'Upload failed'
    const err = telemetryReq.error.value
    telemetryErrors.value = (err?.details?.row_errors as RowError[]) || [{ error: err?.message || 'Unknown error' }]
  }
  telemetrySettled.value = true
  await maybeAutoRescore()
}

async function uploadRepayment(file: File) {
  repaymentStatus.value = 'Uploading…'
  repaymentErrors.value = []
  const res = await repaymentReq.run(() => api.uploadRepayments(file))
  if (res) {
    repaymentStatus.value = `Inserted ${res.inserted} rows`
    toast.success(`Repayments uploaded — ${res.inserted} rows`)
  } else {
    repaymentStatus.value = 'Upload failed'
    const err = repaymentReq.error.value
    repaymentErrors.value = (err?.details?.row_errors as RowError[]) || [{ error: err?.message || 'Unknown error' }]
  }
  repaymentSettled.value = true
  await maybeAutoRescore()
}

async function uploadSwaps(file: File) {
  swapsStatus.value = 'Uploading…'
  swapsErrors.value = []
  const res = await swapsReq.run(() => api.uploadSwaps(file))
  if (res) {
    swapsStatus.value = `Inserted ${res.inserted} rows`
    toast.success(`Swap events uploaded — ${res.inserted} rows`)
  } else {
    swapsStatus.value = 'Upload failed'
    const err = swapsReq.error.value
    swapsErrors.value = (err?.details?.row_errors as RowError[]) || [{ error: err?.message || 'Unknown error' }]
  }
  swapsSettled.value = true
  await maybeAutoRescore()
}
</script>

<template>
  <div>
    <div class="mb-6 flex items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-slate-100">Data Ingestion Portal</h1>
        <p class="mt-1 text-sm text-slate-400">
          Ingest telematics, battery swaps, and repayment events via CSV files to run score models. Batch validation will reject malformed rows with per-row diagnostics.
        </p>
      </div>
      <button
        :disabled="scoring"
        class="inline-flex shrink-0 items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition-all hover:bg-indigo-500 active:bg-indigo-700 shadow-sm disabled:cursor-not-allowed disabled:opacity-50"
        @click="runRescoreAll"
      >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
        </svg>
        {{ scoring ? 'Scoring…' : 'Compute Scores' }}
      </button>
    </div>

    <!-- Auth Error Re-entry Card -->
    <AuthErrorCard
      v-if="authError"
      class="mb-6"
      title="Authentication Failed (401/403)"
      message="Your partner API key is missing or invalid. Enter your key below and click Save & Retry to proceed with ingestion."
      @retry="runRescoreAll"
    />

    <div class="grid gap-6 lg:grid-cols-3 md:grid-cols-2">
      <!-- Telemetry Upload (Leased Fixed / Dedicated Battery) -->
      <div class="glass-card flex flex-col justify-between">
        <div>
          <div class="mb-4">
            <div class="flex items-center justify-between">
              <h2 class="text-base font-semibold text-slate-200">Battery Telematics</h2>
              <span class="rounded bg-indigo-950/60 px-2 py-0.5 text-[10px] font-medium text-indigo-400 border border-indigo-800/40">Leased Fixed</span>
            </div>
            <p class="text-xs text-slate-400 mt-1">Upload daily battery usage logs (voltages, temperatures, state of charge) for dedicated collateral batteries.</p>
          </div>
          <UploadDropzone label="Select Telemetry CSV" hint="Headers: battery_id, reading_at, voltage, temperature_c" @file="uploadTelemetry" />
          <div v-if="telemetryStatus" class="mt-3 font-medium text-sm" :class="telemetryStatus === 'Upload failed' ? 'text-rose-400' : 'text-emerald-400'">
            {{ telemetryStatus }}
          </div>
        </div>
        <div v-if="telemetryErrors.length" class="mt-4 rounded-lg bg-rose-950/20 border border-rose-900/30 p-3">
          <div class="text-xs font-semibold text-rose-400 mb-1">Validation Errors ({{ telemetryErrors.length }}):</div>
          <ul class="space-y-1 font-mono text-[11px] text-rose-300 max-h-48 overflow-y-auto">
            <li v-for="(er, i) in telemetryErrors" :key="i" class="list-disc list-inside">
              {{ er.row ? `Row ${er.row}:` : '' }}
              <span v-if="er.field" class="font-semibold">{{ er.field }}</span>
              <span v-if="er.value" class="text-slate-400"> ({{ er.value }})</span>
              <span v-if="er.field">:</span>
              {{ er.error }}
            </li>
          </ul>
        </div>
      </div>

      <!-- Battery Swaps Upload (Swap-Network Operators) -->
      <div class="glass-card flex flex-col justify-between">
        <div>
          <div class="mb-4">
            <div class="flex items-center justify-between">
              <h2 class="text-base font-semibold text-slate-200">Battery Swaps</h2>
              <span class="rounded bg-emerald-950/60 px-2 py-0.5 text-[10px] font-medium text-emerald-400 border border-emerald-800/40">Swap Network</span>
            </div>
            <p class="text-xs text-slate-400 mt-1">Upload swap logs (station swaps, returned SoC, temperatures) for shared fleet-pool operators.</p>
          </div>
          <UploadDropzone label="Select Swaps CSV" hint="Headers: rider_id, battery_id, swapped_at" @file="uploadSwaps" />
          <div v-if="swapsStatus" class="mt-3 font-medium text-sm" :class="swapsStatus === 'Upload failed' ? 'text-rose-400' : 'text-emerald-400'">
            {{ swapsStatus }}
          </div>
        </div>
        <div v-if="swapsErrors.length" class="mt-4 rounded-lg bg-rose-950/20 border border-rose-900/30 p-3">
          <div class="text-xs font-semibold text-rose-400 mb-1">Validation Errors ({{ swapsErrors.length }}):</div>
          <ul class="space-y-1 font-mono text-[11px] text-rose-300 max-h-48 overflow-y-auto">
            <li v-for="(er, i) in swapsErrors" :key="i" class="list-disc list-inside">
              {{ er.row ? `Row ${er.row}:` : '' }}
              <span v-if="er.field" class="font-semibold">{{ er.field }}</span>
              <span v-if="er.value" class="text-slate-400"> ({{ er.value }})</span>
              <span v-if="er.field">:</span>
              {{ er.error }}
            </li>
          </ul>
        </div>
      </div>

      <!-- Repayments Upload (All Financing Models) -->
      <div class="glass-card flex flex-col justify-between">
        <div>
          <div class="mb-4">
            <div class="flex items-center justify-between">
              <h2 class="text-base font-semibold text-slate-200">Repayment Records</h2>
              <span class="rounded bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300 border border-slate-700">All Models</span>
            </div>
            <p class="text-xs text-slate-400 mt-1">Upload rider credit history, transaction records, and installment schedules for loan risk appraisal.</p>
          </div>
          <UploadDropzone label="Select Repayment CSV" hint="Headers: loan_id, due_date, paid_date, status" @file="uploadRepayment" />
          <div v-if="repaymentStatus" class="mt-3 font-medium text-sm" :class="repaymentStatus === 'Upload failed' ? 'text-rose-400' : 'text-emerald-400'">
            {{ repaymentStatus }}
          </div>
        </div>
        <div v-if="repaymentErrors.length" class="mt-4 rounded-lg bg-rose-950/20 border border-rose-900/30 p-3">
          <div class="text-xs font-semibold text-rose-400 mb-1">Validation Errors ({{ repaymentErrors.length }}):</div>
          <ul class="space-y-1 font-mono text-[11px] text-rose-300 max-h-48 overflow-y-auto">
            <li v-for="(er, i) in repaymentErrors" :key="i" class="list-disc list-inside">
              {{ er.row ? `Row ${er.row}:` : '' }}
              <span v-if="er.field" class="font-semibold">{{ er.field }}</span>
              <span v-if="er.value" class="text-slate-400"> ({{ er.value }})</span>
              <span v-if="er.field">:</span>
              {{ er.error }}
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>
