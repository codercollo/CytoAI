<script setup lang="ts">
import type {
  RiderResponse,
  ChartPoint,
  AnomalyFlag,
  ScoreResponse,
  BatteryCreateRequest,
  LoanCreateRequest,
} from "~/composables/useCytoApi";

const route = useRoute();
const api = useCytoApi();
const toast = useToast();
const req = useCytoRequest<RiderResponse>();
const computeReq = useCytoRequest<ScoreResponse>();
const linkReq = useCytoRequest<RiderResponse>();

const rider = ref<RiderResponse | null>(null);
const insufficientReason = ref<string | null>(null);

const score = computed(() => rider.value?.latest_score ?? null);
const factors = computed(() => rider.value?.factors);
const flags = computed<AnomalyFlag[]>(() => rider.value?.anomaly_flags ?? []);

// true when the rider has not been linked yet (identity-only)
const isUnlinked = computed(
  () => rider.value?.registration_status === "registered",
);

// For swap_network loans the BHI describes the fleet/pool the rider draws
// from, not a single collateral battery — label it distinctly so a reader
// does not mistake fleet health for "this rider's battery".
const isSwapNetwork = computed(
  () => rider.value?.loan?.financing_model === "swap_network",
);
const bhiLabel = computed(() =>
  isSwapNetwork.value ? "Fleet/Pool Health (BHI)" : "Battery Health (BHI)",
);

async function load() {
  insufficientReason.value = null;
  const res = await req.run(() => api.rider(route.params.id as string));
  if (res) rider.value = res;
}

async function triggerComputeScore() {
  if (!rider.value) return;
  insufficientReason.value = null;
  toast.info("Computing score for rider…");
  const res = await computeReq.run(() =>
    api.computeScore(rider.value!.id, rider.value?.battery?.id),
  );
  if (res) {
    if (res.insufficient_data) {
      insufficientReason.value =
        res.reason || "Insufficient history to compute a fair score.";
      toast.warning(
        `Not enough data to score: ${res.reason || "Insufficient records"}`,
      );
    } else {
      toast.success(`Score computed: CytoScore ${res.cyto_score.toFixed(1)}`);
      await load();
    }
  }
}

onMounted(load);

// ─── Link Battery & Loan modal (for identity-only riders on this detail page) ─
const linkModalOpen = ref(false);
const linkSubmitting = ref(false);
const linkFormError = ref("");

const inputClass =
  "w-full rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-200 placeholder:text-slate-600 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500/40";
const labelClass =
  "block text-[10px] font-semibold uppercase tracking-wider text-slate-500 mb-2";

const linkForm = reactive({
  battery_external_ref: "",
  manufacturer: "",
  rated_capacity_wh: "",
  commissioned_at: "",
  loan_external_ref: "",
  principal_kes: "",
  battery_value_kes: "",
  term_months: "",
  daily_installment_kes: "",
  started_at: "",
});

function openLinkModal() {
  Object.assign(linkForm, {
    battery_external_ref: "",
    manufacturer: "",
    rated_capacity_wh: "",
    commissioned_at: "",
    loan_external_ref: "",
    principal_kes: "",
    battery_value_kes: "",
    term_months: "",
    daily_installment_kes: "",
    started_at: "",
  });
  linkFormError.value = "";
  linkModalOpen.value = true;
}

function closeLinkModal() {
  if (linkSubmitting.value) return;
  linkModalOpen.value = false;
}

function str(v: unknown): string {
  return v === null || v === undefined ? "" : String(v).trim();
}
function trimOrUndef(v: string): string | undefined {
  const s = str(v);
  return s ? s : undefined;
}
function numOrUndef(v: string): number | undefined {
  const s = str(v);
  if (!s) return undefined;
  const n = Number(s);
  return Number.isFinite(n) ? n : undefined;
}
function isValidDate(v: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(v) && !Number.isNaN(Date.parse(v));
}

