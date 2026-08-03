<script lang="ts">
  import type { Patient, Appointment, Provider, Operatory } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";

  let {
    showModal = $bindable(false),
    isEditing = false,
    patients = [],
    configuredProviders = [],
    configuredOperatories = [],
    selectedPatientId = $bindable(""),
    providerId = $bindable(""),
    operatoryId = $bindable(""),
    startDateStr = $bindable(new Date().toISOString().split("T")[0]),
    startTimeStr = $bindable("09:00"),
    endTimeStr = $bindable("10:00"),
    status = $bindable("scheduled"),
    reason = $bindable(""),
    color = $bindable("#3b82f6"),
    notes = $bindable(""),
    onsave,
    ondelete,
  } = $props<{
    showModal: boolean;
    isEditing: boolean;
    patients: Patient[];
    configuredProviders?: Provider[];
    configuredOperatories?: Operatory[];
    selectedPatientId: string;
    providerId: string;
    operatoryId: string;
    startDateStr: string;
    startTimeStr: string;
    endTimeStr: string;
    status: string;
    reason: string;
    color: string;
    notes: string;
    onsave: (e: Event) => void;
    ondelete?: () => void;
  }>();

  const statuses = [
    { id: "scheduled", label: "Scheduled", class: "bg-blue-500/20 text-blue-400 border-blue-500/30" },
    { id: "confirmed", label: "Confirmed", class: "bg-sky-500/20 text-sky-400 border-sky-500/30" },
    { id: "arrived", label: "Arrived", class: "bg-amber-500/20 text-amber-400 border-amber-500/30" },
    { id: "in_chair", label: "In Chair", class: "bg-purple-500/20 text-purple-400 border-purple-500/30" },
    { id: "completed", label: "Completed", class: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30" },
    { id: "cancelled", label: "Cancelled", class: "bg-rose-500/20 text-rose-400 border-rose-500/30" },
    { id: "no_show", label: "No Show", class: "bg-slate-500/20 text-slate-400 border-slate-500/30" },
  ];

  const colorOptions = [
    { hex: "#3b82f6", name: "Blue" },
    { hex: "#06b6d4", name: "Cyan" },
    { hex: "#10b981", name: "Emerald" },
    { hex: "#f59e0b", name: "Amber" },
    { hex: "#a855f7", name: "Purple" },
    { hex: "#f43f5e", name: "Rose" },
  ];

  function setDuration(minutes: number) {
    if (!startTimeStr) return;
    const [h, m] = startTimeStr.split(":").map(Number);
    const date = new Date();
    date.setHours(h, m + minutes, 0, 0);
    const endH = String(date.getHours()).padStart(2, "0");
    const endM = String(date.getMinutes()).padStart(2, "0");
    endTimeStr = `${endH}:${endM}`;
  }
</script>

{#if showModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4 backdrop-blur-sm">
    <div
      class="w-full max-w-xl rounded-2xl border border-slate-700 bg-slate-900 p-6 shadow-2xl overflow-y-auto max-h-[90vh] text-slate-100"
    >
      <div class="flex items-center justify-between border-b border-slate-800 pb-4 mb-4">
        <div>
          <h2 class="text-xl font-bold tracking-tight text-white">
            {isEditing ? "Edit Appointment" : "Schedule New Appointment"}
          </h2>
          <p class="text-xs text-slate-400 mt-0.5">
            Book or modify patient chair time and procedure details
          </p>
        </div>
        <button
          type="button"
          onclick={() => (showModal = false)}
          class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white"
        >
          ✕
        </button>
      </div>

      <form onsubmit={onsave} class="space-y-4">
        <!-- Patient Picker -->
        <div>
          <label for="appt-patient" class="block text-xs font-semibold text-slate-300 mb-1">
            Patient <span class="text-rose-400">*</span>
          </label>
          <select
            id="appt-patient"
            bind:value={selectedPatientId}
            required
            class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
          >
            <option value="" disabled>-- Select Patient --</option>
            {#each patients as p}
              <option value={p.id}>
                {p.last_name}, {p.first_name} ({p.phone_primary || p.email || 'No contact info'})
              </option>
            {/each}
          </select>
          {#if patients.length === 0}
            <p class="text-xs text-amber-400 mt-1">
              ⚠️ No patients registered yet. Please add a patient first.
            </p>
          {/if}
        </div>

        <!-- Provider & Operatory -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="appt-provider" class="block text-xs font-semibold text-slate-300 mb-1">Provider</label>
            <select
              id="appt-provider"
              bind:value={providerId}
              class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
            >
              {#if configuredProviders.length === 0}
                <option value="">No Providers Configured</option>
              {:else}
                {#each configuredProviders as prov}
                  <option value={prov.id}>{prov.name} ({prov.role})</option>
                {/each}
              {/if}
            </select>
          </div>

          <div>
            <label for="appt-operatory" class="block text-xs font-semibold text-slate-300 mb-1">Operatory / Chair</label>
            <select
              id="appt-operatory"
              bind:value={operatoryId}
              class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
            >
              {#if configuredOperatories.length === 0}
                <option value="">No Operatories Configured</option>
              {:else}
                {#each configuredOperatories as op}
                  <option value={op.id}>{op.name} ({op.type})</option>
                {/each}
              {/if}
            </select>
          </div>
        </div>

        <!-- Date & Times -->
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label for="appt-date" class="block text-xs font-semibold text-slate-300 mb-1">Date</label>
            <input
              id="appt-date"
              type="date"
              bind:value={startDateStr}
              required
              class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3 py-2 text-sm text-white focus:border-sky-500 focus:outline-none"
            />
          </div>

          <div>
            <label for="appt-start-time" class="block text-xs font-semibold text-slate-300 mb-1">Start Time</label>
            <input
              id="appt-start-time"
              type="time"
              bind:value={startTimeStr}
              required
              class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3 py-2 text-sm text-white focus:border-sky-500 focus:outline-none"
            />
          </div>

          <div>
            <label for="appt-end-time" class="block text-xs font-semibold text-slate-300 mb-1">End Time</label>
            <input
              id="appt-end-time"
              type="time"
              bind:value={endTimeStr}
              required
              class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3 py-2 text-sm text-white focus:border-sky-500 focus:outline-none"
            />
          </div>
        </div>

        <!-- Quick Duration Presets -->
        <div class="flex items-center gap-2 pt-1">
          <span class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Quick Duration:</span>
          <button
            type="button"
            onclick={() => setDuration(30)}
            class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700"
          >
            30 mins
          </button>
          <button
            type="button"
            onclick={() => setDuration(45)}
            class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700"
          >
            45 mins
          </button>
          <button
            type="button"
            onclick={() => setDuration(60)}
            class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700"
          >
            60 mins
          </button>
        </div>

        <!-- Status & Color -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="appt-status" class="block text-xs font-semibold text-slate-300 mb-1">Appointment Status</label>
            <select
              id="appt-status"
              bind:value={status}
              class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
            >
              {#each statuses as st}
                <option value={st.id}>{st.label}</option>
              {/each}
            </select>
          </div>

          <div>
            <span class="block text-xs font-semibold text-slate-300 mb-1">Color Marker</span>
            <div class="flex items-center gap-2 pt-1.5">
              {#each colorOptions as c}
                <button
                  type="button"
                  onclick={() => (color = c.hex)}
                  class={`h-7 w-7 rounded-full transition-transform ${
                    color === c.hex ? "ring-2 ring-white ring-offset-2 ring-offset-slate-900 scale-110" : "opacity-75 hover:opacity-100"
                  }`}
                  style="background-color: {c.hex}"
                  title={c.name}
                ></button>
              {/each}
            </div>
          </div>
        </div>

        <!-- Reason -->
        <div>
          <label for="appt-reason" class="block text-xs font-semibold text-slate-300 mb-1">Reason for Visit</label>
          <input
            id="appt-reason"
            type="text"
            bind:value={reason}
            placeholder="e.g. Comprehensive Examination, Crown Preparation, Hygiene"
            class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
          />
        </div>

        <!-- Clinical / Frontdesk Notes -->
        <div>
          <label for="appt-notes" class="block text-xs font-semibold text-slate-300 mb-1">Clinical & Administrative Notes</label>
          <textarea
            id="appt-notes"
            bind:value={notes}
            rows="2"
            placeholder="Additional details, premedication required, special chair preferences..."
            class="w-full rounded-xl border border-slate-700 bg-slate-800/90 px-3.5 py-2 text-sm text-white focus:border-sky-500 focus:outline-none"
          ></textarea>
        </div>

        <!-- Action buttons -->
        <div class="flex items-center justify-between border-t border-slate-800 pt-4 mt-6">
          <div>
            {#if isEditing && ondelete}
              <button
                type="button"
                onclick={ondelete}
                class="px-3.5 py-2 text-xs font-semibold text-rose-400 hover:text-rose-300 hover:bg-rose-500/10 rounded-xl border border-rose-500/30 transition-colors"
              >
                Delete Appointment
              </button>
            {/if}
          </div>

          <div class="flex items-center gap-3">
            <button
              type="button"
              onclick={() => (showModal = false)}
              class="px-4 py-2 text-sm font-semibold text-slate-400 hover:text-white"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="rounded-xl bg-gradient-to-r from-sky-500 to-blue-600 px-5 py-2 text-sm font-semibold text-white shadow-lg shadow-sky-500/20 hover:from-sky-400 hover:to-blue-500 focus:outline-none"
            >
              {isEditing ? "Save Changes" : "Create Appointment"}
            </button>
          </div>
        </div>
      </form>
    </div>
  </div>
{/if}
