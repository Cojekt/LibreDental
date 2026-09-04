<script lang="ts">
  import type {
    Patient,
    DentalChart,
    ToothCondition,
    CountryConfig,
  } from "@bindings/domain/models.js";
  import { ToothSystem } from "@bindings/domain/models.js";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

  let {
    selectedPatient,
    currentChart,
    countryMeta = null,
    currentToothSystem,
    isCreatingClaim,
    claimNoticeMsg = $bindable(""),
    getToothLabel,
    openEditCondition,
    handleDeleteCondition,
    handleGenerateClaimFromChart,
    formatCurrency,
  } = $props<{
    selectedPatient: Patient;
    currentChart: DentalChart | null;
    countryMeta?: CountryConfig | null;
    currentToothSystem: any;
    isCreatingClaim: boolean;
    claimNoticeMsg: string;
    getToothLabel: (num: number, system: any) => string;
    openEditCondition: (cond: ToothCondition) => void;
    handleDeleteCondition: (id: string) => void;
    handleGenerateClaimFromChart: () => void;
    formatCurrency: (amount: number) => string;
  }>();
</script>

<div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-xl flex flex-col gap-4">
  <div class="flex items-center justify-between">
    <div>
      <h3 class="text-base font-bold text-slate-100 m-0">{m.charting_summary_title()}</h3>
      <p class="text-xs text-slate-400 m-0">
        {m.charting_summary_sub({
          firstName: selectedPatient.first_name,
          lastName: selectedPatient.last_name,
        })}
      </p>
    </div>
    {#if currentChart && currentChart.conditions && currentChart.conditions.length > 0}
      <div class="flex items-center gap-4">
        <div class="text-xs font-semibold text-slate-300">
          {m.charting_total_planned_fee()}
          <span class="text-sky-400 font-bold">
            {formatCurrency(
              currentChart.conditions.reduce(
                (acc: number, c: ToothCondition) => acc + (c.fee || 0),
                0
              )
            )}
          </span>
        </div>
        <button
          type="button"
          onclick={handleGenerateClaimFromChart}
          disabled={isCreatingClaim}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold shadow-lg shadow-emerald-600/20 transition-all cursor-pointer disabled:opacity-50"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="w-4 h-4"
          >
            <path d="M12 5v14M5 12h14" />
          </svg>
          {isCreatingClaim ? m.billing_btn_generating_claim() : m.billing_btn_bill_charted()}
        </button>
      </div>
    {/if}
  </div>

  {#if claimNoticeMsg}
    <div
      class="p-3 rounded-xl bg-emerald-950/60 border border-emerald-500/40 text-emerald-300 text-xs font-semibold flex items-center justify-between"
    >
      <span>{claimNoticeMsg}</span>
      <button
        type="button"
        onclick={() => (claimNoticeMsg = "")}
        class="text-emerald-400 hover:text-white"
        aria-label="Dismiss">✕</button
      >
    </div>
  {/if}

  {#if !currentChart || !currentChart.conditions || currentChart.conditions.length === 0}
    <EmptyState
      title={m.charting_no_conditions_title()}
      subtitle={m.charting_no_conditions_sub()}
    />
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs border-collapse">
        <thead>
          <tr
            class="border-b border-slate-800 text-slate-400 uppercase font-semibold text-[11px] bg-slate-950/60"
          >
            <th class="py-3 px-4"
              >{m.charting_th_tooth({ code: countryMeta?.code || "Universal" })}</th
            >
            <th class="py-3 px-4">{m.charting_th_surfaces()}</th>
            <th class="py-3 px-4">{m.charting_th_code()}</th>
            <th class="py-3 px-4">{m.charting_th_desc()}</th>
            <th class="py-3 px-4">{m.charting_th_status()}</th>
            <th class="py-3 px-4 text-right">{m.charting_th_fee()}</th>
            <th class="py-3 px-4 text-center">{m.charting_th_actions()}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          {#each currentChart.conditions || [] as cond}
            <tr class="hover:bg-slate-800/40 transition-colors">
              <td class="py-3 px-4 font-bold text-sky-400">
                Tooth {getToothLabel(cond.tooth_number, currentToothSystem)}
                <span class="text-[10px] text-slate-500 font-normal"
                  >{m.charting_th_internal_no()}{cond.tooth_number})</span
                >
              </td>
              <td class="py-3 px-4 font-mono font-medium text-slate-300">
                {cond.surfaces && cond.surfaces.length > 0
                  ? cond.surfaces.join(", ")
                  : m.charting_surface_whole()}
              </td>
              <td class="py-3 px-4 font-mono text-slate-300">
                {cond.ada_code || "—"}
              </td>
              <td class="py-3 px-4 text-slate-200 font-medium">
                {cond.description}
              </td>
              <td class="py-3 px-4">
                <StatusBadge variant={cond.status} />
              </td>
              <td class="py-3 px-4 text-right font-semibold text-slate-200">
                {formatCurrency(cond.fee || 0)}
              </td>
              <td class="py-3 px-4 text-center">
                <div class="flex items-center justify-center gap-2">
                  <button
                    type="button"
                    onclick={() => openEditCondition(cond)}
                    class="p-1.5 rounded-lg text-slate-400 hover:text-sky-400 hover:bg-slate-800 transition-colors"
                    title={m.patients_btn_edit()}
                  >
                    <svg
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      class="w-4 h-4"
                    >
                      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    onclick={() => handleDeleteCondition(cond.id)}
                    class="p-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-slate-800 transition-colors"
                    title={m.patient_archive()}
                  >
                    <svg
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      class="w-4 h-4"
                    >
                      <polyline points="3 6 5 6 21 6" />
                      <path
                        d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                      />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
