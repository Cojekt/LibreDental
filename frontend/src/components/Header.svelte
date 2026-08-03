<script lang="ts">
  import type { CountryConfig } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import NavigationTabs from "./NavigationTabs.svelte";

  let {
    activeTab = $bindable("patients"),
    countryMeta,
    onnewpatient,
    onnewappointment,
  } = $props<{
    activeTab: string;
    countryMeta?: CountryConfig | null;
    onnewpatient: () => void;
    onnewappointment: () => void;
  }>();
</script>

<header class="w-full border-b border-slate-800 bg-slate-900 shadow-sm">
  <!-- Top brand bar -->
  <div class="flex h-16 w-full items-center justify-between px-6 sm:px-8">
    <div class="flex items-center gap-3">
      <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-cyan-500 text-white shadow-md shadow-cyan-500/20">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="h-[22px] w-[22px]"
        >
          <path
            d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 14.5v-9l6 4.5-6 4.5z"
          />
        </svg>
      </div>
      <div class="flex items-center">
        <h1 class="m-0 text-xl font-bold tracking-tight text-slate-50">
          LibreDental<span class="align-super text-[11px] text-slate-400">™</span>
        </h1>
        {#if countryMeta}
          <span class="ml-2.5 rounded-xl border border-slate-700 bg-slate-800/80 px-2.5 py-0.5 text-[11px] font-medium text-slate-300 flex items-center gap-1">
            📍 {countryMeta.name || countryMeta.code}
          </span>
        {/if}
      </div>
    </div>

    <div class="flex items-center gap-3">
      {#if activeTab === "patients"}
        <button class="btn btn-primary shadow-md shadow-sky-500/20" onclick={onnewpatient}>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-5 w-5"
          >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          New Patient
        </button>
      {:else if activeTab === "appointments"}
        <button class="btn btn-primary shadow-md shadow-sky-500/20" onclick={onnewappointment}>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-5 w-5"
          >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          New Appointment
        </button>
      {/if}
    </div>
  </div>

  <!-- Navigation Tabs -->
  <NavigationTabs bind:activeTab ontabchange={(t) => (activeTab = t)} />
</header>
