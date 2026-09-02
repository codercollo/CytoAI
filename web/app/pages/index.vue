<script setup lang="ts">
import type { Score, PortfolioResponse, AnomalyFlag, StressFlagsResponse } from "~/composables/useCytoApi";

const api = useCytoApi();
const req = useCytoRequest<PortfolioResponse>();
const scores = ref<Score[]>([]);
const hasData = ref(false);

// Operator-facing fleet stress (swap-network). Kept separate from the
// lender-facing portfolio because it means something different: battery-pool
// health for the operator, not repayment risk for a lender.
const stressReq = useCytoRequest<StressFlagsResponse>();
const stressFlags = ref<AnomalyFlag[]>([]);

// Bumped by the upload page after a rescore-all run; refresh if we're mounted.
const portfolioRefresh = useState<number>("portfolio_refresh", () => 0);

async function load() {
  const res = await req.run(() => api.portfolio());
  if (res) {
    scores.value = res.scores || [];
    hasData.value = true;
  }
}

async function loadStress() {
  const res = await stressReq.run(() => api.operatorStressFlags());
  if (res) stressFlags.value = res.flags || [];
}

onMounted(() => {
  load();
  loadStress();
});
watch(portfolioRefresh, () => {
  load();
  loadStress();
});

const errorMessage = computed(() => req.error.value?.message || "");

function short(id?: string): string {
  return id ? id.slice(0, 8) : "—";
}

function riskClass(v: number): string {
  if (v >= 70)
    return "text-emerald-400 bg-emerald-500/10 border-emerald-500/20";
  if (v >= 40) return "text-amber-400 bg-amber-500/10 border-amber-500/20";
  return "text-rose-400 bg-rose-500/10 border-rose-500/20";
}
</script>

