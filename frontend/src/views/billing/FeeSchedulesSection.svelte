<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService } from "@bindings/services/index.js";
  import type {
    Provider,
    CountryConfig,
    ProcedureCode,
    FeeSchedule,
  } from "@bindings/domain/index.js";
  import Modal from "../../components/ui/Modal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

  let { providers = [], countryMeta = null } = $props<{
    providers: Provider[];
    countryMeta?: CountryConfig | null;
  }>();

  let procedureCodes = $state<ProcedureCode[]>([]);
  let feeSchedules = $state<FeeSchedule[]>([]);
  let loadingFees = $state(false);
  let feeFilterProvider = $state("");

  // Fee schedule modal state
  let showFeeModal = $state(false);
  let editingFeeCode = $state("");
  let editingFeeCustom = $state<number>(0);
  let editingFeeProviderId = $state("");

  function fmt(n: number) {
    const curr = countryMeta?.default_currency || "USD";
    try {
      return new Intl.NumberFormat("en-US", { style: "currency", currency: curr }).format(n);
    } catch {
      return `${n.toFixed(2)}`;
    }
  }

  export async function loadProcedureCodes() {
    const cc = countryMeta?.code || "US";
    try {
      const res = await BillingService.ListProcedureCodes(cc, feeFilterProvider);
      procedureCodes = (res?.filter(Boolean) as ProcedureCode[]) || [];
    } catch (e) {
      console.error("Failed to load procedure codes:", e);
    }
  }

  export async function loadFeeSchedules() {
    loadingFees = true;
    const cc = countryMeta?.code || "US";
    try {
      const res = await BillingService.ListFeeSchedules(cc, feeFilterProvider);
      feeSchedules = (res?.filter(Boolean) as FeeSchedule[]) || [];
    } catch (e) {
      console.error("Failed to load fee schedules:", e);
    } finally {
      loadingFees = false;
    }
  }

  function openEditFeeModal(code: string, currentFee: number) {
    editingFeeCode = code;
    editingFeeCustom = currentFee;
    editingFeeProviderId = feeFilterProvider;
    showFeeModal = true;
  }

  async function saveFeeSchedule(e: Event) {
    e.preventDefault();
    if (!editingFeeCode || editingFeeCustom < 0) return;
    const cc = countryMeta?.code || "US";
    try {
      await BillingService.SaveFeeSchedule({
        id: `fee_${Date.now()}`,
        country_code: cc as any,
        code: editingFeeCode,
        provider_id: editingFeeProviderId,
        custom_fee: Number(editingFeeCustom),
        updated_at: new Date().toISOString(),
      });
      showFeeModal = false;
      await loadProcedureCodes();
      await loadFeeSchedules();
    } catch (err) {
      console.error("Failed to save fee schedule:", err);
    }
  }

  onMount(async () => {
    await loadProcedureCodes();
    await loadFeeSchedules();
  });
</script>

<div class="space-y-4">
  <div
    class="flex flex-wrap justify-between items-center bg-slate-900 border border-slate-800 rounded-xl p-4 gap-4"
  >
    <div class="flex items-center gap-3">
      <span class="text-xs font-semibold text-slate-300">{m.billing_filter_provider_label()}</span>
      <select
        bind:value={feeFilterProvider}
        onchange={() => {
          loadProcedureCodes();
          loadFeeSchedules();
        }}
        class="w-full max-w-xs rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
      >
        <option value="">{m.billing_filter_all_providers()}</option>
        {#each providers as pr}
          <option value={pr.id}>{pr.name}</option>
        {/each}
      </select>
    </div>
    <div class="text-xs text-sky-400 font-semibold">
      📍 {countryMeta?.name || "Global"}
      {m.billing_catalog_banner()}
    </div>
  </div>

  {#if loadingFees}
    <div class="p-8 text-center text-sm text-slate-400">{m.common_loading()}</div>
  {:else if procedureCodes.length === 0}
    <EmptyState
      title={`No procedure codes available for ${countryMeta?.name || "this country"}.`}
    />
  {:else}
    <div class="rounded-xl border border-slate-800 bg-slate-900/40 overflow-x-auto">
      <table class="w-full text-left text-sm text-slate-300">
        <thead
          class="bg-slate-900/80 border-b border-slate-800 text-xs font-semibold uppercase tracking-wider text-slate-400"
        >
          <tr>
            <th class="px-4 py-3">{m.billing_th_code()}</th>
            <th class="px-4 py-3">{m.billing_th_category()}</th>
            <th class="px-4 py-3">{m.billing_th_desc()}</th>
            <th class="px-4 py-3">{m.billing_th_base_fee()}</th>
            <th class="px-4 py-3">{m.billing_th_effective_fee()}</th>
            <th class="px-4 py-3 text-center">{m.patients_th_actions()}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          {#each procedureCodes as p (p.code)}
            {@const hasCustom = (p.effective_fee || 0) > 0 && p.effective_fee !== p.default_fee}
            <tr class="hover:bg-slate-900/50 transition-colors">
              <td class="px-4 py-3 font-mono font-bold text-sky-400">{p.code}</td>
              <td class="px-4 py-3"><StatusBadge variant="draft" label={p.category} /></td>
              <td class="px-4 py-3 text-slate-200 font-medium">{p.description}</td>
              <td class="px-4 py-3 text-slate-400 font-mono">{fmt(p.default_fee)}</td>
              <td class="px-4 py-3 font-bold text-slate-100 font-mono">
                {fmt(p.effective_fee || p.default_fee)}
                {#if hasCustom}
                  <span
                    class="ml-2 text-[10px] text-amber-400 font-semibold px-1.5 py-0.5 rounded bg-amber-950/60 border border-amber-800"
                  >
                    {m.billing_custom_tag()}
                  </span>
                {/if}
              </td>
              <td class="px-4 py-3 text-center">
                <button
                  type="button"
                  class="btn btn-secondary text-xs py-1 px-2.5 cursor-pointer"
                  onclick={() => openEditFeeModal(p.code, p.effective_fee || p.default_fee)}
                >
                  Edit Fee
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- Edit Fee Schedule Modal -->
<Modal
  bind:showModal={showFeeModal}
  title={`${m.billing_fee_edit_title()}: ${editingFeeCode}`}
  subtitle={m.billing_fee_custom_desc()}
  icon="🏷️"
  maxWidth="max-w-md"
>
  <form onsubmit={saveFeeSchedule} class="space-y-4">
    <FormField
      label={`${m.billing_fee_custom_label()} (${countryMeta?.default_currency || "USD"})`}
      forId="fee-custom-amount"
      required
    >
      <Input
        id="fee-custom-amount"
        type="number"
        step="0.01"
        min="0"
        bind:value={editingFeeCustom}
        required
      />
    </FormField>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
        onclick={() => (showFeeModal = false)}
      >
        {m.common_cancel()}
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        {m.common_save()}
      </button>
    </div>
  </form>
</Modal>
