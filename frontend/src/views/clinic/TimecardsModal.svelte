<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { m } from "../../paraglide/messages.js";
  import type { Timecard } from "@bindings/domain/models.js";
  import { TimecardService } from "@bindings/services/index.js";
  import Modal from "../../components/ui/Modal.svelte";
  import ConfirmModal from "../../components/ui/ConfirmModal.svelte";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import { getLocalDateString } from "$lib/date.js";
  import { handleError } from "$lib/error.js";

  let {
    showModal = $bindable(false),
    providerId = "",
    providerName = "",
    onrefresh,
  } = $props<{
    showModal: boolean;
    providerId: string;
    providerName: string;
    onrefresh: () => Promise<void>;
  }>();

  let timecards = $state<Timecard[]>([]);
  let loading = $state(false);
  let errorMsg = $state("");

  function get30DaysAgo(): Date {
    const d = new Date();
    d.setDate(d.getDate() - 30);
    return d;
  }

  // Filter
  let filterEndDate = $state(getLocalDateString());
  let filterStartDate = $state(getLocalDateString(get30DaysAgo()));

  // Manual Entry Form
  let showManualEntry = $state(false);
  let manualDate = $state(getLocalDateString());
  let manualHours = $state(0.0);

  // Edit Form
  let editingId = $state<string | null>(null);
  let editHours = $state(0.0);
  let requestGen = 0;

  $effect(() => {
    if (showModal && providerId) {
      untrack(() => {
        loadTimecards();
      });
    }
  });

  async function loadTimecards() {
    const gen = ++requestGen;
    loading = true;
    errorMsg = "";
    try {
      let start = "";
      if (filterStartDate) {
        start = new Date(filterStartDate + "T00:00:00").toISOString();
      }
      let end = "";
      if (filterEndDate) {
        end = new Date(filterEndDate + "T23:59:59.999").toISOString();
      }
      const results = await TimecardService.ListTimecards(providerId, start, end);
      if (gen === requestGen) {
        timecards = results ? (results.filter(Boolean) as Timecard[]) : [];
      }
    } catch (e: any) {
      if (gen === requestGen) {
        errorMsg = handleError(e, m.timecard_err_load());
      }
    } finally {
      if (gen === requestGen) {
        loading = false;
      }
    }
  }

  async function handleAddManualEntry() {
    try {
      // Create date with time at local noon to avoid timezone shift to prev/next day
      const d = new Date(manualDate + "T12:00:00").toISOString();
      await TimecardService.CreateManualTimecard(providerId, Math.round(manualHours * 60), d);
      showManualEntry = false;
      await loadTimecards();
      await onrefresh();
    } catch (e: any) {
      errorMsg = handleError(e, m.timecard_err_add());
    }
  }

  function startEdit(t: Timecard) {
    editingId = t.id;
    editHours = Number((t.total_minutes / 60).toFixed(2));
  }

  async function handleSaveEdit(t: Timecard) {
    try {
      await TimecardService.EditTimecardHours(t.id, providerId, Math.round(editHours * 60));
      editingId = null;
      await loadTimecards();
      await onrefresh();
    } catch (e: any) {
      errorMsg = handleError(e, m.timecard_err_save());
    }
  }

  let showConfirmDelete = $state(false);
  let timecardToDelete = $state<string | null>(null);

  function promptDelete(id: string) {
    timecardToDelete = id;
    showConfirmDelete = true;
  }

  async function executeDelete() {
    if (!timecardToDelete) return;
    try {
      await TimecardService.DeleteTimecard(timecardToDelete);
      await loadTimecards();
      await onrefresh();
    } catch (e: any) {
      errorMsg = handleError(e, m.timecard_err_delete());
    } finally {
      timecardToDelete = null;
    }
  }
</script>

<Modal
  bind:showModal
  title={m.timecard_modal_title({ providerName })}
  subtitle={m.timecard_modal_subtitle()}
  icon="⏱️"
  maxWidth="max-w-3xl"