<template>
  <div>
    <div class="mb-6 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-slate-100">
          Portfolio Credit & Battery Risks
        </h1>
        <p class="mt-1 text-sm text-slate-400">
          Overview of Kenyan EV finance portfolios and active scored riders.
        </p>
      </div>
      <button
        class="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition-all hover:bg-indigo-500 active:bg-indigo-700 shadow-sm"
        @click="load"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          stroke-width="2"
          stroke="currentColor"
          class="w-4 h-4"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"
          />
        </svg>
        Refresh Data
      </button>
    </div>

    <!-- Error banner (only when a refresh failed but we still have stale data) -->
    <div
      v-if="errorMessage && hasData"
      class="mb-6 flex items-center justify-between gap-4 rounded-lg bg-rose-950/20 border border-rose-900/30 p-3 text-xs text-rose-300"
    >
      <span>
        {{ errorMessage }}
        <span v-if="hasData" class="text-slate-400">— showing last loaded data.</span>
      </span>
      <button
        class="shrink-0 rounded-lg border border-rose-800/50 px-3 py-1.5 text-xs font-semibold text-rose-200 hover:bg-rose-950/30"
        @click="load"
      >
        Retry
      </button>
    </div>

    <!-- Quick stats overview row -->
    <div v-if="hasData" class="grid grid-cols-1 md:grid-cols-3 gap-5 mb-6">
      <div class="glass-card flex flex-col justify-between py-4">
        <span
          class="text-xs font-semibold text-slate-400 uppercase tracking-wider"
          >Total Scored Riders</span
        >
        <span class="text-3xl font-bold text-slate-100 mt-2">{{
          scores.length
        }}</span>
      </div>
      <div class="glass-card flex flex-col justify-between py-4">
        <span
          class="text-xs font-semibold text-slate-400 uppercase tracking-wider"
          >Average CytoScore</span
        >
        <span class="text-3xl font-bold text-indigo-400 mt-2">
          {{
            scores.length
              ? (
                  scores.reduce((acc, s) => acc + (s.cyto_score ?? 0), 0) /
                  scores.length
                ).toFixed(1)
              : "—"
          }}
        </span>
      </div>
      <div class="glass-card flex flex-col justify-between py-4">
        <span
          class="text-xs font-semibold text-slate-400 uppercase tracking-wider"
          >High Risk Count (&lt;40)</span
        >
        <span class="text-3xl font-bold text-rose-400 mt-2">
          {{ scores.filter((s) => (s.cyto_score ?? 0) < 40).length }}
        </span>
      </div>
    </div>

    <div class="glass-card overflow-x-auto p-0 border border-slate-800">
      <template v-if="hasData">
        <table class="w-full text-sm">
        <thead
          class="bg-slate-900/60 border-b border-slate-850 font-sans text-xs uppercase tracking-wider text-slate-400"
        >
          <tr>
            <th class="px-5 py-3.5 text-left font-semibold">Rider ID</th>
            <th class="px-5 py-3.5 text-left font-semibold">Battery ID</th>
            <th class="px-5 py-3.5 text-right font-semibold">
              Battery Health (BHI)
            </th>
            <th class="px-5 py-3.5 text-right font-semibold">
              Repayment Risk (RRI)
            </th>
            <th class="px-5 py-3.5 text-right font-semibold">CytoScore</th>
            <th class="px-5 py-3.5 text-center font-semibold">Flags</th>
            <th class="px-5 py-3.5 text-right font-semibold">
              Assessment Date
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          <tr
            v-for="s in scores"
            :key="s.id ?? s.rider_id"
            class="transition-colors hover:bg-slate-900/40"
          >
            <td class="px-5 py-4">
              <NuxtLink
                :to="`/riders/${s.rider_id}`"
                class="font-mono font-medium text-indigo-400 hover:text-indigo-300 hover:underline"
              >
                {{ short(s.rider_id) }}
              </NuxtLink>
            </td>
            <td class="px-5 py-4 font-mono text-slate-400 text-xs">
              {{ short(s.battery_id) }}
            </td>
            <td class="px-5 py-4 text-right font-mono text-slate-200">
              {{ (s.battery_health_index ?? 0).toFixed(1) }}
            </td>
            <td class="px-5 py-4 text-right font-mono text-slate-200">
              {{ (s.repayment_risk_index ?? 0).toFixed(1) }}
            </td>
            <td class="px-5 py-4 text-right">
              <span
                class="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold border font-mono"
                :class="riskClass(s.cyto_score ?? 0)"
              >
                {{ (s.cyto_score ?? 0).toFixed(1) }}
              </span>
            </td>
            <td class="px-5 py-4 text-center">
              <AnomalyBadge :flags="s.anomaly_flags" />
            </td>
            <td class="px-5 py-4 text-right font-mono text-xs text-slate-500">
              {{ s.scored_at?.slice(0, 10) }}
            </td>
          </tr>
          <tr v-if="scores.length === 0">
            <td colspan="7" class="px-5 py-10 text-center text-slate-500">
              No scored records found. Upload telemetry and repayment data to
              begin appraisal.
            </td>
          </tr>
        </tbody>
      </table>
      </template>

      <div v-if="errorMessage" class="px-5 py-10 text-center">
        <p class="text-rose-400">{{ errorMessage }}</p>
        <button
          class="mt-4 inline-flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
          @click="load"
        >
          Retry
        </button>
      </div>
      <div v-else-if="req.pending" class="px-5 py-10 text-center text-slate-400">
        Loading portfolio assessments…
      </div>
    </div>

    <!-- Operator view: swap-network fleet stress (distinct from lender risk) -->
    <div class="glass-card mt-6 p-5 border border-slate-800">
      <div class="mb-4 flex items-center justify-between gap-4">
        <div>
          <h2 class="text-sm font-semibold text-slate-200 uppercase tracking-wider">
            Swap-Network Fleet Stress
          </h2>
          <p class="mt-1 text-xs text-slate-400">
            Operator-facing high-stress patterns across the battery pool
            (thermal / deep-discharge). Separate from lender credit risk.
          </p>
        </div>
        <button
          class="shrink-0 rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
          @click="loadStress"
        >
          Refresh
        </button>
      </div>

      <div v-if="stressFlags.length === 0" class="py-4 text-xs text-slate-500">
        No high-stress swap patterns detected in the fleet.
      </div>
      <ul v-else class="divide-y divide-slate-800 text-xs">
        <li
          v-for="(f, i) in stressFlags"
          :key="i"
          class="flex flex-wrap items-center gap-2 py-3"
        >
          <span class="font-mono text-indigo-400">{{ short(f.entity_id) }}</span>
          <span
            class="rounded border px-2 py-0.5 text-[11px] font-semibold font-mono"
            :class="
              f.severity === 'high'
                ? 'text-rose-400 border-rose-500/20 bg-rose-500/10'
                : 'text-amber-400 border-amber-500/20 bg-amber-500/10'
            "
          >
            {{ f.rule_or_model }}
          </span>
          <span class="text-slate-400">{{ f.reason }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>