function validateLink(): string {
  if (!str(linkForm.principal_kes)) return "Loan principal (KES) is required.";
  if (!str(linkForm.battery_value_kes))
    return "Battery value (KES) is required.";
  const principal = Number(linkForm.principal_kes);
  if (!Number.isFinite(principal) || principal <= 0)
    return "Loan principal must be a positive number.";
  const batteryValue = Number(linkForm.battery_value_kes);
  if (!Number.isFinite(batteryValue) || batteryValue <= 0)
    return "Battery value must be a positive number.";
  if (str(linkForm.rated_capacity_wh)) {
    const rc = Number(linkForm.rated_capacity_wh);
    if (!Number.isFinite(rc) || rc <= 0)
      return "Rated capacity must be a positive number.";
  }
  if (str(linkForm.term_months)) {
    const tm = Number(linkForm.term_months);
    if (!Number.isInteger(tm) || tm <= 0)
      return "Term months must be a positive whole number.";
  }
  if (str(linkForm.daily_installment_kes)) {
    const di = Number(linkForm.daily_installment_kes);
    if (!Number.isFinite(di) || di < 0)
      return "Daily installment must be a non-negative number.";
  }
  if (linkForm.started_at && !isValidDate(linkForm.started_at))
    return "Loan start date is invalid.";
  if (
    str(linkForm.commissioned_at) &&
    Number.isNaN(Date.parse(linkForm.commissioned_at.trim()))
  )
    return "Battery commissioned date/time is invalid.";
  return "";
}

async function submitLink() {
  linkFormError.value = "";
  const msg = validateLink();
  if (msg) {
    linkFormError.value = msg;
    return;
  }
  if (!rider.value) return;

  linkSubmitting.value = true;
  const battery: BatteryCreateRequest = {
    external_ref: trimOrUndef(linkForm.battery_external_ref),
    manufacturer: trimOrUndef(linkForm.manufacturer),
    rated_capacity_wh: numOrUndef(linkForm.rated_capacity_wh),
    commissioned_at: trimOrUndef(linkForm.commissioned_at),
  };
  const loan: LoanCreateRequest = {
    external_ref: trimOrUndef(linkForm.loan_external_ref),
    principal_kes: Number(linkForm.principal_kes),
    battery_value_kes: Number(linkForm.battery_value_kes),
    term_months: numOrUndef(linkForm.term_months),
    daily_installment_kes: numOrUndef(linkForm.daily_installment_kes),
    started_at: trimOrUndef(linkForm.started_at),
  };

  const updated = await linkReq.run(() =>
    api.linkRider(rider.value!.id, { battery, loan }),
  );
  linkSubmitting.value = false;

  if (updated) {
    rider.value = updated;
    linkModalOpen.value = false;
    toast.success("Battery & loan linked — rider is now scorable");
  } else {
    linkFormError.value =
      linkReq.error.value?.message || "Failed to link battery & loan.";
  }
}

// ─── Chart / factor helpers ─────────────────────────────────────────────────
const points = computed<ChartPoint[]>(() => {
  if (!score.value) return [];
  return [
    {
      t: score.value.scored_at || "latest",
      bhi: score.value.battery_health_index ?? 0,
      rri: score.value.repayment_risk_index ?? 0,
    },
  ];
});

// Simple, documented MVP thresholds for the qualitative checklist lines.
const onTimePct = computed(() =>
  Math.round((factors.value?.repayment.on_time_ratio ?? 0) * 100),
);
const cadence = computed(
  () => factors.value?.repayment.telemetry_cadence_proxy ?? 0,
);
const onTimeOk = computed(
  () => (factors.value?.repayment.on_time_ratio ?? 0) >= 0.8,
);
const daysLateOk = computed(
  () => (factors.value?.repayment.avg_days_late ?? 0) <= 2,
);
const cadenceOk = computed(() => cadence.value <= 0.5);
const tempOk = computed(() => {
  const t = factors.value?.battery.avg_temperature_c ?? 0;
  return t >= 18 && t <= 32;
});
</script>


