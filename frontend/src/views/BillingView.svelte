<script lang="ts">
  import type { Patient, Provider, CountryConfig } from "@bindings/domain/index.js";
  import ViewHeader from "../components/ui/ViewHeader.svelte";
  import TabNav from "../components/ui/TabNav.svelte";
  import ClaimsSection from "./billing/ClaimsSection.svelte";
  import PaymentsSection from "./billing/PaymentsSection.svelte";
  import BundlesSection from "./billing/BundlesSection.svelte";
  import FeeSchedulesSection from "./billing/FeeSchedulesSection.svelte";
  import { m } from "../paraglide/messages.js";

  let {
    patients = [],
    providers = [],
    countryMeta = null,
  } = $props<{
    patients: Patient[];
    providers: Provider[];
    countryMeta?: CountryConfig | null;
  }>();

  let billingTab = $state<"claims" | "payments" | "bundles" | "fees">("claims");

  let tabs = $derived([
    {
      id: "claims",
      label: m.billing_tab_claims(),
      icon: "M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414A1 1 0 0 1 19 9.414V19a2 2 0 0 1-2 2z",
    },
    {
      id: "payments",
      label: m.billing_tab_payments(),
      icon: "M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 0 0 3-3V8a3 3 0 0 0-3-3H6a3 3 0 0 0-3 3v8a3 3 0 0 0 3 3z",
    },
    {
      id: "bundles",
      label: m.billing_tab_bundles(),
      icon: "M19 11H5m14 0a2 2 0 0 1 2 2v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6a2 2 0 0 1 2-2m14 0V9a2 2 0 0 1-2-2M5 11V9a2 2 0 0 1 2-2m0 0V5a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2M7 7h10",
    },
    {
      id: "fees",
      label: m.billing_tab_fees(),
      icon: "M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z",
    },
  ]);
</script>

<div class="space-y-6">
  <ViewHeader title={m.billing_title()} subtitle={m.billing_subtitle()} icon="💳" />

  <TabNav {tabs} bind:activeTab={billingTab} />

  <div>
    {#if billingTab === "claims"}
      <ClaimsSection {patients} {providers} {countryMeta} />
    {:else if billingTab === "payments"}
      <PaymentsSection {patients} {countryMeta} />
    {:else if billingTab === "bundles"}
      <BundlesSection {countryMeta} />
    {:else if billingTab === "fees"}
      <FeeSchedulesSection {providers} {countryMeta} />
    {/if}
  </div>
</div>
