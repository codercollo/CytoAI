<script setup lang="ts">
const toast = useToast();
const { apiKey, setApiKey } = useApiKey();
const hydrated = ref(false);
const lastWasEmpty = ref(true);
let keyDebounce: ReturnType<typeof setTimeout> | undefined;

onMounted(() => {
  lastWasEmpty.value = apiKey.value === "";
  hydrated.value = true;
});

watch(apiKey, (v) => {
  if (import.meta.client) {
    if (v) {
      localStorage.setItem("cyto_api_key", v);
    } else {
      localStorage.removeItem("cyto_api_key");
    }
  }

  // Only toast on real user edits, not the initial hydration set above.
  if (!hydrated.value) return;

  if (keyDebounce) clearTimeout(keyDebounce);
  keyDebounce = setTimeout(() => {
    const empty = v === "";
    if (empty && !lastWasEmpty.value) {
      toast.warning("Partner API key cleared — requests will be unauthenticated");
    } else if (!empty && lastWasEmpty.value) {
      toast.success("API key set");
    }
    lastWasEmpty.value = empty;
  }, 500);
});

onBeforeUnmount(() => {
  if (keyDebounce) clearTimeout(keyDebounce);
});
</script>

<template>
  <div
    class="min-h-screen text-slate-100 flex flex-col md:flex-row bg-[#080c14]"
  >
    <!-- Sidebar Navigation -->
    <aside
      class="w-full md:w-64 border-b md:border-b-0 md:border-r border-slate-800 bg-[#0c1220] flex flex-col justify-between shrink-0"
    >
      <div>
        <!-- Logo / Branding -->
        <div
          class="px-4 sm:px-6 py-4 sm:py-5 border-b border-slate-800/60 flex items-center justify-between"
        >
          <NuxtLink
            to="/"
            class="flex items-center gap-3 font-sans font-semibold tracking-tight text-slate-100"
          >
            <span
              class="inline-flex h-7 w-7 items-center justify-center rounded-lg bg-indigo-600 shadow-[0_2px_8px_rgba(99,102,241,0.3)] shrink-0"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                class="w-4 h-4 text-white"
              >
                <path
                  d="M12 9a1 1 0 0 1-1-1V3.07a8.001 8.001 0 0 0-7.93 7.93H9a1 1 0 0 1 1 1v6.93A8.001 8.001 0 0 0 17.93 11H12V9Z"
                />
                <path d="M11 2.07A8.007 8.007 0 0 1 18.93 10H11V2.07Z" />
              </svg>
            </span>
            <span class="text-base text-slate-200 font-bold">Cyto Risk</span>
          </NuxtLink>
          <span
            class="inline-flex items-center rounded-full bg-indigo-950/80 px-2 py-0.5 text-[10px] font-medium text-indigo-400 border border-indigo-800/40"
          >
            B2B v1.0
          </span>
        </div>

        <!-- Navigation Links -->
        <nav class="px-3 py-3 sm:py-4 flex flex-row md:flex-col overflow-x-auto md:overflow-x-visible gap-1 border-b md:border-b-0 border-slate-800/40">
          <NuxtLink
            to="/"
            exact-active-class="bg-indigo-600/10 text-indigo-400 border-indigo-500/50!"
            class="flex items-center gap-2 sm:gap-3 px-3 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm text-slate-400 hover:bg-slate-800/50 hover:text-slate-200 transition-colors border border-transparent font-medium shrink-0 whitespace-nowrap"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke-width="1.5"
              stroke="currentColor"
              class="w-4 h-4 sm:w-5 sm:h-5 shrink-0"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v2.25a2.25 2.25 0 0 1-2.25 2.25H6a2.25 2.25 0 0 1-2.25-2.25V6ZM3.75 15.75A2.25 2.25 0 0 1 6 13.5h2.25a2.25 2.25 0 0 1 2.25 2.25V18a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18v-2.25ZM13.5 6a2.25 2.25 0 0 1 2.25-2.25H18A2.25 2.25 0 0 1 20.25 6v2.25A2.25 2.25 0 0 1 18 10.5h-2.25a2.25 2.25 0 0 1-2.25-2.25V6ZM13.5 15.75a2.25 2.25 0 0 1 2.25-2.25H18a2.25 2.25 0 0 1 2.25 2.25V18A2.25 2.25 0 0 1 18 20.25h-2.25A2.25 2.25 0 0 1 13.5 18v-2.25Z"
              />
            </svg>
            Portfolio Index
          </NuxtLink>

          <NuxtLink
            to="/riders"
            exact-active-class="bg-indigo-600/10 text-indigo-400 border-indigo-500/50!"
            class="flex items-center gap-2 sm:gap-3 px-3 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm text-slate-400 hover:bg-slate-800/50 hover:text-slate-200 transition-colors border border-transparent font-medium shrink-0 whitespace-nowrap"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke-width="1.5"
              stroke="currentColor"
              class="w-4 h-4 sm:w-5 sm:h-5 shrink-0"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M15 19.128a9.38 9.38 0 0 0 2.625.372 9.337 9.337 0 0 0 4.121-.952 4.125 4.125 0 0 0-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 0 1 8.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0 1 11.964-3.07M12 6.375a3.375 3.375 0 1 1-6.75 0 3.375 3.375 0 0 1 6.75 0Zm8.25 2.25a2.625 2.625 0 1 1-5.25 0 2.625 2.625 0 0 1 5.25 0Z"
              />
            </svg>
            Riders
          </NuxtLink>

          <NuxtLink
            to="/upload"
            exact-active-class="bg-indigo-600/10 text-indigo-400 border-indigo-500/50!"
            class="flex items-center gap-2 sm:gap-3 px-3 py-2 sm:py-2.5 rounded-lg text-xs sm:text-sm text-slate-400 hover:bg-slate-800/50 hover:text-slate-200 transition-colors border border-transparent font-medium shrink-0 whitespace-nowrap"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke-width="1.5"
              stroke="currentColor"
              class="w-4 h-4 sm:w-5 sm:h-5 shrink-0"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 9.75v6.75m0 0-3-3m3 3 3-3m-8.25 6a9 9 0 1 1 18 0v-1.5a2.25 2.25 0 0 0-2.25-2.25h-13.5A2.25 2.25 0 0 0 3 18v1.5Z"
              />
            </svg>
            Data Ingestion
          </NuxtLink>
        </nav>
      </div>

      <!-- Settings / API Configuration -->
      <div class="p-3 sm:p-4 border-t border-slate-800 bg-slate-950/20">
        <label
          class="block text-[10px] font-semibold uppercase tracking-wider text-slate-500 mb-1.5 sm:mb-2"
        >
          PARTNER API KEY
        </label>
        <div class="relative">
          <input
            v-model="apiKey"
            type="password"
            placeholder="Enter partner API key...."
            autocomplete="off"
            class="w-full rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-200 placeholder:text-slate-600 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500/40"
          />
        </div>
      </div>
    </aside>

    <!-- Main Content Area -->
    <div class="flex-1 flex flex-col min-w-0">
      <main class="flex-1 px-4 sm:px-6 md:px-8 py-5 sm:py-8 max-w-6xl w-full mx-auto">
        <NuxtPage />
      </main>
    </div>

    <Toaster />
  </div>
</template>
