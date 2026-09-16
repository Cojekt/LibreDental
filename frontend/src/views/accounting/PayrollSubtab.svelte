<script lang="ts">
  import type { Provider } from "@bindings/domain/models.js";
  import { TimecardService } from "@bindings/services/index.js";
  import { untrack } from "svelte";
  import { auth } from "../../stores/auth.svelte.js";
  import ConfirmModal from "../../components/ui/ConfirmModal.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";
  import TimecardsModal from "./TimecardsModal.svelte";

  let { providers = [] } = $props<{
    providers: Provider[];
  }>();

  let totalOwed = $state<Record<string, number | undefined>>({});
  let providerGen: Record<string, number> = {};

  let showTimecardsModal = $state(false);
  let selectedProviderId = $state("");
  let selectedProviderName = $state("");

  $effect(() => {
    const currentProviders = providers;
    untrack(() => {
      loadTotalOwed(currentProviders);
    });
  });

  async function loadTotalOwed(provs: Provider[] = providers) {
    for (const p of provs) {
      const gen = (providerGen[p.id] = (providerGen[p.id] || 0) + 1);
      try {
        const owed = await TimecardService.GetTotalOwed(p.id);
        if (providerGen[p.id] === gen) {
          totalOwed[p.id] = owed;
        }
      } catch (e) {
        if (providerGen[p.id] === gen) {
          totalOwed[p.id] = undefined;
        }
      }
    }
  }

  let showConfirmPay = $state(false);
  let providerToPay = $state<string | null>(null);

  function promptPay(id: string) {
    providerToPay = id;
    showConfirmPay = true;
  }

  async function executePay() {
    if (!providerToPay) return;
    try {
      await TimecardService.PaySalary(auth.token, providerToPay);
      await loadTotalOwed();
    } catch (e) {
      console.error("Pay Salary failed", e);
      throw e;
    } finally {
      providerToPay = null;
    }
  }
</script>

<div class="space-y-4">
  {#if providers.length === 0}
    <EmptyState title={m.prov_empty_title()} subtitle={m.prov_empty_sub()} />
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each providers as p}
        <div
          class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 relative group hover:border-slate-700 transition-colors"
        >
          <div class="flex items-center gap-3">
            <div
              class="h-10 w-10 rounded-full flex items-center justify-center text-white font-bold text-sm shadow-md"
              style="background-color: {p.color || '#3b82f6'};"
            >
              {p.name.charAt(0)}
            </div>
            <div>
              <h4 class="text-sm font-bold text-slate-100">{p.name}</h4>
              <p class="text-xs text-sky-400 capitalize font-medium">{p.role}</p>
            </div>
          </div>

          {#if p.hourly_rate}
            <div class="text-xs font-medium text-emerald-400">
              {m.prov_wage_prefix()} ${(p.hourly_rate / 100).toFixed(2)}{m.prov_wage_suffix()}
            </div>
          {/if}

          <div class="bg-slate-800/40 rounded-lg p-3 mt-2 border border-slate-700/50">
            <div class="flex items-center justify-between">
              <div class="text-slate-300 text-xs font-semibold">
                {m.prov_total_owed()}
                {#if totalOwed[p.id] === undefined}
                  <span class="text-slate-500 text-sm ml-1">...</span>
                {:else}
                  <span class="text-emerald-400 text-sm ml-1"
                    >${((totalOwed[p.id] || 0) / 100).toFixed(2)}</span
                  >
                {/if}
              </div>
              <button
                type="button"
                onclick={() => promptPay(p.id)}
                class="bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 px-3 py-1 rounded text-xs font-bold transition-colors border border-emerald-500/30"
              >
                {m.prov_pay_salary()}
              </button>
            </div>
            <div class="mt-3 flex justify-end">
              <button
                type="button"
                onclick={() => {
                  selectedProviderId = p.id;
                  selectedProviderName = p.name;
                  showTimecardsModal = true;
                }}
                class="text-sky-400 hover:text-sky-300 text-xs font-semibold flex items-center gap-1"
              >
                {m.prov_view_timecards()}
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<TimecardsModal
  bind:showModal={showTimecardsModal}
  providerId={selectedProviderId}
  providerName={selectedProviderName}
  onrefresh={loadTotalOwed}
/>

<ConfirmModal
  bind:showModal={showConfirmPay}
  title={m.prov_pay_salary()}
  message={m.prov_confirm_pay_salary()}
  confirmText={m.billing_btn_record_payment()}
  onConfirm={executePay}
/>