>
  <div class="space-y-4 max-h-[60vh] overflow-y-auto pr-2 custom-scrollbar">
    {#if errorMsg}
      <div class="bg-rose-500/10 text-rose-400 border border-rose-500/20 p-3 rounded-lg text-sm">
        {errorMsg}
      </div>
    {/if}

    <div class="flex items-center justify-between border-b border-slate-800 pb-3">
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2">
          <label for="filter-start" class="text-xs text-slate-400">{m.timecard_from()}</label>
          <input
            id="filter-start"
            type="date"
            bind:value={filterStartDate}
            onchange={loadTimecards}
            class="rounded bg-slate-900 border border-slate-700 px-2 py-1 text-xs text-slate-200 focus:border-sky-500 focus:outline-none"
          />
        </div>
        <div class="flex items-center gap-2">
          <label for="filter-end" class="text-xs text-slate-400">{m.timecard_to()}</label>
          <input
            id="filter-end"
            type="date"
            bind:value={filterEndDate}
            onchange={loadTimecards}
            class="rounded bg-slate-900 border border-slate-700 px-2 py-1 text-xs text-slate-200 focus:border-sky-500 focus:outline-none"
          />
        </div>
      </div>
      <button
        type="button"
        onclick={() => (showManualEntry = !showManualEntry)}
        class="text-xs font-semibold px-3 py-1.5 bg-sky-500/10 text-sky-400 hover:bg-sky-500/20 rounded border border-sky-500/30"
      >
        {showManualEntry ? m.timecard_cancel() : m.timecard_add_manual()}
      </button>
    </div>

    {#if showManualEntry}
      <div class="bg-slate-800/50 p-4 rounded-xl border border-slate-700 space-y-3">
        <h4 class="text-sm font-semibold text-slate-200">{m.timecard_add_manual()}</h4>
        <div class="flex items-center gap-3">
          <div>
            <label for="manual-date" class="block text-xs text-slate-400 mb-1"
              >{m.timecard_label_date()}</label
            >
            <input
              id="manual-date"
              type="date"
              bind:value={manualDate}
              class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-sm text-slate-100"
            />
          </div>
          <div>
            <label for="manual-hours" class="block text-xs text-slate-400 mb-1"
              >{m.timecard_label_hours()}</label
            >
            <input
              id="manual-hours"
              type="number"
              min="0"
              step="0.01"
              bind:value={manualHours}
              class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-sm text-slate-100"
            />
          </div>
          <div class="flex-1"></div>
          <button
            type="button"
            onclick={handleAddManualEntry}
            class="mt-5 btn btn-primary text-xs px-4 py-1.5"
          >
            {m.timecard_save_entry()}
          </button>
        </div>
      </div>
    {/if}

    {#if loading}
      <div class="text-center py-8 text-slate-400 text-sm">{m.timecard_state_loading()}</div>
    {:else if timecards.length === 0}
      <div class="text-center py-8 text-slate-500 text-sm">{m.timecard_state_empty()}</div>
    {:else}
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-slate-800 text-slate-400">
            <th class="py-2 font-medium">{m.timecard_th_date()}</th>
            <th class="py-2 font-medium">{m.timecard_th_type()}</th>
            <th class="py-2 font-medium">{m.timecard_th_hours()}</th>
            <th class="py-2 font-medium">{m.timecard_th_pay()}</th>
            <th class="py-2 font-medium">{m.timecard_th_status()}</th>
            <th class="py-2 font-medium text-right">{m.timecard_th_actions()}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          {#each timecards as t}
            <tr class="group hover:bg-slate-800/30 transition-colors">
              <td class="py-2.5">
                {new Date(t.clock_in).toLocaleDateString()}
              </td>
              <td class="py-2.5">
                {#if t.is_manual}
                  <StatusBadge variant="manual" label={m.timecard_badge_manual()} size="sm" />
                {:else}
                  <StatusBadge variant="punched" label={m.timecard_badge_punched()} size="sm" />
                {/if}
              </td>
              <td class="py-2.5">
                {#if editingId === t.id}
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    bind:value={editHours}
                    class="w-20 rounded border border-slate-600 bg-slate-900 px-2 py-1 text-xs"
                  />
                {:else}
                  <span class="font-mono text-slate-300">{(t.total_minutes / 60).toFixed(2)}h</span>
                {/if}
              </td>
              <td class="py-2.5 font-mono text-emerald-400">
                ${(t.total_pay / 100).toFixed(2)}
              </td>
              <td class="py-2.5">
                {#if t.paid_at}
                  <StatusBadge variant="timecard_paid" label={m.timecard_badge_paid()} size="sm" />
                {:else}
                  <StatusBadge variant="unpaid" label={m.timecard_badge_unpaid()} size="sm" />
                {/if}
              </td>
              <td class="py-2.5 text-right">
                {#if editingId === t.id}
                  <button
                    type="button"
                    onclick={() => handleSaveEdit(t)}
                    class="text-xs text-emerald-400 hover:text-emerald-300 font-semibold mr-2"
                  >
                    Save
                  </button>
                  <button
                    type="button"
                    onclick={() => (editingId = null)}
                    class="text-xs text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                {:else}
                  <button
                    type="button"
                    onclick={() => promptDelete(t.id)}
                    class="text-xs text-rose-400 hover:text-rose-300 font-semibold mr-2"
                  >
                    Delete
                  </button>
                  {#if !t.paid_at && t.clock_out}
                    <button
                      type="button"
                      onclick={() => startEdit(t)}
                      class="text-xs text-sky-400 hover:text-sky-300 font-semibold"
                    >
                      Edit
                    </button>
                  {/if}
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <div class="mt-6 flex justify-end border-t border-slate-800 pt-4">
    <button
      type="button"
      onclick={() => (showModal = false)}
      class="rounded-xl bg-slate-800 px-5 py-2 text-sm font-semibold text-white hover:bg-slate-700 transition-colors"
    >
      Close
    </button>
  </div>
</Modal>

<ConfirmModal
  bind:showModal={showConfirmDelete}
  title={m.timecard_delete_title()}
  message={m.timecard_confirm_delete()}
  confirmText={m.timecard_delete_confirm()}
  onConfirm={executeDelete}
/>
