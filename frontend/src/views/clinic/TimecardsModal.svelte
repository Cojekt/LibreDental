<script lang="ts">
  import { onMount } from "svelte";
  import type { Timecard } from "@bindings/domain/models.js";
  import { TimecardService } from "@bindings/services/index.js";
  import Modal from "../../components/ui/Modal.svelte";

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

  function getLocalDateString(d: Date = new Date()): string {
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
  }

  // Filter
  let filterEndDate = $state(getLocalDateString());
  let filterStartDate = $state(
    getLocalDateString(new Date(Date.now() - 30 * 24 * 60 * 60 * 1000))
  );

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
      loadTimecards();
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
        errorMsg = e.message || "Failed to load timecards";
      }
    } finally {
      if (gen === requestGen) {
        loading = false;
      }
    }
  }

  async function handleAddManualEntry() {
    try {
      // Create date with time at noon to avoid timezone shift to prev day
      const d = new Date(manualDate + "T12:00:00Z").toISOString();
      await TimecardService.CreateManualTimecard(providerId, Math.round(manualHours * 60), d);
      showManualEntry = false;
      await loadTimecards();
      await onrefresh();
    } catch (e: any) {
      errorMsg = e.message || "Failed to add manual entry";
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
      errorMsg = e.message || "Failed to save edit";
    }
  }

  async function handleDelete(id: string) {
    if (
      confirm(
        "Warning: Deleting a timecard record is permanent and will alter payroll calculations. Proceed?"
      )
    ) {
      try {
        await TimecardService.DeleteTimecard(id);
        await loadTimecards();
        await onrefresh();
      } catch (e: any) {
        errorMsg = e.message || "Failed to delete timecard";
      }
    }
  }
</script>

<Modal
  bind:showModal
  title={`Timecards: ${providerName}`}
  subtitle="View and edit recorded hours"
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
          <label for="filter-start" class="text-xs text-slate-400">From:</label>
          <input
            id="filter-start"
            type="date"
            bind:value={filterStartDate}
            onchange={loadTimecards}
            class="rounded bg-slate-900 border border-slate-700 px-2 py-1 text-xs text-slate-200 focus:border-sky-500 focus:outline-none"
          />
        </div>
        <div class="flex items-center gap-2">
          <label for="filter-end" class="text-xs text-slate-400">To:</label>
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
        {showManualEntry ? "Cancel" : "+ Add Manual Entry"}
      </button>
    </div>

    {#if showManualEntry}
      <div class="bg-slate-800/50 p-4 rounded-xl border border-slate-700 space-y-3">
        <h4 class="text-sm font-semibold text-slate-200">Add Manual Entry</h4>
        <div class="flex items-center gap-3">
          <div>
            <label for="manual-date" class="block text-xs text-slate-400 mb-1">Date</label>
            <input
              id="manual-date"
              type="date"
              bind:value={manualDate}
              class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-sm text-slate-100"
            />
          </div>
          <div>
            <label for="manual-hours" class="block text-xs text-slate-400 mb-1">Hours</label>
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
            Save Entry
          </button>
        </div>
      </div>
    {/if}

    {#if loading}
      <div class="text-center py-8 text-slate-400 text-sm">Loading timecards...</div>
    {:else if timecards.length === 0}
      <div class="text-center py-8 text-slate-500 text-sm">No timecards recorded yet.</div>
    {:else}
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-slate-800 text-slate-400">
            <th class="py-2 font-medium">Date</th>
            <th class="py-2 font-medium">Type</th>
            <th class="py-2 font-medium">Hours</th>
            <th class="py-2 font-medium">Pay</th>
            <th class="py-2 font-medium">Status</th>
            <th class="py-2 font-medium text-right">Actions</th>
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
                  <span
                    class="text-xs text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20"
                    >Manual</span
                  >
                {:else}
                  <span class="text-xs text-slate-400 bg-slate-800 px-2 py-0.5 rounded"
                    >Punched</span
                  >
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
                  <span class="text-xs text-emerald-400">Paid</span>
                {:else}
                  <span class="text-xs text-rose-400">Unpaid</span>
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
                    onclick={() => handleDelete(t.id)}
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
