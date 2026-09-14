<script setup lang="ts">
const toast = useToast()

const activeTab = ref<'rider' | 'ingest' | 'score'>('rider')
const copiedSnippet = ref<string | null>(null)

function copyCode(text: string, label: string) {
  if (import.meta.client && navigator.clipboard) {
    navigator.clipboard.writeText(text)
    copiedSnippet.value = label
    toast.success(`Copied ${label} snippet to clipboard`)
    setTimeout(() => {
      if (copiedSnippet.value === label) {
        copiedSnippet.value = null
      }
    }, 2000)
  }
}

const riderCurl = `curl -X POST "https://api.cytoai.com/v1/riders" \\
  -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "external_ref": "rider_1001",
    "loan": {
      "external_ref": "loan_1001",
      "principal_kes": 300000,
      "term_months": 12,
      "daily_installment_kes": 250,
      "financing_model": "swap_network"
    }
  }'`

const riderResponse = `{
  "id": "7f8c9d0a-1b2c-3d4e-5f6a-7b8c9d0a1b2c",
  "external_ref": "rider_1001",
  "onboarded_at": "2026-09-14T21:00:00Z",
  "registration_status": "linked",
  "loan": {
    "id": "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
    "external_ref": "loan_1001",
    "principal_kes": 300000,
    "term_months": 12,
    "daily_installment_kes": 250,
    "financing_model": "swap_network"
  },
  "anomaly_flags": []
}`

const ingestCurl = `curl -X POST "https://api.cytoai.com/v1/swaps" \\
  -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \\
  -H "Content-Type: text/csv" \\
  --data-binary $'rider_id,battery_id,station_id,swapped_at,returned_state_of_charge,received_state_of_charge,returned_temperature_c,returned_cycle_count,returned_depth_of_discharge,distance_km_since_last_swap,paid_amount_kes\\nrider_1001,battery_pool_01,ST-01,2026-09-14T08:00:00Z,35,98,28,110,65,42.0,300.0'`

const ingestResponse = `{
  "inserted": 1
}`

const scoreCurl = `curl -X POST "https://api.cytoai.com/v1/score" \\
  -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "rider_id": "rider_1001"
  }'`

const scoreResponse = `{
  "rider_id": "7f8c9d0a-1b2c-3d4e-5f6a-7b8c9d0a1b2c",
  "financing_model": "swap_network",
  "bhi_context": "fleet",
  "battery_health_index": 82.5,
  "repayment_risk_index": 18.0,
  "cyto_score": 82.25,
  "battery_model_version": "bhi_swap_v1",
  "repayment_model_version": "rri_swap_v1",
  "scored_at": "2026-09-14T22:00:00Z",
  "estimated_swap_fee_burden_kes": 300.0,
  "swap_fee_burden_rri_adjustment": 5.0,
  "anomaly_flags": []
}`
</script>

