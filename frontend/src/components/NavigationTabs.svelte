<script lang="ts">
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../lib/locale.svelte.js";

  let { activeTab = $bindable("clinic"), ontabchange } = $props<{
    activeTab?: string;
    ontabchange: (tab: string) => void;
  }>();

  // Tabs are derived so labels re-evaluate when localeVersion changes (locale switch).
  const tabs = $derived([
    {
      id: "clinic",
      label: (getLocaleVersion(), m.nav_clinic()),
      enabled: true,
    },
    {
      id: "patients",
      label: (getLocaleVersion(), m.nav_patients()),
      enabled: true,
    },
    {
      id: "appointments",
      label: (getLocaleVersion(), m.nav_appointments()),
      enabled: true,
    },
    {
      id: "charting",
      label: (getLocaleVersion(), m.nav_charting()),
      enabled: true,
    },
    {
      id: "billing",
      label: (getLocaleVersion(), m.nav_billing()),
      enabled: true,
    },
  ]);
</script>

<nav class="w-full border-t border-slate-800/80 select-none">
  <div class="flex w-full items-center space-x-1 px-6 sm:px-8 pt-1.5">
    {#each tabs as tab}
      {#if tab.enabled}
        <button
          type="button"
          onclick={() => ontabchange(tab.id)}
          class={`group relative flex items-center gap-2 px-4 py-2.5 text-sm font-semibold transition-all duration-150 rounded-t-lg outline-none ${
            activeTab === tab.id
              ? "text-sky-400 bg-slate-950/60 shadow-sm"
              : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"
          }`}
        >
          <span>{tab.label}</span>
          {#if activeTab === tab.id}
            <div class="absolute bottom-0 left-0 right-0 h-[2px] bg-sky-400"></div>
          {/if}
        </button>
      {:else}
        <div
          class="flex items-center gap-2 px-3 py-2.5 text-sm font-medium text-slate-500 cursor-not-allowed opacity-50 hover:opacity-75 transition-opacity"
          title={m.nav_billing_coming_soon()}
        >
          <span>{tab.label}</span>
        </div>
      {/if}
    {/each}
  </div>
</nav>
