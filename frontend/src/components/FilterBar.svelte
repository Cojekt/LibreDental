<script lang="ts">
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../lib/locale.svelte.js";

  let {
    searchQuery = $bindable(),
    statusFilter = $bindable(),
    onloadpatients,
  } = $props<{
    searchQuery: string;
    statusFilter: string;
    onloadpatients: () => void;
  }>();

  const searchPlaceholder = $derived.by(() => {
    getLocaleVersion();
    return m.patients_search_placeholder();
  });
</script>

<div class="mb-4 flex items-center gap-3">
  <div class="relative w-full max-w-[480px] flex-1">
    <svg
      class="absolute left-3.5 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-slate-400 pointer-events-none z-10"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
    >
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
    <input
      type="text"
      placeholder={searchPlaceholder}
      class="box-border w-full rounded-xl border border-slate-700 bg-slate-900 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none shadow-sm transition-all"
      style="padding-left: 2.75rem; padding-right: 0.75rem;"
      bind:value={searchQuery}
      oninput={onloadpatients}
    />
  </div>
  <div
    class="flex items-center gap-1 rounded-xl border border-slate-800 bg-slate-900/90 p-1 shadow-sm select-none"
  >
    <button
      type="button"
      onclick={() => {
        statusFilter = "active";
        onloadpatients();
      }}
      class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
        statusFilter === "active"
          ? "bg-sky-500/20 text-sky-400 border border-sky-500/30 shadow-sm"
          : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"
      }`}
    >
      {m.patients_filter_active()}
    </button>
    <button
      type="button"
      onclick={() => {
        statusFilter = "archived";
        onloadpatients();
      }}
      class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
        statusFilter === "archived"
          ? "bg-amber-500/20 text-amber-400 border border-amber-500/30 shadow-sm"
          : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"
      }`}
    >
      {m.patients_filter_archived()}
    </button>
  </div>
</div>