<template>
  <div class="space-y-8">
    <!-- Header -->
    <div>
      <div class="flex items-center gap-3">
        <span class="inline-flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-600/20 border border-indigo-500/30 text-indigo-400">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75 22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3-4.5 16.5" />
          </svg>
        </span>
        <div>
          <h1 class="text-xl font-bold tracking-tight text-slate-100 sm:text-2xl">Developer &amp; Partner API Access</h1>
          <p class="text-xs sm:text-sm text-slate-400">Production integration reference for EV financing partners and swap operators.</p>
        </div>
      </div>
    </div>

    <!-- Auth Overview Box -->
    <div class="rounded-xl border border-slate-800 bg-[#0c1220] p-5 sm:p-6 space-y-4">
      <h2 class="text-sm font-semibold uppercase tracking-wider text-indigo-400">Authentication Specifications</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs text-slate-300">
        <div class="rounded-lg border border-slate-800 bg-slate-900/60 p-4 space-y-1.5">
          <div class="font-semibold text-slate-200">Base API URL</div>
          <div class="font-mono text-emerald-400 bg-slate-950/80 px-2.5 py-1.5 rounded border border-slate-800/80 select-all">
            https://api.cytoai.com/v1
          </div>
        </div>
        <div class="rounded-lg border border-slate-800 bg-slate-900/60 p-4 space-y-1.5">
          <div class="font-semibold text-slate-200">Header Format</div>
          <div class="font-mono text-indigo-300 bg-slate-950/80 px-2.5 py-1.5 rounded border border-slate-800/80 select-all">
            Authorization: Bearer &lt;YOUR_PARTNER_API_KEY&gt;
          </div>
        </div>
      </div>
      <div class="rounded-lg border border-indigo-500/20 bg-indigo-950/20 p-4 text-xs text-slate-300 leading-relaxed">
        <span class="font-semibold text-indigo-300">Provisioning Note:</span> Partner API keys are issued upon onboarding via the administrative CLI tool (<code class="font-mono text-indigo-200">cytoai seedpartner</code>). Provided keys are bcrypt-hashed at rest and enforce strict multi-tenant partner data isolation for all queries.
      </div>
    </div>

    <!-- Code Examples Section -->
    <div class="space-y-4">
      <div class="flex items-center justify-between border-b border-slate-800 pb-3">
        <h2 class="text-base font-semibold text-slate-200">Integration Workflows</h2>
        <!-- Tabs -->
        <div class="flex items-center gap-1.5 bg-slate-900/80 p-1 rounded-lg border border-slate-800 text-xs">
          <button
            class="px-3 py-1.5 rounded-md font-medium transition-colors"
            :class="activeTab === 'rider' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'"
            @click="activeTab = 'rider'"
          >
            1. Register Rider
          </button>
          <button
            class="px-3 py-1.5 rounded-md font-medium transition-colors"
            :class="activeTab === 'ingest' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'"
            @click="activeTab = 'ingest'"
          >
            2. Upload Swaps
          </button>
          <button
            class="px-3 py-1.5 rounded-md font-medium transition-colors"
            :class="activeTab === 'score' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'"
            @click="activeTab = 'score'"
          >
            3. Compute Score
          </button>
        </div>
      </div>

      <!-- Tab Content: 1. Register Rider -->
      <div v-if="activeTab === 'rider'" class="space-y-4">
        <div class="text-xs text-slate-300">
          Registers a rider under your partner account with their financing loan terms.
        </div>
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <!-- Request -->
          <div class="rounded-xl border border-slate-800 bg-[#0c1220] overflow-hidden">
            <div class="px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 flex items-center justify-between text-xs">
              <span class="font-mono text-emerald-400 font-semibold">POST /v1/riders</span>
              <button
                class="text-slate-400 hover:text-white transition-colors"
                @click="copyCode(riderCurl, 'Register Rider')"
              >
                {{ copiedSnippet === 'Register Rider' ? '✓ Copied' : 'Copy cURL' }}
              </button>
            </div>
            <pre class="p-4 text-xs font-mono text-slate-200 overflow-x-auto whitespace-pre-wrap leading-relaxed">{{ riderCurl }}</pre>
          </div>
          <!-- Response -->
          <div class="rounded-xl border border-slate-800 bg-[#0c1220] overflow-hidden">
            <div class="px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 text-xs font-mono text-slate-400">
              Response (201 Created)
            </div>
            <pre class="p-4 text-xs font-mono text-indigo-300 overflow-x-auto whitespace-pre-wrap leading-relaxed">{{ riderResponse }}</pre>
          </div>
        </div>
      </div>

      <!-- Tab Content: 2. Upload Swaps -->
      <div v-if="activeTab === 'ingest'" class="space-y-4">
        <div class="text-xs text-slate-300">
          Ingests real-time or daily battery swap event records with actual KES paid amounts.
        </div>
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <!-- Request -->
          <div class="rounded-xl border border-slate-800 bg-[#0c1220] overflow-hidden">
            <div class="px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 flex items-center justify-between text-xs">
              <span class="font-mono text-emerald-400 font-semibold">POST /v1/swaps</span>
              <button
                class="text-slate-400 hover:text-white transition-colors"
                @click="copyCode(ingestCurl, 'Upload Swaps')"
              >
                {{ copiedSnippet === 'Upload Swaps' ? '✓ Copied' : 'Copy cURL' }}
              </button>
            </div>
            <pre class="p-4 text-xs font-mono text-slate-200 overflow-x-auto whitespace-pre-wrap leading-relaxed">{{ ingestCurl }}</pre>
          </div>
          <!-- Response -->
          <div class="rounded-xl border border-slate-800 bg-[#0c1220] overflow-hidden">
            <div class="px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 text-xs font-mono text-slate-400">
              Response (200 OK)
            </div>
            <pre class="p-4 text-xs font-mono text-indigo-300 overflow-x-auto whitespace-pre-wrap leading-relaxed">{{ ingestResponse }}</pre>
          </div>
        </div>
      </div>

      <!-- Tab Content: 3. Compute Score -->
      <div v-if="activeTab === 'score'" class="space-y-4">
        <div class="text-xs text-slate-300">
          Computes the CytoScore, Battery Health Index (BHI), and Repayment Risk Index (RRI) with rule-based swap fee adjustments.
        </div>
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <!-- Request -->
          <div class="rounded-xl border border-slate-800 bg-[#0c1220] overflow-hidden">
            <div class="px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 flex items-center justify-between text-xs">
              <span class="font-mono text-emerald-400 font-semibold">POST /v1/score</span>
              <button
                class="text-slate-400 hover:text-white transition-colors"
                @click="copyCode(scoreCurl, 'Compute Score')"
              >
                {{ copiedSnippet === 'Compute Score' ? '✓ Copied' : 'Copy cURL' }}
              </button>
            </div>
            <pre class="p-4 text-xs font-mono text-slate-200 overflow-x-auto whitespace-pre-wrap leading-relaxed">{{ scoreCurl }}</pre>
          </div>
          <!-- Response -->
          <div class="rounded-xl border border-slate-800 bg-[#0c1220] overflow-hidden">
            <div class="px-4 py-2.5 bg-slate-900/90 border-b border-slate-800 text-xs font-mono text-slate-400">
              Response (200 OK)
            </div>
            <pre class="p-4 text-xs font-mono text-indigo-300 overflow-x-auto whitespace-pre-wrap leading-relaxed">{{ scoreResponse }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
