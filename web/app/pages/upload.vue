<script setup lang="ts">
import type { RowError, RescoreResponse } from '~/composables/useCytoApi'

const api = useCytoApi()
const toast = useToast()
const portfolioRefresh = useState<number>('portfolio_refresh', () => 0)

const telemetryReq = useCytoRequest<{ inserted: number }>()
const repaymentReq = useCytoRequest<{ inserted: number }>()
const rescoreReq = useCytoRequest<RescoreResponse>()

const telemetryStatus = ref('')
const telemetryErrors = ref<RowError[]>([])
const repaymentStatus = ref('')
const repaymentErrors = ref<RowError[]>([])

const telemetrySettled = ref(false)
const repaymentSettled = ref(false)
const scoring = ref(false)

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
  if (telemetrySettled.value && repaymentSettled.value) {
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
</script>

<template>
  <div>
    <div class="mb-6 flex items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-slate-100">Data Ingestion Portal</h1>
        <p class="mt-1 text-sm text-slate-400">
          Ingest telemetry and repayment events via CSV files to run score models. Batch validation will reject malformed rows.
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

    <div class="grid gap-6 md:grid-cols-2">
      <!-- Telemetry Upload -->
      <div class="glass-card">
        <div class="mb-4">
          <h2 class="text-base font-semibold text-slate-200">Battery Telematics</h2>
          <p class="text-xs text-slate-400 mt-1">Upload daily battery usage logs (voltages, temperatures, state of charge).</p>
        </div>
        <UploadDropzone label="Select Telemetry CSV" hint="Required headers: battery_id, reading_at, voltage, temperature_c" @file="uploadTelemetry" />
        <div class="mt-3 font-medium text-sm" :class="telemetryStatus === 'Upload failed' ? 'text-rose-400' : 'text-emerald-400'">
          {{ telemetryStatus }}
        </div>
        <div v-if="telemetryErrors.length" class="mt-4 rounded-lg bg-rose-950/20 border border-rose-900/30 p-3">
          <div class="text-xs font-semibold text-rose-400 mb-1">Validation Errors:</div>
          <ul class="space-y-1 font-mono text-[11px] text-rose-300">
            <li v-for="(er, i) in telemetryErrors" :key="i" class="list-disc list-inside">
              {{ er.row ? `Row ${er.row}:` : '' }} <span class="font-semibold">{{ er.field }}</span> {{ er.error }}
            </li>
          </ul>
        </div>
      </div>

      <!-- Repayments Upload -->
      <div class="glass-card">
        <div class="mb-4">
          <h2 class="text-base font-semibold text-slate-200">Repayment Records</h2>
          <p class="text-xs text-slate-400 mt-1">Upload rider credit history and transaction repayment schedules.</p>
        </div>
        <UploadDropzone label="Select Repayment CSV" hint="Required headers: loan_id, due_date, paid_date, status" @file="uploadRepayment" />
        <div class="mt-3 font-medium text-sm" :class="repaymentStatus === 'Upload failed' ? 'text-rose-400' : 'text-emerald-400'">
          {{ repaymentStatus }}
        </div>
        <div v-if="repaymentErrors.length" class="mt-4 rounded-lg bg-rose-950/20 border border-rose-900/30 p-3">
          <div class="text-xs font-semibold text-rose-400 mb-1">Validation Errors:</div>
          <ul class="space-y-1 font-mono text-[11px] text-rose-300">
            <li v-for="(er, i) in repaymentErrors" :key="i" class="list-disc list-inside">
              {{ er.row ? `Row ${er.row}:` : '' }} <span class="font-semibold">{{ er.field }}</span> {{ er.error }}
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>
