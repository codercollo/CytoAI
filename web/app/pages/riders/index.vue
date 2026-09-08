<script setup lang="ts">
import type {
  RiderResponse,
  RiderCreateRequest,
  RiderListResponse,
  BatteryCreateRequest,
  LoanCreateRequest,
  BatteryInfo,
} from "~/composables/useCytoApi";

const api = useCytoApi();
const toast = useToast();

const listReq = useCytoRequest<RiderListResponse>();
const createReq = useCytoRequest<RiderResponse>();
const linkReq = useCytoRequest<RiderResponse>();
const batteryReq = useCytoRequest<BatteryInfo>();

const riders = ref<RiderResponse[]>([]);

// ─── Register Rider modal ───────────────────────────────────────────────────
const modalOpen = ref(false);
const submitting = ref(false);
const formError = ref("");
const registrationMode = ref<"linked" | "swap_network" | "identity">("linked");

const form = reactive({
  external_ref: "",
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

// ─── Link Battery & Loan modal ─────────────────────────────────────────────
const linkModalOpen = ref(false);
const linkSubmitting = ref(false);
const linkFormError = ref("");
const linkTargetRider = ref<RiderResponse | null>(null);

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

// ─── Register Pool Battery modal ───────────────────────────────────────────
const poolBatteryModalOpen = ref(false);
const poolBatterySubmitting = ref(false);
const poolBatteryFormError = ref("");

const poolBatteryForm = reactive({
  external_ref: "",
  manufacturer: "",
  rated_capacity_wh: "",
  commissioned_at: "",
});

// ─── Shared style constants ─────────────────────────────────────────────────
const inputClass =
  "w-full rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-200 placeholder:text-slate-600 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500/40";

const labelClass =
  "block text-[10px] font-semibold uppercase tracking-wider text-slate-500 mb-2";

// ─── Data loading ───────────────────────────────────────────────────────────
async function load() {
  const res = await listReq.run(() => api.riders());
  if (res) riders.value = res.riders || [];
}

onMounted(load);

// ─── Register modal helpers ─────────────────────────────────────────────────
function openModal() {
  resetForm();
  formError.value = "";
  registrationMode.value = "linked";
  modalOpen.value = true;
}

function closeModal() {
  if (submitting.value) return;
  modalOpen.value = false;
}

function resetForm() {
  form.external_ref = "";
  form.battery_external_ref = "";
  form.manufacturer = "";
  form.rated_capacity_wh = "";
  form.commissioned_at = "";
  form.loan_external_ref = "";
  form.principal_kes = "";
  form.battery_value_kes = "";
  form.term_months = "";
  form.daily_installment_kes = "";
  form.started_at = "";
}

// ─── Link modal helpers ─────────────────────────────────────────────────────
function openLinkModal(rider: RiderResponse) {
  linkTargetRider.value = rider;
  resetLinkForm();
  linkFormError.value = "";
  linkModalOpen.value = true;
}

function closeLinkModal() {
  if (linkSubmitting.value) return;
  linkModalOpen.value = false;
  linkTargetRider.value = null;
}

function resetLinkForm() {
  linkForm.battery_external_ref = "";
  linkForm.manufacturer = "";
  linkForm.rated_capacity_wh = "";
  linkForm.commissioned_at = "";
  linkForm.loan_external_ref = "";
  linkForm.principal_kes = "";
  linkForm.battery_value_kes = "";
  linkForm.term_months = "";
  linkForm.daily_installment_kes = "";
  linkForm.started_at = "";
}

// ─── Pool Battery modal helpers ─────────────────────────────────────────────
function openPoolBatteryModal() {
  poolBatteryForm.external_ref = "";
  poolBatteryForm.manufacturer = "";
  poolBatteryForm.rated_capacity_wh = "";
  poolBatteryForm.commissioned_at = "";
  poolBatteryFormError.value = "";
  poolBatteryModalOpen.value = true;
}

function closePoolBatteryModal() {
  if (poolBatterySubmitting.value) return;
  poolBatteryModalOpen.value = false;
}

// ─── Shared utilities ───────────────────────────────────────────────────────
function trimOrUndef(v: string): string | undefined {
  const s = str(v);
  return s ? s : undefined;
}

function str(v: unknown): string {
  return v === null || v === undefined ? "" : String(v).trim();
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

// ─── Register form validation ───────────────────────────────────────────────
function validate(): string {
  if (!form.external_ref.trim()) return "Rider reference is required.";

  if (registrationMode.value === "identity") return "";

  if (!str(form.principal_kes)) return "Loan principal (KES) is required.";
  const principal = Number(form.principal_kes);
  if (!Number.isFinite(principal) || principal <= 0)
    return "Loan principal must be a positive number.";

  if (registrationMode.value === "linked") {
    if (!str(form.battery_value_kes)) return "Battery value (KES) is required.";
    const batteryValue = Number(form.battery_value_kes);
    if (!Number.isFinite(batteryValue) || batteryValue <= 0)
      return "Battery value must be a positive number.";

    if (str(form.rated_capacity_wh)) {
      const rc = Number(form.rated_capacity_wh);
      if (!Number.isFinite(rc) || rc <= 0)
        return "Rated capacity must be a positive number.";
    }
    if (
      str(form.commissioned_at) &&
      Number.isNaN(Date.parse(form.commissioned_at.trim()))
    ) {
      return "Battery commissioned date/time is invalid.";
    }
  }

  if (str(form.term_months)) {
    const tm = Number(form.term_months);
    if (!Number.isInteger(tm) || tm <= 0)
      return "Term months must be a positive whole number.";
  }
  if (str(form.daily_installment_kes)) {
    const di = Number(form.daily_installment_kes);
    if (!Number.isFinite(di) || di < 0)
      return "Daily installment must be a non-negative number.";
  }
  if (form.started_at && !isValidDate(form.started_at))
    return "Loan start date is invalid.";

  return "";
}

// ─── Link form validation ───────────────────────────────────────────────────
function validateLink(): string {
  if (!str(linkForm.principal_kes)) return "Loan principal (KES) is required.";
  if (!str(linkForm.battery_value_kes)) return "Battery value (KES) is required.";

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
  ) {
    return "Battery commissioned date/time is invalid.";
  }
  return "";
}

// ─── Register submit ────────────────────────────────────────────────────────
async function submit() {
  formError.value = "";
  const message = validate();
  if (message) {
    formError.value = message;
    return;
  }

  submitting.value = true;
  const payload: RiderCreateRequest = {
    external_ref: form.external_ref.trim(),
  };

  if (registrationMode.value === "linked") {
    payload.battery = {
      external_ref: trimOrUndef(form.battery_external_ref),
      manufacturer: trimOrUndef(form.manufacturer),
      rated_capacity_wh: numOrUndef(form.rated_capacity_wh),
      commissioned_at: trimOrUndef(form.commissioned_at),
    };
    payload.loan = {
      external_ref: trimOrUndef(form.loan_external_ref),
      principal_kes: Number(form.principal_kes),
      battery_value_kes: Number(form.battery_value_kes),
      term_months: numOrUndef(form.term_months),
      daily_installment_kes: numOrUndef(form.daily_installment_kes),
      started_at: trimOrUndef(form.started_at),
      financing_model: "leased_fixed",
    };
  } else if (registrationMode.value === "swap_network") {
    payload.loan = {
      external_ref: trimOrUndef(form.loan_external_ref),
      principal_kes: Number(form.principal_kes),
      term_months: numOrUndef(form.term_months),
      daily_installment_kes: numOrUndef(form.daily_installment_kes),
      started_at: trimOrUndef(form.started_at),
      financing_model: "swap_network",
    };
  }

  const created = await createReq.run(() => api.createRider(payload));
  submitting.value = false;
  if (created) {
    riders.value = [created, ...riders.value];
    modalOpen.value = false;
    resetForm();
    toast.success("Rider registered");
  } else {
    formError.value =
      createReq.error.value?.message || "Failed to register rider.";
  }
}

// ─── Link submit ────────────────────────────────────────────────────────────
async function submitLink() {
  linkFormError.value = "";
  const message = validateLink();
  if (message) {
    linkFormError.value = message;
    return;
  }
  if (!linkTargetRider.value) return;

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

  const targetId = linkTargetRider.value.id;
  const updated = await linkReq.run(() => api.linkRider(targetId, { battery, loan }));
  linkSubmitting.value = false;

  if (updated) {
    const idx = riders.value.findIndex((r) => r.id === targetId);
    if (idx !== -1) riders.value[idx] = updated;
    linkModalOpen.value = false;
    linkTargetRider.value = null;
    resetLinkForm();
    toast.success("Battery & loan linked — rider is now scorable");
  } else {
    linkFormError.value =
      linkReq.error.value?.message || "Failed to link battery & loan.";
  }
}

// ─── Pool Battery submit ───────────────────────────────────────────────────
async function submitPoolBattery() {
  poolBatteryFormError.value = "";
  if (!poolBatteryForm.external_ref.trim()) {
    poolBatteryFormError.value = "Battery reference is required.";
    return;
  }
  if (
    str(poolBatteryForm.rated_capacity_wh) &&
    Number(poolBatteryForm.rated_capacity_wh) <= 0
  ) {
    poolBatteryFormError.value = "Rated capacity must be positive.";
    return;
  }

  poolBatterySubmitting.value = true;
  const created = await batteryReq.run(() =>
    api.createBattery({
      external_ref: poolBatteryForm.external_ref.trim(),
      manufacturer: trimOrUndef(poolBatteryForm.manufacturer),
      rated_capacity_wh: numOrUndef(poolBatteryForm.rated_capacity_wh),
      commissioned_at: trimOrUndef(poolBatteryForm.commissioned_at),
    }),
  );
  poolBatterySubmitting.value = false;

  if (created) {
    poolBatteryModalOpen.value = false;
    toast.success(`Pool battery ${created.external_ref || created.id} registered`);
  } else {
    poolBatteryFormError.value =
      batteryReq.error.value?.message || "Failed to register pool battery.";
  }
}

// ─── Display helpers ────────────────────────────────────────────────────────
function short(id?: string): string {
  return id ? id.slice(0, 8) : "—";
}

function riskClass(v: number): string {
  if (v >= 70)
    return "text-emerald-400 bg-emerald-500/10 border-emerald-500/20";
  if (v >= 40) return "text-amber-400 bg-amber-500/10 border-amber-500/20";
  return "text-rose-400 bg-rose-500/10 border-rose-500/20";
}

function modeButtonClass(mode: "linked" | "swap_network" | "identity"): string {
  return registrationMode.value === mode
    ? "bg-indigo-600 text-white"
    : "text-slate-400 hover:text-slate-200";
}
</script>

<template>
  <div>
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 class="text-xl sm:text-2xl font-bold tracking-tight text-slate-100">Riders</h1>
        <p class="mt-1 text-xs sm:text-sm text-slate-400">
          Registered riders with battery, loan, and scoring status.
        </p>
      </div>
      <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 w-full sm:w-auto">
        <button
          type="button"
          class="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg border border-slate-700 bg-slate-800/80 px-3.5 py-2 text-xs font-semibold text-slate-200 transition-colors hover:bg-slate-700 shadow-sm"
          @click="openPoolBatteryModal"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            stroke-width="2"
            stroke="currentColor"
            class="w-3.5 h-3.5 text-emerald-400"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z"
            />
          </svg>
          Register Pool Battery
        </button>
        <button
          class="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition-all hover:bg-indigo-500 active:bg-indigo-700 shadow-sm"
          @click="openModal"
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
              d="M12 4.5v15m7.5-7.5h-15"
            />
          </svg>
          Register Rider
        </button>
      </div>
    </div>

    <!-- Auth Error Re-entry Card -->
    <AuthErrorCard
      v-if="listReq.error.value?.kind === 'auth'"
      class="mb-6"
      title="Authentication Failed (401/403)"
      message="Your partner API key is missing or invalid. Enter your key below to load riders."
      @retry="load"
    />

    <div class="glass-card overflow-x-auto p-0 border border-slate-800">
      <table class="w-full text-sm">
        <thead
          class="bg-slate-900/60 border-b border-slate-800/60 font-sans text-xs uppercase tracking-wider text-slate-400"
        >
          <tr>
            <th class="px-5 py-3.5 text-left font-semibold">Rider</th>
            <th class="px-5 py-3.5 text-left font-semibold">Status</th>
            <th class="px-5 py-3.5 text-left font-semibold">Battery / Fleet</th>
            <th class="px-5 py-3.5 text-left font-semibold">Loan</th>
            <th class="px-5 py-3.5 text-right font-semibold">CytoScore</th>
            <th class="px-5 py-3.5 text-right font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          <tr
            v-for="r in riders"
            :key="r.id"
            class="transition-colors hover:bg-slate-900/40"
          >
            <td class="px-5 py-4">
              <NuxtLink
                :to="`/riders/${r.id}`"
                class="font-medium text-slate-100 hover:text-indigo-400 hover:underline"
              >
                {{ r.external_ref || "—" }}
              </NuxtLink>
              <div class="mt-0.5 font-mono text-xs text-slate-500">
                {{ short(r.id) }}
              </div>
            </td>
            <td class="px-5 py-4">
              <span
                v-if="r.registration_status === 'registered'"
                class="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold border text-amber-400 bg-amber-500/10 border-amber-500/20"
              >
                Needs battery &amp; loan
              </span>
              <span
                v-else
                class="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold border text-indigo-400 bg-indigo-500/10 border-indigo-500/20"
              >
                Linked
              </span>
            </td>
            <td class="px-5 py-4 text-slate-300">
              <template v-if="r.loan?.financing_model === 'swap_network'">
                <span
                  class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-medium bg-emerald-950/50 text-emerald-400 border border-emerald-800/40 font-sans"
                >
                  Shared Fleet Pool
                </span>
              </template>
              <template v-else-if="r.battery">
                <div>{{ r.battery.external_ref || "—" }}</div>
                <div
                  v-if="r.battery.manufacturer"
                  class="text-xs text-slate-500"
                >
                  {{ r.battery.manufacturer }}
                </div>
              </template>
              <span v-else class="text-slate-600">—</span>
            </td>
            <td class="px-5 py-4 text-slate-300">
              <template v-if="r.loan">
                <div class="flex items-center gap-2">
                  <span>{{ r.loan.external_ref || "—" }}</span>
                  <span
                    class="rounded px-1.5 py-0.2 text-[9px] font-medium border"
                    :class="
                      r.loan.financing_model === 'swap_network'
                        ? 'bg-emerald-950/60 text-emerald-400 border-emerald-800/40'
                        : 'bg-slate-800 text-slate-400 border-slate-700'
                    "
                  >
                    {{
                      r.loan.financing_model === "swap_network"
                        ? "Swap"
                        : "Lease"
                    }}
                  </span>
                </div>
                <div
                  v-if="r.loan.principal_kes != null"
                  class="text-xs text-slate-500"
                >
                  KES {{ r.loan.principal_kes.toLocaleString() }}
                </div>
              </template>
              <span v-else class="text-slate-600">—</span>
            </td>
            <td class="px-5 py-4 text-right">
              <span
                v-if="r.latest_score"
                class="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold border font-mono"
                :class="riskClass(r.latest_score.cyto_score)"
              >
                {{ r.latest_score.cyto_score.toFixed(1) }}
              </span>
              <span v-else class="text-xs font-medium text-slate-500"
                >Not scored</span
              >
            </td>
            <td class="px-5 py-4">
              <div class="flex items-center justify-end gap-2">
                <NuxtLink
                  v-if="r.registration_status === 'linked'"
                  :to="`/riders/${r.id}`"
                  class="inline-flex items-center rounded-lg border border-indigo-700/40 bg-indigo-600/10 px-3 py-1.5 text-xs font-semibold text-indigo-400 transition-colors hover:bg-indigo-600/20"
                >
                  {{ r.latest_score ? "Appraisal" : "Score" }}
                </NuxtLink>
                <button
                  v-else
                  type="button"
                  class="inline-flex items-center gap-1.5 rounded-lg border border-amber-700/40 bg-amber-600/10 px-3 py-1.5 text-xs font-semibold text-amber-400 transition-colors hover:bg-amber-600/20"
                  @click="openLinkModal(r)"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke-width="2"
                    stroke="currentColor"
                    class="w-3 h-3"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244"
                    />
                  </svg>
                  Link
                </button>
                <NuxtLink
                  :to="`/riders/${r.id}`"
                  class="inline-flex items-center rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
                >
                  View
                </NuxtLink>
              </div>
            </td>
          </tr>

          <tr
            v-if="
              !listReq.pending.value &&
              !listReq.error.value &&
              riders.length === 0
            "
          >
            <td colspan="6" class="px-5 py-12 text-center">
              <p class="text-slate-400">No riders registered yet.</p>
              <button
                class="mt-4 inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition-all hover:bg-indigo-500 active:bg-indigo-700 shadow-sm"
                @click="openModal"
              >
                Register Rider
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <div
        v-if="listReq.pending.value"
        class="px-5 py-10 text-center text-slate-400"
      >
        Loading riders…
      </div>
      <div v-else-if="listReq.error.value" class="px-5 py-10 text-center">
        <p class="text-rose-400">{{ listReq.error.value?.message }}</p>
        <button
          class="mt-4 inline-flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-300 transition-colors hover:bg-slate-800/50"
          @click="load"
        >
          Retry
        </button>
      </div>
    </div>

    <!-- ── Register Rider Modal ───────────────────────────────────────────── -->
    <Teleport to="body">
      <div
        v-if="modalOpen"
        class="fixed inset-0 z-[90] flex items-center justify-center p-4"
      >
        <div class="absolute inset-0 bg-black/60" @click="closeModal"></div>
        <div
          class="relative w-full max-w-2xl glass-card border border-slate-700 max-h-[90vh] overflow-y-auto"
        >
          <div class="mb-5 flex items-start justify-between">
            <div>
              <h2 class="text-lg font-bold tracking-tight text-slate-100">
                Register Rider
              </h2>
              <p class="mt-1 text-xs text-slate-400">
                Create a rider with the details needed for scoring.
              </p>
            </div>
            <button
              type="button"
              :disabled="submitting"
              aria-label="Close"
              class="rounded-md p-1 text-slate-500 transition-colors hover:text-slate-300 disabled:opacity-50 disabled:cursor-not-allowed"
              @click="closeModal"
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

          <div
            class="grid grid-cols-3 gap-2 p-1 rounded-lg bg-slate-900/60 border border-slate-800 mb-5"
          >
            <button
              type="button"
              class="rounded-md px-3 py-2 text-xs font-semibold transition-colors text-center"
              :class="modeButtonClass('linked')"
              @click="registrationMode = 'linked'"
            >
              Full registration
            </button>
            <button
              type="button"
              class="rounded-md px-3 py-2 text-xs font-semibold transition-colors text-center"
              :class="modeButtonClass('swap_network')"
              @click="registrationMode = 'swap_network'"
            >
              Swap Network (Fleet)
            </button>
            <button
              type="button"
              class="rounded-md px-3 py-2 text-xs font-semibold transition-colors text-center"
              :class="modeButtonClass('identity')"
              @click="registrationMode = 'identity'"
            >
              Identity only
            </button>
          </div>

          <form class="space-y-5" @submit.prevent="submit">
            <div>
              <label :class="labelClass" for="external_ref"
                >Rider reference</label
              >
              <input
                id="external_ref"
                v-model="form.external_ref"
                :class="inputClass"
                placeholder="rider_123"
                autocomplete="off"
              />
            </div>

            <!-- Full registration fields -->
            <template v-if="registrationMode === 'linked'">
              <div class="rounded-xl border border-slate-800 p-4">
                <h3
                  class="mb-3 text-xs font-semibold uppercase tracking-wider text-indigo-400"
                >
                  Battery
                </h3>
                <div class="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label :class="labelClass" for="battery_external_ref"
                      >Battery reference</label
                    >
                    <input
                      id="battery_external_ref"
                      v-model="form.battery_external_ref"
                      :class="inputClass"
                      placeholder="battery_123"
                      autocomplete="off"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="manufacturer"
                      >Manufacturer</label
                    >
                    <input
                      id="manufacturer"
                      v-model="form.manufacturer"
                      :class="inputClass"
                      placeholder="Roam"
                      autocomplete="off"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="rated_capacity_wh"
                      >Rated capacity (Wh)</label
                    >
                    <input
                      id="rated_capacity_wh"
                      v-model="form.rated_capacity_wh"
                      type="number"
                      min="0"
                      step="any"
                      :class="inputClass"
                      placeholder="2000"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="commissioned_at"
                      >Commissioned at</label
                    >
                    <input
                      id="commissioned_at"
                      v-model="form.commissioned_at"
                      :class="inputClass"
                      placeholder="2026-08-01T00:00:00Z"
                    />
                    <p class="mt-1 text-[11px] text-slate-500">
                      Optional. e.g. 2026-08-01T00:00:00Z
                    </p>
                  </div>
                </div>
              </div>

              <div class="rounded-xl border border-slate-800 p-4">
                <h3
                  class="mb-3 text-xs font-semibold uppercase tracking-wider text-indigo-400"
                >
                  Loan
                </h3>
                <div class="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label :class="labelClass" for="loan_external_ref"
                      >Loan reference</label
                    >
                    <input
                      id="loan_external_ref"
                      v-model="form.loan_external_ref"
                      :class="inputClass"
                      placeholder="loan_123"
                      autocomplete="off"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="principal_kes"
                      >Principal (KES) <span class="text-rose-400">*</span></label
                    >
                    <input
                      id="principal_kes"
                      v-model="form.principal_kes"
                      type="number"
                      min="0"
                      step="any"
                      :class="inputClass"
                      placeholder="300000"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="battery_value_kes"
                      >Battery value (KES) <span class="text-rose-400">*</span></label
                    >
                    <input
                      id="battery_value_kes"
                      v-model="form.battery_value_kes"
                      type="number"
                      min="0"
                      step="any"
                      :class="inputClass"
                      placeholder="150000"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="term_months"
                      >Term (months)</label
                    >
                    <input
                      id="term_months"
                      v-model="form.term_months"
                      type="number"
                      min="1"
                      step="1"
                      :class="inputClass"
                      placeholder="12"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="daily_installment_kes"
                      >Daily installment (KES)</label
                    >
                    <input
                      id="daily_installment_kes"
                      v-model="form.daily_installment_kes"
                      type="number"
                      min="0"
                      step="any"
                      :class="inputClass"
                      placeholder="1000"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="started_at"
                      >Start date</label
                    >
                    <input
                      id="started_at"
                      v-model="form.started_at"
                      type="date"
                      :class="inputClass"
                    />
                  </div>
                </div>
              </div>
            </template>

            <!-- Swap Network (Fleet Pool) fields -->
            <template v-else-if="registrationMode === 'swap_network'">
              <div class="rounded-xl border border-emerald-900/40 bg-emerald-950/10 p-4">
                <div class="mb-3 flex items-center justify-between">
                  <h3
                    class="text-xs font-semibold uppercase tracking-wider text-emerald-400"
                  >
                    Swap Network Loan
                  </h3>
                  <span class="rounded bg-emerald-950/60 px-2 py-0.5 text-[10px] font-medium text-emerald-400 border border-emerald-800/40">
                    Shared Fleet Pool
                  </span>
                </div>
                <p class="mb-4 text-xs text-slate-400">
                  Swap-network riders use shared battery pools. No dedicated collateral battery is assigned.
                </p>
                <div class="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label :class="labelClass" for="swap_loan_external_ref"
                      >Loan reference</label
                    >
                    <input
                      id="swap_loan_external_ref"
                      v-model="form.loan_external_ref"
                      :class="inputClass"
                      placeholder="loan_swap_123"
                      autocomplete="off"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="swap_principal_kes"
                      >Principal (KES) <span class="text-rose-400">*</span></label
                    >
                    <input
                      id="swap_principal_kes"
                      v-model="form.principal_kes"
                      type="number"
                      min="0"
                      step="any"
                      :class="inputClass"
                      placeholder="300000"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="swap_term_months"
                      >Term (months)</label
                    >
                    <input
                      id="swap_term_months"
                      v-model="form.term_months"
                      type="number"
                      min="1"
                      step="1"
                      :class="inputClass"
                      placeholder="12"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="swap_daily_installment_kes"
                      >Daily installment (KES)</label
                    >
                    <input
                      id="swap_daily_installment_kes"
                      v-model="form.daily_installment_kes"
                      type="number"
                      min="0"
                      step="any"
                      :class="inputClass"
                      placeholder="1000"
                    />
                  </div>
                  <div>
                    <label :class="labelClass" for="swap_started_at"
                      >Start date</label
                    >
                    <input
                      id="swap_started_at"
                      v-model="form.started_at"
                      type="date"
                      :class="inputClass"
                    />
                  </div>
                </div>
              </div>
            </template>

            <!-- Identity only note -->
            <p
              v-else
              class="rounded-lg bg-amber-950/20 border border-amber-900/30 p-3 text-xs text-amber-300"
            >
              This rider will need a battery and loan linked before it can be
              scored. You can do this later using the <strong>Link</strong>
              action in the rider table.
            </p>

            <div
              v-if="formError"
              class="rounded-lg bg-rose-950/20 border border-rose-900/30 p-3 text-xs text-rose-300"
            >
              {{ formError }}
            </div>

            <div class="flex items-center justify-end gap-3 pt-1">
              <button
                type="button"
                :disabled="submitting"
                class="rounded-lg border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-300 transition-colors hover:bg-slate-800/50 disabled:opacity-50 disabled:cursor-not-allowed"
                @click="closeModal"
              >
                Cancel
              </button>
              <button
                type="submit"
                :disabled="submitting"
                class="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition-all hover:bg-indigo-500 active:bg-indigo-700 shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <svg
                  v-if="submitting"
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
                {{ submitting ? "Registering…" : "Register Rider" }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </Teleport>

    <!-- ── Link Battery & Loan Modal ─────────────────────────────────────── -->
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
                  linkTargetRider?.external_ref || linkTargetRider?.id
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
                  <label :class="labelClass" for="link_battery_external_ref"
                    >Battery reference</label
                  >
                  <input
                    id="link_battery_external_ref"
                    v-model="linkForm.battery_external_ref"
                    :class="inputClass"
                    placeholder="battery_123"
                    autocomplete="off"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_manufacturer"
                    >Manufacturer</label
                  >
                  <input
                    id="link_manufacturer"
                    v-model="linkForm.manufacturer"
                    :class="inputClass"
                    placeholder="Roam"
                    autocomplete="off"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_rated_capacity_wh"
                    >Rated capacity (Wh)</label
                  >
                  <input
                    id="link_rated_capacity_wh"
                    v-model="linkForm.rated_capacity_wh"
                    type="number"
                    min="0"
                    step="any"
                    :class="inputClass"
                    placeholder="2000"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_commissioned_at"
                    >Commissioned at</label
                  >
                  <input
                    id="link_commissioned_at"
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
                  <label :class="labelClass" for="link_loan_external_ref"
                    >Loan reference</label
                  >
                  <input
                    id="link_loan_external_ref"
                    v-model="linkForm.loan_external_ref"
                    :class="inputClass"
                    placeholder="loan_123"
                    autocomplete="off"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_principal_kes"
                    >Principal (KES) <span class="text-rose-400">*</span></label
                  >
                  <input
                    id="link_principal_kes"
                    v-model="linkForm.principal_kes"
                    type="number"
                    min="0"
                    step="any"
                    :class="inputClass"
                    placeholder="300000"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_battery_value_kes"
                    >Battery value (KES)
                    <span class="text-rose-400">*</span></label
                  >
                  <input
                    id="link_battery_value_kes"
                    v-model="linkForm.battery_value_kes"
                    type="number"
                    min="0"
                    step="any"
                    :class="inputClass"
                    placeholder="150000"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_term_months"
                    >Term (months)</label
                  >
                  <input
                    id="link_term_months"
                    v-model="linkForm.term_months"
                    type="number"
                    min="1"
                    step="1"
                    :class="inputClass"
                    placeholder="12"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_daily_installment_kes"
                    >Daily installment (KES)</label
                  >
                  <input
                    id="link_daily_installment_kes"
                    v-model="linkForm.daily_installment_kes"
                    type="number"
                    min="0"
                    step="any"
                    :class="inputClass"
                    placeholder="1000"
                  />
                </div>
                <div>
                  <label :class="labelClass" for="link_started_at"
                    >Start date</label
                  >
                  <input
                    id="link_started_at"
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

    <!-- ── Register Pool Battery Modal ────────────────────────────────────── -->
    <Teleport to="body">
      <div
        v-if="poolBatteryModalOpen"
        class="fixed inset-0 z-[90] flex items-center justify-center p-4"
      >
        <div class="absolute inset-0 bg-black/60" @click="closePoolBatteryModal"></div>
        <div
          class="relative w-full max-w-lg glass-card border border-slate-700 max-h-[90vh] overflow-y-auto"
        >
          <div class="mb-5 flex items-start justify-between">
            <div>
              <h2 class="text-lg font-bold tracking-tight text-slate-100">
                Register Pool Battery
              </h2>
              <p class="mt-1 text-xs text-slate-400">
                Pre-register a shared fleet battery for swap network operations.
              </p>
            </div>
            <button
              type="button"
              :disabled="poolBatterySubmitting"
              aria-label="Close"
              class="rounded-md p-1 text-slate-500 transition-colors hover:text-slate-300 disabled:opacity-50 disabled:cursor-not-allowed"
              @click="closePoolBatteryModal"
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

          <form class="space-y-4" @submit.prevent="submitPoolBattery">
            <div>
              <label :class="labelClass" for="pool_battery_ref"
                >Battery Reference <span class="text-rose-400">*</span></label
              >
              <input
                id="pool_battery_ref"
                v-model="poolBatteryForm.external_ref"
                :class="inputClass"
                placeholder="pool_1"
                autocomplete="off"
              />
            </div>
            <div>
              <label :class="labelClass" for="pool_manufacturer"
                >Manufacturer</label
              >
              <input
                id="pool_manufacturer"
                v-model="poolBatteryForm.manufacturer"
                :class="inputClass"
                placeholder="Spiro"
                autocomplete="off"
              />
            </div>
            <div>
              <label :class="labelClass" for="pool_capacity"
                >Rated capacity (Wh)</label
              >
              <input
                id="pool_capacity"
                v-model="poolBatteryForm.rated_capacity_wh"
                type="number"
                min="0"
                step="any"
                :class="inputClass"
                placeholder="2400"
              />
            </div>
            <div>
              <label :class="labelClass" for="pool_commissioned"
                >Commissioned at</label
              >
              <input
                id="pool_commissioned"
                v-model="poolBatteryForm.commissioned_at"
                :class="inputClass"
                placeholder="2026-01-01T00:00:00Z"
              />
            </div>

            <div
              v-if="poolBatteryFormError"
              class="rounded-lg bg-rose-950/20 border border-rose-900/30 p-3 text-xs text-rose-300"
            >
              {{ poolBatteryFormError }}
            </div>

            <div class="flex items-center justify-end gap-3 pt-2">
              <button
                type="button"
                :disabled="poolBatterySubmitting"
                class="rounded-lg border border-slate-700 px-4 py-2 text-sm font-semibold text-slate-300 transition-colors hover:bg-slate-800/50 disabled:opacity-50 disabled:cursor-not-allowed"
                @click="closePoolBatteryModal"
              >
                Cancel
              </button>
              <button
                type="submit"
                :disabled="poolBatterySubmitting"
                class="inline-flex items-center gap-2 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white transition-all hover:bg-emerald-500 active:bg-emerald-700 shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <svg
                  v-if="poolBatterySubmitting"
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
                {{ poolBatterySubmitting ? "Registering…" : "Register Pool Battery" }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </Teleport>
  </div>
</template>
