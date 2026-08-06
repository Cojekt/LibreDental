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
      icon: `<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21h18"/><path d="M5 21V7l7-4 7 4v14"/><path d="M9 18h6v-4H9v4z"/><path d="M9 10h2v2H9v-2z"/><path d="M13 10h2v2h-2v-2z"/></svg>`,
      enabled: true,
    },
    {
      id: "patients",
      label: (getLocaleVersion(), m.nav_patients()),
      icon: `<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
      enabled: true,
    },
    {
      id: "appointments",
      label: (getLocaleVersion(), m.nav_appointments()),
      icon: `<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>`,
      enabled: true,
    },
    {
      id: "charting",
      label: (getLocaleVersion(), m.nav_charting()),
      icon: `<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2C8 2 5 5 5 9c0 5.25 3 9.25 7 13 4-3.75 7-7.75 7-13 0-4-3-7-7-7z"/><circle cx="12" cy="9" r="2.5"/></svg>`,
      enabled: true,
    },
    {
      id: "billing",
      label: (getLocaleVersion(), m.nav_billing()),
      icon: `<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="5" width="20" height="14" rx="2"/><line x1="2" y1="10" x2="22" y2="10"/></svg>`,
      enabled: false,
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
          <span class={`transition-colors ${activeTab === tab.id ? "text-sky-400" : "text-slate-400 group-hover:text-slate-300"}`}>
            {@html tab.icon}
          </span>
          <span>{tab.label}</span>
          {#if activeTab === tab.id}
            <div class="absolute bottom-0 left-0 right-0 h-[2px] bg-sky-400"></div>
          {/if}
        </button>
      {:else}
        <div
          class="flex items-center gap-2 px-3 py-2.5 text-sm font-medium text-slate-500 cursor-not-allowed opacity-50 hover:opacity-75 transition-opacity"
          title="Coming soon"
        >
          <span>{@html tab.icon}</span>
          <span>{tab.label}</span>
        </div>
      {/if}
    {/each}
  </div>
</nav>
