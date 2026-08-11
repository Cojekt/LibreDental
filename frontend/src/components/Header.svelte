<script lang="ts">
  import type { CountryConfig } from "@bindings/domain/models.js";
  import NavigationTabs from "./NavigationTabs.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";

  let {
    activeTab = $bindable("clinic"),
    countryMeta,
    onnewpatient,
    onnewappointment,
    onopensettings,
  } = $props<{
    activeTab: string;
    countryMeta?: CountryConfig | null;
    onnewpatient: () => void;
    onnewappointment: () => void;
    onopensettings: () => void;
  }>();
</script>

<header class="w-full border-b border-slate-800 bg-slate-900 shadow-sm">
  <!-- Top brand bar -->
  <div class="flex h-16 w-full items-center justify-between px-6 sm:px-8">
    <div class="flex items-center gap-3">
      <div
        class="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-cyan-500 text-white shadow-md shadow-cyan-500/20"
      >
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
        <h1 class="m-0 text-xl font-bold tracking-tight text-slate-50">LibreDental</h1>
        {#if countryMeta}
          <span
            class="ml-2.5 rounded-xl border border-slate-700 bg-slate-800/80 px-2.5 py-0.5 text-[11px] font-medium text-slate-300 flex items-center gap-1"
          >
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
          {(getLocaleVersion(), m.header_new_patient())}
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
          {(getLocaleVersion(), m.header_new_appointment())}
        </button>
      {/if}

      <button
        type="button"
        onclick={onopensettings}
        class="flex h-9 w-9 items-center justify-center rounded-xl border border-slate-700/80 bg-slate-800/90 text-slate-300 hover:bg-slate-700 hover:text-white transition-all shadow-sm cursor-pointer"
        title={m.settings_title()}
        aria-label={m.header_settings_label()}
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="h-5 w-5"
        >
          <circle cx="12" cy="12" r="3"></circle>
          <path
            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
          ></path>
        </svg>
      </button>
    </div>
  </div>

  <!-- Navigation Tabs -->
  <NavigationTabs bind:activeTab ontabchange={(t) => (activeTab = t)} />
</header>
