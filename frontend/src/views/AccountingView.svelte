<script lang="ts">
  import type { CountryConfig, Provider } from "@bindings/domain/index.js";
  import TabNav from "../components/ui/TabNav.svelte";
  import PayrollSection from "./accounting/PayrollSection.svelte";
  import AnalyticsSection from "./accounting/AnalyticsSection.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../stores/locale.svelte.js";

  let {
    providers = [],
    countryMeta = null,
  }: {
    providers: Provider[];
    countryMeta?: CountryConfig | null;
  } = $props();

  let activeSubtab = $state("payroll");

  const subtabs = $derived([
    { id: "payroll", label: (getLocaleVersion(), m.acct_tab_payroll()) },
    { id: "analysis", label: (getLocaleVersion(), m.acct_tab_analysis()) },
  ]);
</script>

<div class="flex h-full w-full flex-col gap-6">
  <div class="border-b border-slate-800/80 pb-3">
    <TabNav tabs={subtabs} bind:activeTab={activeSubtab} />
  </div>

  <div
    class="flex-1 overflow-auto rounded-xl border border-slate-800/50 bg-slate-900/40 p-6 shadow-sm"
  >
    {#if activeSubtab === "payroll"}
      <PayrollSection {providers} />
    {:else if activeSubtab === "analysis"}
      <AnalyticsSection {countryMeta} />
    {/if}
  </div>
</div>
