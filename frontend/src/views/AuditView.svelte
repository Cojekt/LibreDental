<script lang="ts">
  import { m } from "../paraglide/messages.js";
  import type { Patient } from "@bindings/domain/index.js";
  import AuditingSubtab from "./AuditingSubtab.svelte";
  import AnalyticsSubtab from "./AnalyticsSubtab.svelte";

  let { patients = [] } = $props<{
    patients: Patient[];
  }>();

  let activeSubtab = $state("analytics");

  const subtabs = [
    { id: "analytics", label: m.audit_tab_analytics() },
    { id: "auditing", label: m.audit_tab_auditing() }
  ];

</script>

<div class="flex h-full w-full flex-col gap-6">
  <!-- Sub-navigation -->
  <div class="flex items-center space-x-1 border-b border-slate-800/80 pb-3">
    {#each subtabs as tab}
      <button
        type="button"
        onclick={() => (activeSubtab = tab.id)}
        class={`rounded-lg px-4 py-2 text-sm font-semibold transition-all ${
          activeSubtab === tab.id
            ? "bg-slate-800 text-sky-400 shadow-sm"
            : "text-slate-400 hover:bg-slate-800/50 hover:text-slate-200"
        }`}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  <!-- Content Area -->
  <div class="flex-1 overflow-auto rounded-xl border border-slate-800/50 bg-slate-900/40 p-6 shadow-sm">
    {#if activeSubtab === "auditing"}
      <AuditingSubtab {patients} />
    {:else if activeSubtab === "analytics"}
      <AnalyticsSubtab />
    {/if}
  </div>
</div>