<template>
  <div>
    <NuxtLink
      to="/"
      class="inline-flex items-center gap-1 text-sm text-slate-400 hover:text-indigo-400 transition-colors"
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
          d="M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18"
        />
      </svg>
      Back to Portfolio
    </NuxtLink>

    <!-- Auth Error Re-entry Card -->
    <AuthErrorCard
      v-if="req.error.value?.kind === 'auth'"
      class="mt-6"
      title="Authentication Failed (401/403)"
      message="Your partner API key is missing or unauthorized to view this rider. Enter your key below."
      @retry="load"
    />

    <div v-else-if="req.pending.value" class="mt-8 text-slate-400">
      Loading appraisal metrics…
    </div>

    <div v-else-if="req.error.value" class="mt-8">
      <p class="text-rose-400">{{ req.error.value?.message }}</p>
      <button
        class="mt-3 inline-flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
        @click="load"
      >
        Retry
      </button>
    </div>

    <template v-else-if="rider">
      <div
        class="mt-4 mb-6 flex flex-col md:flex-row md:items-center justify-between gap-4"
      >
        <div>
          <h1 class="text-2xl font-bold tracking-tight text-slate-100">
            Rider Appraisal Profile
          </h1>
          <div
            class="mt-1 flex flex-wrap items-center gap-2 text-sm text-slate-400"
          >
            <span>Reference ID:</span>
            <span
              class="font-mono text-indigo-400 bg-indigo-950/40 px-2 py-0.5 rounded border border-indigo-900/30 text-xs"
              >{{ route.params.id }}</span
            >
            <span v-if="rider.external_ref" class="text-xs text-slate-500"
              >({{ rider.external_ref }})</span
            >
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <!-- Financing model badge — only for linked riders -->
          <span
            v-if="!isUnlinked"
            class="rounded px-2.5 py-1 text-xs font-semibold border"
            :class="
              isSwapNetwork
                ? 'bg-emerald-950/60 text-emerald-400 border-emerald-800/40'
                : 'bg-indigo-950/60 text-indigo-400 border-indigo-800/40'
            "
          >
            {{
              isSwapNetwork
                ? "Swap Network (Shared Fleet)"
                : "Leased Fixed (Dedicated)"
            }}
          </span>

          <!-- Unlinked rider: show Link Battery & Loan CTA instead of Score -->
          <template v-if="isUnlinked">
            <span
              class="inline-flex items-center px-2.5 py-1 rounded text-xs font-semibold border text-amber-400 bg-amber-500/10 border-amber-500/20"
            >
              Needs battery &amp; loan
            </span>
            <button
              class="inline-flex items-center gap-1.5 rounded-lg bg-amber-600 px-3 py-1.5 text-xs font-semibold text-white transition-all hover:bg-amber-500 active:bg-amber-700 shadow-sm"
              @click="openLinkModal"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke-width="2"
                stroke="currentColor"
                class="w-3.5 h-3.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244"
                />
              </svg>
              Link Battery &amp; Loan
            </button>
          </template>

          <!-- Linked rider: show model badge + Compute Score button -->
          <template v-else>
            <div
              class="flex items-center gap-2 text-xs text-slate-400 bg-slate-900/60 border border-slate-800 rounded-lg px-3 py-1.5"
            >
              <span
                class="inline-block h-2 w-2 rounded-full bg-emerald-500"
              ></span>
              <span>Model: {{ score?.model_version || "v1.0" }}</span>
            </div>

            <button
              :disabled="computeReq.pending.value"
              class="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white transition-all hover:bg-indigo-500 active:bg-indigo-700 shadow-sm disabled:opacity-50"
              @click="triggerComputeScore"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke-width="2"
                stroke="currentColor"
                class="w-3.5 h-3.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"
                />
              </svg>
              {{
                computeReq.pending.value
                  ? "Scoring…"
                  : score
                    ? "Re-compute Score"
                    : "Compute Score"
              }}
            </button>
          </template>
        </div>
      </div>

      <!-- Insufficient Data Notification Banner -->
      <div
        v-if="insufficientReason"
        class="mb-6 rounded-xl border border-amber-800/40 bg-amber-950/20 p-4 text-xs text-amber-300"
      >
        <div class="flex items-center gap-2 font-semibold text-amber-200 mb-1">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            class="w-4 h-4 text-amber-400"
          >
            <path
              fill-rule="evenodd"
              d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0Zm-7-4a1 1 0 1 1-2 0 1 1 0 0 1 2 0ZM9 9a.75.75 0 0 0 0 1.5h.253a.25.25 0 0 1 .244.304l-.459 2.066A1.75 1.75 0 0 0 10.747 15H11a.75.75 0 0 0 0-1.5h-.253a.25.25 0 0 1-.244-.304l.459-2.066A1.75 1.75 0 0 0 9.253 9H9Z"
              clip-rule="evenodd"
            />
          </svg>
          Thin-File Assessment: Insufficient History to Score Fairly
        </div>
        <p>{{ insufficientReason }}</p>
        <p class="mt-2 text-[11px] text-slate-400">
          Under bias-monitoring policy (spec.md §9), thin-file riders are
          protected from arbitrary low scores until sufficient operational
          history is ingested.
        </p>
      </div>

      <template v-if="score">
        <!-- Score gauges row -->
        <div class="grid gap-4 sm:gap-6 grid-cols-1 sm:grid-cols-3">
          <div class="glass-card flex flex-col items-center justify-center p-4 sm:p-6">
            <ScoreGauge
              :value="score.cyto_score ?? 0"
              label="Combined CytoScore"
            />
          </div>
          <div class="glass-card flex flex-col items-center justify-center p-4 sm:p-6">
            <ScoreGauge
              :value="score.battery_health_index ?? 0"
              :label="bhiLabel"
            />
          </div>
          <div class="glass-card flex flex-col items-center justify-center p-4 sm:p-6">
            <ScoreGauge
              :value="100 - (score.repayment_risk_index ?? 0)"
              label="Repayment Safety Index"
            />
          </div>
        </div>

        <!-- History and Details section -->
        <div class="grid gap-6 grid-cols-1 md:grid-cols-3 mt-6">
          <div class="glass-card md:col-span-2">
            <h2
              class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider"
            >
              Risk Trend History
            </h2>
            <DegradationChart :points="points" :bhi-label="bhiLabel" />
          </div>

          <div class="glass-card flex flex-col justify-between">
            <div>
              <h2
                class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider"
              >
                Evaluation Details
              </h2>
              <div class="divide-y divide-slate-800 text-xs">
                <div class="py-3 flex justify-between">
                  <span class="text-slate-400">Financing Model</span>
                  <span class="font-semibold text-slate-200">{{
                    isSwapNetwork ? "Swap Network" : "Leased Fixed"
                  }}</span>
                </div>
                <div class="py-3 flex justify-between">
                  <span class="text-slate-400">BHI Scope</span>
                  <span class="font-mono text-slate-200">{{
                    isSwapNetwork ? "Shared Fleet Pool" : "Dedicated Battery"
                  }}</span>
                </div>
                <div class="py-3 flex justify-between">
                  <span class="text-slate-400">{{ bhiLabel }}</span>
                  <span class="font-mono font-semibold text-slate-200">{{
                    (score.battery_health_index ?? 0).toFixed(1)
                  }}</span>
                </div>
                <div class="py-3 flex justify-between">
                  <span class="text-slate-400">Repayment Risk (RRI)</span>
                  <span class="font-mono font-semibold text-slate-200">{{
                    (score.repayment_risk_index ?? 0).toFixed(1)
                  }}</span>
                </div>
                <div class="py-3 flex justify-between">
                  <span class="text-slate-400">Combined CytoScore</span>
                  <span class="font-mono font-semibold text-indigo-400">{{
                    (score.cyto_score ?? 0).toFixed(1)
                  }}</span>
                </div>
                <div class="py-3 flex justify-between">
                  <span class="text-slate-400">Scored At</span>
                  <span class="font-mono text-slate-200">{{
                    score.scored_at || "—"
                  }}</span>
                </div>
              </div>
            </div>

            <div
              class="mt-6 pt-4 border-t border-slate-800 text-[11px] text-slate-500 leading-normal"
            >
              This risk evaluation is generated via green EV telematics paired
              with repayment data models.
            </div>
          </div>
        </div>

        <!-- Contributing factors checklist (mvp.md §5) -->
        <div class="glass-card mt-6 p-5">
          <h2
            class="mb-4 text-sm font-semibold text-slate-200 uppercase tracking-wider"
          >
            Contributing Factors
          </h2>
          <div v-if="factors" class="grid gap-6 grid-cols-1 sm:grid-cols-2 md:grid-cols-3">
            <div>
              <h3
                class="mb-3 text-xs font-semibold text-indigo-400 uppercase tracking-wider"
              >
                Repayment
              </h3>
              <ul class="space-y-2 text-xs text-slate-300">
                <li class="flex items-center gap-2">
                  <span v-if="onTimeOk" class="font-bold text-emerald-400"
                    >✓</span
                  ><span v-else class="font-bold text-amber-400">!</span>
                  <span>{{ onTimePct }}% payments on time</span>
                </li>
                <li class="flex items-center gap-2">
                  <span v-if="daysLateOk" class="font-bold text-emerald-400"
                    >✓</span
                  ><span v-else class="font-bold text-amber-400">!</span>
                  <span
                    >{{
                      (factors.repayment.avg_days_late ?? 0).toFixed(1)
                    }}
                    days late (avg)</span
                  >
                </li>
                <li class="flex items-center gap-2">
                  <span class="font-bold text-slate-600">•</span>
                  <span
                    >Payment cadence
                    {{
                      (factors.repayment.payment_cadence_proxy ?? 0).toFixed(2)
                    }}
                    std days</span
                  >
                </li>
                <li class="flex items-center gap-2">
                  <span class="font-bold text-slate-600">•</span>
                  <span
                    >Loan-to-battery
                    {{
                      (
                        factors.repayment.loan_to_battery_value_ratio ?? 0
                      ).toFixed(2)
                    }}</span
                  >
                </li>
                <li class="flex items-center gap-2">
                  <span class="font-bold text-slate-600">•</span>
                  <span
                    >{{ (factors.repayment.tenure_days ?? 0).toFixed(0) }} days
                    tenure</span
                  >
                </li>
                <!-- Swap-network specific factor: battery stress profile vs fleet -->
                <li
                  v-if="
                    isSwapNetwork &&
                    factors.repayment.battery_stress_profile != null
                  "
                  class="flex items-center gap-2"
                >
                  <span class="font-bold text-emerald-400">•</span>
                  <span
                    >Fleet stress profile
                    {{
                      (factors.repayment.battery_stress_profile ?? 0).toFixed(2)
                    }}
                    (relative to fleet)</span
                  >
                </li>
              </ul>
            </div>

            <div>
              <h3
                class="mb-3 text-xs font-semibold text-indigo-400 uppercase tracking-wider"
              >
                Activity
              </h3>
              <ul class="space-y-2 text-xs text-slate-300">
                <li class="flex items-center gap-2">
                  <span v-if="cadenceOk" class="font-bold text-emerald-400"
                    >✓</span
                  ><span v-else class="font-bold text-amber-400">!</span>
                  <span
                    >Battery swap/usage cadence
                    {{ cadence.toFixed(3) }} (regularity)</span
                  >
                </li>
              </ul>
            </div>

            <div>
              <h3
                class="mb-3 text-xs font-semibold text-indigo-400 uppercase tracking-wider"
              >
                {{
                  isSwapNetwork ? "Fleet Battery Pool" : "Collateral Battery"
                }}
              </h3>
              <ul class="space-y-2 text-xs text-slate-300">
                <li class="flex items-center gap-2">
                  <span v-if="tempOk" class="font-bold text-emerald-400">✓</span
                  ><span v-else class="font-bold text-amber-400">!</span>
                  <span
                    >{{ (factors.battery.avg_temperature_c ?? 0).toFixed(1) }}°C
                    average temperature</span
                  >
                </li>
                <li class="flex items-center gap-2">
                  <span class="font-bold text-slate-600">•</span>
                  <span
                    >{{ (factors.battery.cycle_count ?? 0).toFixed(0) }} charge
                    cycles</span
                  >
                </li>
                <li class="flex items-center gap-2">
                  <span class="font-bold text-slate-600">•</span>
                  <span
                    >{{
                      (factors.battery.avg_depth_of_discharge ?? 0).toFixed(1)
                    }}% average depth of discharge</span
                  >
                </li>
                <li class="flex items-center gap-2">
                  <span class="font-bold text-slate-600">•</span>
                  <span
                    >{{ (factors.battery.age_days ?? 0).toFixed(0) }} days
                    age</span
                  >
                </li>
              </ul>
            </div>
          </div>
        </div>

        <!-- Anomaly review -->
        <div class="glass-card mt-6 p-5">
          <h2
            class="mb-3 text-sm font-semibold text-slate-200 uppercase tracking-wider"
          >
            Anomaly Review
          </h2>
          <p v-if="flags.length === 0" class="text-xs text-slate-500">
            No anomalies detected for this rider.
          </p>
          <ul v-else class="space-y-2">
            <li
              v-for="(f, i) in flags"
              :key="i"
              class="flex items-start gap-2 text-xs"
            >
              <AnomalyBadge :flags="[f]" />
              <span class="text-slate-300">{{ f.reason }}</span>
            </li>
          </ul>
          <p class="mt-3 text-[11px] text-slate-500">
            Anomalies are review signals, not rejections (spec.md §9).
          </p>
        </div>
      </template>

      <!-- Empty state when no score exists for rider yet -->
      <div
        v-else
        class="glass-card mt-8 p-8 text-center border border-slate-800"
      >
        <div
          class="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-slate-900 border border-slate-800 text-slate-400 mb-3"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            stroke-width="1.5"
            stroke="currentColor"
            class="w-5 h-5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
            />
          </svg>
        </div>

        <!-- Unlinked (identity-only) rider: needs battery & loan first -->
        <template v-if="isUnlinked">
          <h3 class="text-sm font-semibold text-slate-200">
            Battery &amp; Loan Required
          </h3>
          <p class="mt-1 text-xs text-slate-400 max-w-sm mx-auto">
            This rider was registered without a battery or loan. Link them
            before triggering a score.
          </p>
          <div class="mt-4 flex items-center justify-center gap-3">
            <button
              class="inline-flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-xs font-semibold text-white transition-all hover:bg-amber-500 shadow-sm"
              @click="openLinkModal"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke-width="2"
                stroke="currentColor"
                class="w-3.5 h-3.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244"
                />
              </svg>
              Link Battery &amp; Loan
            </button>
            <NuxtLink
              to="/riders"
              class="inline-flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-xs font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
            >
              Back to Riders
            </NuxtLink>
          </div>
        </template>

        <!-- Linked rider with no score yet: show Compute Score button -->
        <template v-else>
          <h3 class="text-sm font-semibold text-slate-200">
            No Score Appraisal Record Yet
          </h3>
          <p class="mt-1 text-xs text-slate-400 max-w-sm mx-auto">
            This rider has not been evaluated yet. You can trigger an on-demand
            score calculation or ingest telemetry / repayments.
          </p>
          <div class="mt-4 flex items-center justify-center gap-3">
            <button
              :disabled="computeReq.pending.value"
              class="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition-all hover:bg-indigo-500 shadow-sm disabled:opacity-50"
              @click="triggerComputeScore"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke-width="2"
                stroke="currentColor"
                class="w-3.5 h-3.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z"
                />
              </svg>
              {{
                computeReq.pending.value
                  ? "Computing Score…"
                  : "Compute Score Now"
              }}
            </button>
            <NuxtLink
              to="/upload"
              class="inline-flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-xs font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
            >
              Upload Data
            </NuxtLink>
          </div>
        </template>
      </div>
    </template>
  </div>

  <!-- ── Link Battery & Loan Modal ───────────────────────────────────────── -->
  <Teleport to="body">
    <div
      v-if="linkModalOpen"
      class="fixed inset-0 z-[90] flex items-center justify-center p-4"
    >
      <div class="absolute inset-0 bg-black/60" @click="closeLinkModal"></div>
      <div
        class="relative w-full max-w-2xl glass-card border border-slate-700 max-h-[90vh] overflow-y-auto"
      >
        <div class="mb-5 flex items-start justify-between">
          <div>
            <h2 class="text-lg font-bold tracking-tight text-slate-100">
              Link Battery &amp; Loan
            </h2>
            <p class="mt-1 text-xs text-slate-400">
              Attach a battery and loan to
              <span class="font-mono text-indigo-400">{{
                rider?.external_ref || rider?.id
              }}</span>
              so it can be scored.
            </p>
          </div>
          <button
            type="button"
            :disabled="linkSubmitting"
            aria-label="Close"
            class="rounded-md p-1 text-slate-500 transition-colors hover:text-slate-300 disabled:opacity-50 disabled:cursor-not-allowed"
            @click="closeLinkModal"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke-width="1.5"
              stroke="currentColor"
              class="h-5 w-5"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M6 18 18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        <form class="space-y-5" @submit.prevent="submitLink">
          <!-- Battery section -->
          <div class="rounded-xl border border-slate-800 p-4">
            <h3
              class="mb-3 text-xs font-semibold uppercase tracking-wider text-indigo-400"
            >
              Battery
            </h3>
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label :class="labelClass" for="id_link_battery_ref"
                  >Battery reference</label
                >
                <input
                  id="id_link_battery_ref"
                  v-model="linkForm.battery_external_ref"
                  :class="inputClass"
                  placeholder="battery_123"
                  autocomplete="off"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_manufacturer"
                  >Manufacturer</label
                >
                <input
                  id="id_link_manufacturer"
                  v-model="linkForm.manufacturer"
                  :class="inputClass"
                  placeholder="Roam"
                  autocomplete="off"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_capacity"
                  >Rated capacity (Wh)</label
                >
                <input
                  id="id_link_capacity"
                  v-model="linkForm.rated_capacity_wh"
                  type="number"
                  min="0"
                  step="any"
                  :class="inputClass"
                  placeholder="2000"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_commissioned"
                  >Commissioned at</label
                >
                <input
                  id="id_link_commissioned"
                  v-model="linkForm.commissioned_at"
                  :class="inputClass"
                  placeholder="2026-08-01T00:00:00Z"
                />
                <p class="mt-1 text-[11px] text-slate-500">
                  Optional. e.g. 2026-08-01T00:00:00Z
                </p>
              </div>
            </div>
          </div>

          <!-- Loan section -->
          <div class="rounded-xl border border-slate-800 p-4">
            <h3
              class="mb-3 text-xs font-semibold uppercase tracking-wider text-indigo-400"
            >
              Loan
            </h3>
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <label :class="labelClass" for="id_link_loan_ref"
                  >Loan reference</label
                >
                <input
                  id="id_link_loan_ref"
                  v-model="linkForm.loan_external_ref"
                  :class="inputClass"
                  placeholder="loan_123"
                  autocomplete="off"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_principal"
                  >Principal (KES) <span class="text-rose-400">*</span></label
                >
                <input
                  id="id_link_principal"
                  v-model="linkForm.principal_kes"
                  type="number"
                  min="0"
                  step="any"
                  :class="inputClass"
                  placeholder="300000"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_batt_value"
                  >Battery value (KES)
                  <span class="text-rose-400">*</span></label
                >
                <input
                  id="id_link_batt_value"
                  v-model="linkForm.battery_value_kes"
                  type="number"
                  min="0"
                  step="any"
                  :class="inputClass"
                  placeholder="150000"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_term"
                  >Term (months)</label
                >
                <input
                  id="id_link_term"
                  v-model="linkForm.term_months"
                  type="number"
                  min="1"
                  step="1"
                  :class="inputClass"
                  placeholder="12"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_daily"
                  >Daily installment (KES)</label
                >
                <input
                  id="id_link_daily"
                  v-model="linkForm.daily_installment_kes"
                  type="number"
                  min="0"
                  step="any"
                  :class="inputClass"
                  placeholder="1000"
                />
              </div>
              <div>
                <label :class="labelClass" for="id_link_started"
                  >Start date</label
                >
                <input
                  id="id_link_started"
                  v-model="linkForm.started_at"
                  type="date"
                  :class="inputClass"
                />
              </div>
            </div>
          </div>

          <div
            v-if="linkFormError"
            class="rounded-lg bg-rose-950/20 border border-rose-900/30 p-3 text-xs text-rose-300"
          >
            {{ linkFormError }}
          </div>

          <div class="flex items-center justify-end gap-3 pt-1">
            <button
              type="button"
              :disabled="linkSubmitting"
              class="rounded-lg border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-300 transition-colors hover:bg-slate-800/50 disabled:opacity-50 disabled:cursor-not-allowed"
              @click="closeLinkModal"
            >
              Cancel
            </button>
            <button
              type="submit"
              :disabled="linkSubmitting"
              class="inline-flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-semibold text-white transition-all hover:bg-amber-500 active:bg-amber-700 shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <svg
                v-if="linkSubmitting"
                class="h-4 w-4 animate-spin"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 0 1 8-8v4a4 4 0 0 0-4 4H4z"
                ></path>
              </svg>
              <svg
                v-else
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke-width="2"
                stroke="currentColor"
                class="w-3.5 h-3.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244"
                />
              </svg>
              {{ linkSubmitting ? "Linking…" : "Link Battery & Loan" }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

