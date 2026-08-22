<script lang="ts">
  import type { Patient, Appointment, Provider, Operatory } from "@bindings/domain/models.js";
  import Modal from "./ui/Modal.svelte";
  import FormField from "./ui/FormField.svelte";
  import Input from "./ui/Input.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";
  import { getLocalDateString } from "$lib/date.js";

  let {
    showModal = $bindable(false),
    isEditing = false,
    patients = [],
    configuredProviders = [],
    configuredOperatories = [],
    selectedPatientId = $bindable(""),
    providerId = $bindable(""),
    operatoryId = $bindable(""),
    startDateStr = $bindable(getLocalDateString()),
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

  const statuses = $derived([
    { id: "scheduled", label: m.appts_status_scheduled() },
    { id: "confirmed", label: m.appts_status_confirmed() },
    { id: "arrived", label: m.appts_status_arrived() },
    { id: "in_chair", label: m.appts_status_in_chair() },
    { id: "completed", label: m.appts_status_completed() },
    { id: "cancelled", label: m.appts_status_cancelled() },
    { id: "no_show", label: m.appts_status_no_show() },
  ]);

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
    const [h, mins] = startTimeStr.split(":").map(Number);
    const date = new Date();
    date.setHours(h, mins + minutes, 0, 0);
    const endH = String(date.getHours()).padStart(2, "0");
    const endM = String(date.getMinutes()).padStart(2, "0");
    endTimeStr = `${endH}:${endM}`;
  }

  const modalTitle = $derived.by(() => {
    getLocaleVersion();
    return isEditing ? m.appt_modal_edit_title() : m.appt_modal_add_title();
  });
  const modalSubtitle = $derived.by(() => {
    getLocaleVersion();
    return m.appt_modal_subtitle();
  });
</script>

<Modal bind:showModal title={modalTitle} subtitle={modalSubtitle} icon="📅" maxWidth="max-w-xl">
  <form onsubmit={onsave} class="space-y-4">
    <!-- Patient Picker -->
    <FormField label={m.appt_label_patient()} forId="appt-patient" required>
      <select
        id="appt-patient"
        bind:value={selectedPatientId}
        required
        class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
      >
        <option value="" disabled>{m.appt_select_patient_placeholder()}</option>
        {#each patients as p}
          <option value={p.id}>
            {p.last_name}, {p.first_name} ({p.phone_primary || p.email || "No contact info"})
          </option>
        {/each}
      </select>
      {#if patients.length === 0}
        <p class="text-xs text-amber-400 mt-1">
          {m.appt_no_patients_warning()}
        </p>
      {/if}
    </FormField>

    <!-- Provider & Operatory -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <FormField label={m.appt_label_provider()} forId="appt-provider" required>
        <select
          id="appt-provider"
          bind:value={providerId}
          required
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
        >
          {#if configuredProviders.length === 0}
            <option value="">{m.appt_no_providers()}</option>
          {:else}
            {#each configuredProviders as prov}
              {#if prov.is_active || prov.id === providerId}
                <option value={prov.id}>{prov.name} ({prov.role})</option>
              {/if}
            {/each}
          {/if}
        </select>
      </FormField>

      <FormField label={m.appt_label_operatory()} forId="appt-operatory" required>
        <select
          id="appt-operatory"
          bind:value={operatoryId}
          required
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none"
        >
          {#if configuredOperatories.length === 0}
            <option value="">{m.appt_no_operatories()}</option>
          {:else}
            {#each configuredOperatories as op}
              {#if op.is_active || op.id === operatoryId}
                <option value={op.id}>{op.name} ({op.type})</option>
              {/if}
            {/each}
          {/if}
        </select>
      </FormField>
    </div>

    <!-- Date & Times -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
      <FormField label={m.appt_label_date()} forId="appt-date" required>
        <Input id="appt-date" type="date" bind:value={startDateStr} required />
      </FormField>

      <FormField label={m.appt_label_start_time()} forId="appt-start-time" required>
        <Input id="appt-start-time" type="time" bind:value={startTimeStr} required />
      </FormField>

      <FormField label={m.appt_label_end_time()} forId="appt-end-time" required>
        <Input id="appt-end-time" type="time" bind:value={endTimeStr} required />
      </FormField>
    </div>

    <!-- Quick Duration Presets -->
    <div class="flex items-center gap-2 pt-1">
      <span class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider"
        >{m.appt_label_quick_duration()}</span
      >
      <button
        type="button"
        onclick={() => setDuration(30)}
        class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 cursor-pointer"
      >
        {m.appt_quick_duration_30()}
      </button>
      <button
        type="button"
        onclick={() => setDuration(45)}
        class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 cursor-pointer"
      >
        {m.appt_quick_duration_45()}
      </button>
      <button
        type="button"
        onclick={() => setDuration(60)}
        class="px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 cursor-pointer"
      >
        {m.appt_quick_duration_60()}
      </button>
    </div>

    <!-- Status & Color -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <FormField label={m.appt_label_status()} forId="appt-status">
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
        <span class="text-xs font-semibold text-slate-300">{m.appt_label_color_marker()}</span>
        <div class="flex items-center gap-2 pt-1">
          {#each colorOptions as c}
            <button
              type="button"
              onclick={() => (color = c.hex)}
              class={`h-7 w-7 rounded-full transition-transform cursor-pointer ${
                color === c.hex
                  ? "ring-2 ring-white ring-offset-2 ring-offset-slate-900 scale-110"
                  : "opacity-75 hover:opacity-100"
              }`}
              style="background-color: {c.hex}"
              title={c.name}
            ></button>
          {/each}
        </div>
      </div>
    </div>

    <!-- Reason -->
    <FormField label={m.appt_label_reason()} forId="appt-reason">
      <Input
        id="appt-reason"
        type="text"
        bind:value={reason}
        placeholder={m.appt_reason_placeholder()}
      />
    </FormField>

    <!-- Clinical / Frontdesk Notes -->
    <FormField label={m.appt_label_notes()} forId="appt-notes">
      <textarea
        id="appt-notes"
        bind:value={notes}
        rows={2}
        placeholder={m.appt_notes_placeholder()}
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
            {m.appt_delete()}
          </button>
        {/if}
      </div>

      <div class="flex items-center gap-3">
        <button
          type="button"
          onclick={() => (showModal = false)}
          class="px-4 py-2 text-sm font-semibold text-slate-400 hover:text-white cursor-pointer"
        >
          {m.common_cancel()}
        </button>
        <button
          type="submit"
          class="rounded-xl bg-gradient-to-r from-sky-500 to-blue-600 px-5 py-2 text-sm font-semibold text-white shadow-lg shadow-sky-500/20 hover:from-sky-400 hover:to-blue-500 focus:outline-none cursor-pointer"
        >
          {isEditing ? m.appt_save_changes() : m.appt_create()}
        </button>
      </div>
    </div>
  </form>
</Modal>
