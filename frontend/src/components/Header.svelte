<script lang="ts">
  import NavigationTabs from "./NavigationTabs.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";

  let {
    activeTab = $bindable("clinic"),
    onnewpatient,
    onnewappointment,
    onopensettings,
    onopenstafflogin,
  } = $props<{
    activeTab: string;
    onnewpatient: () => void;
    onnewappointment: () => void;
    onopensettings: () => void;
    onopenstafflogin: () => void;
  }>();

  import { auth } from "../stores/auth.svelte.js";
</script>

<header class="w-full border-b border-slate-800 bg-slate-900 shadow-sm">
  <!-- Top brand bar -->
  <div class="flex h-16 w-full items-center justify-between px-6 sm:px-8">
    <div class="flex items-center gap-3">
      <img
        src="/sourceicon.svg"
        alt="LibreDental Logo"
        class="h-9 w-9 rounded-xl shadow-md shadow-purple-500/20 object-contain"
      />
      <div class="flex items-center">
        <h1 class="m-0 text-xl font-bold tracking-tight text-slate-50">LibreDental</h1>
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
        onclick={onopenstafflogin}
        class="flex items-center justify-center rounded-xl border px-4 py-2 transition-all shadow-sm cursor-pointer min-w-[140px] font-bold text-sm {auth.currentStaffId
          ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20'
          : 'border-rose-500/30 bg-rose-500/10 text-rose-400 hover:bg-rose-500/20'}"
      >
        {#if auth.currentStaffId}
          {auth.currentStaff?.name || m.common_loading()}
        {:else}
          {m.header_staff_not_logged_in()}
        {/if}
      </button>

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
