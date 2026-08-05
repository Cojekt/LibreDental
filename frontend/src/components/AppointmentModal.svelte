<script lang="ts">
  import type { Patient, Appointment, Provider, Operatory } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import Modal from "./ui/Modal.svelte";
  import FormField from "./ui/FormField.svelte";
  import Input from "./ui/Input.svelte";

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
    { id: "scheduled", label: "Scheduled" },
    { id: "confirmed", label: "Confirmed" },
    { id: "arrived", label: "Arrived" },
    { id: "in_chair", label: "In Chair" },
    { id: "completed", label: "Completed" },
    { id: "cancelled", label: "Cancelled" },
    { id: "no_show", label: "No Show" },
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

<Modal
  bind:showModal
  title={isEditing ? "Edit Appointment" : "Schedule New Appointment"}
  subtitle="Book or modify patient chair time and procedure details"
  icon="📅"
  maxWidth="max-w-xl"
>
  <form onsubmit={onsave} class="space-y-4">
    <!-- Patient Picker -->
    <FormField label="Patient" forId="appt-patient" required>
      <select
        id="appt-patient"
        bind:value={selectedPatientId}
        required
        class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
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
    </FormField>

    <!-- Provider & Operatory -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <FormField label="Provider" forId="appt-provider" required>
        <select
          id="appt-provider"
          bind:value={providerId}
          required
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
        >
          {#if configuredProviders.length === 0}
            <option value="">No Providers Configured</option>
          {:else}
            {#each configuredProviders as prov}
              <option value={prov.id}>{prov.name} ({prov.role})</option>
            {/each}
          {/if}
        </select>
      </FormField>

      <FormField label="Operatory / Chair" forId="appt-operatory" required>
        <select
          id="appt-operatory"
          bind:value={operatoryId}
          required
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
        >
          {#if configuredOperatories.length === 0}
            <option value="">No Operatories Configured</option>
          {:else}
            {#each configuredOperatories as op}
              <option value={op.id}>{op.name} ({op.type})</option>
            {/each}
          {/if}
        </select>
      </FormField>
    </div>

    <!-- Date & Times -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
      <FormField label="Date" forId="appt-date" required>
        <Input id="appt-date" type="date" bind:value={startDateStr} required />
      </FormField>

      <FormField label="Start Time" forId="appt-start-time" required>
        <Input id="appt-start-time" type="time" bind:value={startTimeStr} required />
      </FormField>

      <FormField label="End Time" forId="appt-end-time" required>
        <Input id="appt-end-time" type="time" bind:value={endTimeStr} required />
      </FormField>
    </div>

    <!-- Quick Duration Presets -->
    <div class="flex items-center gap-2 pt-1">
      <span class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Quick Duration:</span>
      <button
        type="button"
        onclick={() => setDuration(30)}
        class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 cursor-pointer"
      >
        30 mins
      </button>
      <button
        type="button"
        onclick={() => setDuration(45)}
        class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 cursor-pointer"
      >
        45 mins
      </button>
      <button
        type="button"
        onclick={() => setDuration(60)}
        class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 cursor-pointer"
      >
        60 mins
      </button>
    </div>

    <!-- Status & Color -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <FormField label="Appointment Status" forId="appt-status">
        <select
          id="appt-status"
          bind:value={status}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
        >
          {#each statuses as st}
            <option value={st.id}>{st.label}</option>
          {/each}
        </select>
      </FormField>

      <div class="flex flex-col gap-1.5">
        <span class="text-xs font-semibold text-slate-300">Color Marker</span>
        <div class="flex items-center gap-2 pt-1">
          {#each colorOptions as c}
            <button
              type="button"
              onclick={() => (color = c.hex)}
              class={`h-7 w-7 rounded-full transition-transform cursor-pointer ${
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
    <FormField label="Reason for Visit" forId="appt-reason">
      <Input
        id="appt-reason"
        type="text"
        bind:value={reason}
        placeholder="e.g. Comprehensive Examination, Crown Preparation, Hygiene"
      />
    </FormField>

    <!-- Clinical / Frontdesk Notes -->
    <FormField label="Clinical & Administrative Notes" forId="appt-notes">
      <textarea
        id="appt-notes"
        bind:value={notes}
        rows={2}
        placeholder="Additional details, premedication required, special chair preferences..."
        class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2 text-sm text-white focus:border-sky-500 focus:outline-none"
      ></textarea>
    </FormField>

    <!-- Action buttons -->
    <div class="flex items-center justify-between border-t border-slate-800 pt-4 mt-6">
      <div>
        {#if isEditing && ondelete}
          <button
            type="button"
            onclick={ondelete}
            class="px-3.5 py-2 text-xs font-semibold text-rose-400 hover:text-rose-300 hover:bg-rose-500/10 rounded-xl border border-rose-500/30 transition-colors cursor-pointer"
          >
            Delete Appointment
          </button>
        {/if}
      </div>

      <div class="flex items-center gap-3">
        <button
          type="button"
          onclick={() => (showModal = false)}
          class="px-4 py-2 text-sm font-semibold text-slate-400 hover:text-white cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="submit"
          class="rounded-xl bg-gradient-to-r from-sky-500 to-blue-600 px-5 py-2 text-sm font-semibold text-white shadow-lg shadow-sky-500/20 hover:from-sky-400 hover:to-blue-500 focus:outline-none cursor-pointer"
        >
          {isEditing ? "Save Changes" : "Create Appointment"}
        </button>
      </div>
    </div>
  </form>
</Modal>
