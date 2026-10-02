<script lang="ts">
  import { auth } from "../stores/auth.svelte.js";
  import { m } from "../paraglide/messages.js";
  import { BillingService } from "@bindings/services/index.js";
  import type { Payment, CountryConfig } from "@bindings/domain/index.js";
  import { getTodayDateString, getLocalDateString } from "$lib/date.js";
  import { formatCurrency } from "$lib/currency.js";

  let { countryMeta = null } = $props<{
    countryMeta?: CountryConfig | null;
  }>();

  let payments = $state<Payment[]>([]);
  let loading = $state(false);

  // Default to last 30 days
  const today = new Date();
  const thirtyDaysAgo = new Date();
  thirtyDaysAgo.setDate(today.getDate() - 30);

  let startDate = $state(getLocalDateString(thirtyDaysAgo));
  let endDate = $state(getTodayDateString());

  let totalRevenue = $derived(payments.reduce((sum, p) => sum + (p.amount || 0), 0));

  // Discards responses from superseded requests when the date range changes quickly.
  let requestGen = 0;

  async function fetchRevenue() {
    const gen = ++requestGen;
    loading = true;
    try {
      const res = await BillingService.GetRevenueStats(auth.token, startDate, endDate);
      if (gen !== requestGen) return;
      payments = (res?.filter(Boolean) as Payment[]) || [];
    } catch (e) {
      if (gen !== requestGen) return;
      console.error("Failed to fetch revenue stats", e);
      payments = [];
    } finally {
      if (gen === requestGen) loading = false;
    }
  }

  // Also covers the initial load.
  $effect(() => {
    if (startDate && endDate) {
      fetchRevenue();
    }
  });
</script>

<div class="flex h-full flex-col">
  <!-- Toolbar -->
  <div
    class="mb-6 flex items-center justify-between rounded-xl bg-slate-800/80 p-4 shadow-sm border border-slate-700"
  >
    <div>
      <h2 class="text-xl font-bold text-slate-100">{m.revenue_tracker()}</h2>
      <p class="text-sm text-slate-400">{m.revenue_subtitle()}</p>
    </div>

    <div class="flex items-center gap-4">
      <div class="flex flex-col">
        <label for="start-date" class="text-xs font-medium text-slate-400 mb-1"
          >{m.revenue_start_date()}</label
        >
        <input
          id="start-date"
          type="date"
          bind:value={startDate}
          class="input input-sm bg-slate-900 border-slate-700 text-slate-200"
        />
      </div>
      <div class="flex flex-col">
        <label for="end-date" class="text-xs font-medium text-slate-400 mb-1"
          >{m.revenue_end_date()}</label
        >
        <input
          id="end-date"
          type="date"
          bind:value={endDate}
          class="input input-sm bg-slate-900 border-slate-700 text-slate-200"
        />
      </div>
      <div class="flex flex-col justify-end h-full pt-4">
        <button type="button" class="btn btn-primary btn-sm" onclick={fetchRevenue}
          >{m.revenue_refresh()}</button
        >
      </div>
    </div>
  </div>

  <!-- Stats Cards -->
  <div class="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
    <div
      class="flex flex-col justify-center rounded-xl border border-sky-900/50 bg-sky-950/20 p-6 shadow-sm"
    >
      <span class="text-sm font-medium text-sky-400">{m.revenue_total()}</span>
      <span class="mt-2 text-3xl font-bold text-slate-100">
        {formatCurrency(totalRevenue, countryMeta?.default_currency)}
      </span>
    </div>
    <div
      class="flex flex-col justify-center rounded-xl border border-emerald-900/50 bg-emerald-950/20 p-6 shadow-sm"
    >
      <span class="text-sm font-medium text-emerald-400">{m.revenue_transactions()}</span>
      <span class="mt-2 text-3xl font-bold text-slate-100">
        {payments.length}
      </span>
    </div>
  </div>

  <!-- Transactions Table -->
  <div class="flex-1 overflow-auto rounded-lg border border-slate-800 bg-slate-900/50">
    <table class="w-full text-left text-sm text-slate-300">
      <thead class="sticky top-0 bg-slate-800/90 uppercase text-slate-400 backdrop-blur">
        <tr>
          <th scope="col" class="px-4 py-3 font-medium">{m.revenue_th_date()}</th>
          <th scope="col" class="px-4 py-3 font-medium">{m.revenue_th_method()}</th>
          <th scope="col" class="px-4 py-3 font-medium text-right">{m.revenue_th_amount()}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-800">
        {#if loading}
          <tr>
            <td colspan="3" class="p-8 text-center text-slate-500">{m.revenue_loading()}</td>
          </tr>
        {:else if payments.length === 0}
          <tr>
            <td colspan="3" class="p-8 text-center text-slate-500">{m.revenue_empty()}</td>
          </tr>
        {:else}
          {#each payments as payment}
            <tr class="hover:bg-slate-800/40">
              <td class="whitespace-nowrap px-4 py-3">{payment.date}</td>
              <td class="px-4 py-3 capitalize">{payment.method}</td>
              <td class="px-4 py-3 text-right font-medium text-slate-200"
                >{formatCurrency(payment.amount, countryMeta?.default_currency)}</td
              >
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>
