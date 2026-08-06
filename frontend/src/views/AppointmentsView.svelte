<script lang="ts">
  import type {
    Patient,
    Appointment,
    Provider,
    Operatory,
  } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import AppointmentStats from "../components/AppointmentStats.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../lib/locale.svelte.js";

  let {
    appointments = [],
    patients = [],
    providers = [],
    operatories = [],
    loading = false,
    selectedDate = $bindable(new Date().toISOString().split("T")[0]),
    selectedProvider = $bindable("all"),
    viewMode = $bindable("grid"),
    oneditappointment,
    onupdatestatus,
    ondeleteappointment,
  } = $props<{
    appointments: Appointment[];
    patients: Patient[];
    providers?: Provider[];
    operatories?: Operatory[];
    loading: boolean;
    selectedDate: string;
    selectedProvider: string;
    viewMode: "grid" | "agenda";
    onnewappointment: () => void;
    oneditappointment: (appt: Appointment) => void;
    onupdatestatus: (id: string, status: string) => void;
    ondeleteappointment: (id: string) => void;
  }>();

  function getProviderName(id: string): string {
    const p = providers.find((prov: Provider) => prov.id === id);
    if (p) return p.name;
    return id || m.patients_unassigned_id();
  }

  function getOperatoryName(id: string): string {
    const op = operatories.find((o: Operatory) => o.id === id);
    if (op) return op.name;
    return id || m.patients_unassigned_id();
  }

  const statusBadges: Record<string, { label: string; bg: string; text: string; border: string }> =
    $derived({
      scheduled: {
        label: m.appts_status_scheduled(),
        bg: "bg-blue-500/15",
        text: "text-blue-400",
        border: "border-blue-500/30",
      },
      confirmed: {
        label: m.appts_status_confirmed(),
        bg: "bg-sky-500/15",
        text: "text-sky-400",
        border: "border-sky-500/30",
      },
      arrived: {
        label: m.appts_status_arrived(),
        bg: "bg-amber-500/15",
        text: "text-amber-400",
        border: "border-amber-500/30",
      },
      in_chair: {
        label: m.appts_status_in_chair(),
        bg: "bg-purple-500/15",
        text: "text-purple-400",
        border: "border-purple-500/30",
      },
      completed: {
        label: m.appts_status_completed(),
        bg: "bg-emerald-500/15",
        text: "text-emerald-400",
        border: "border-emerald-500/30",
      },
      cancelled: {
        label: m.appts_status_cancelled(),
        bg: "bg-rose-500/15",
        text: "text-rose-400",
        border: "border-rose-500/30",
      },
      no_show: {
        label: m.appts_status_no_show(),
        bg: "bg-slate-500/15",
        text: "text-slate-400",
        border: "border-slate-500/30",
      },
    });

  // Time slots 8:00 AM to 6:00 PM
  const timeSlots = [
    "08:00",
    "09:00",
    "10:00",
    "11:00",
    "12:00",
    "13:00",
    "14:00",
    "15:00",
    "16:00",
    "17:00",
  ];

  function getPatientName(patientId: string): string {
    const p = patients.find((pat: Patient) => pat.id === patientId);
    if (!p) return "Unknown Patient";
    return `${p.first_name} ${p.last_name}`;
  }

  function getPatientPhone(patientId: string): string {
    const p = patients.find((pat: Patient) => pat.id === patientId);
    return p?.phone_primary || "";
  }

  function changeDay(delta: number) {
    const curr = new Date(selectedDate + "T00:00:00");
    curr.setDate(curr.getDate() + delta);
    selectedDate = curr.toISOString().split("T")[0];
  }

  function setToday() {
    selectedDate = new Date().toISOString().split("T")[0];
  }

  let filteredAppointments = $derived(
    appointments.filter((a: Appointment) => {
      if (selectedProvider !== "all" && a.provider_id !== selectedProvider) return false;
      return true;
    })
  );

  function formatTime(isoStr: string): string {
    if (!isoStr) return "";
    try {
      const d = new Date(isoStr);
      return d.toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        hour12: true,
      });
    } catch {
      return isoStr;
    }
  }

  function getApptHour(isoStr: string): string {
    if (!isoStr) return "";
    try {
      const d = new Date(isoStr);
      const h = String(d.getHours()).padStart(2, "0");
      return `${h}:00`;
    } catch {
      return "";
    }
  }

  function formattedDateHeading(dateStr: string): string {
    try {
      const d = new Date(dateStr + "T00:00:00");
      return d.toLocaleDateString(undefined, {
        weekday: "long",
        year: "numeric",
        month: "long",
        day: "numeric",
      });
    } catch {
      return dateStr;
    }
  }
</script>

<div class="space-y-6">
  <!-- Stats Summary -->
  <AppointmentStats appointments={filteredAppointments} />

  <!-- Control Bar -->
  <div
    class="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-slate-700/80 bg-slate-800/80 p-4 shadow-sm backdrop-blur"
  >
    <!-- Date Navigation -->
    <div class="flex items-center gap-2">
      <button
        type="button"
        onclick={() => changeDay(-1)}
        class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
      >
        {(getLocaleVersion(), m.appts_prev_day())}
      </button>
      <button
        type="button"
        onclick={setToday}
        class="rounded-lg border border-sky-500/30 bg-sky-500/10 px-3 py-1.5 text-xs font-semibold text-sky-400 hover:bg-sky-500/20"
      >
        {m.appts_today()}
      </button>
      <button
        type="button"
        onclick={() => changeDay(1)}
        class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
      >
        {m.appts_next_day()}
      </button>
      <input
        type="date"
        bind:value={selectedDate}
        class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1 text-sm text-slate-200 focus:border-sky-500 focus:outline-none"
      />
    </div>

    <!-- Provider Filter & View Switcher -->
    <div class="flex items-center gap-3">
      <div class="flex items-center gap-2">
        <label for="provider-filter" class="text-xs font-medium text-slate-400"
          >{m.appts_provider_filter()}</label
        >
        <select
          id="provider-filter"
          bind:value={selectedProvider}
          class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-200 focus:border-sky-500 focus:outline-none"
        >
          <option value="all">{m.appts_all_providers()}</option>
          {#each providers as p}
            <option value={p.id}>{p.name} ({p.role})</option>
          {/each}
        </select>
      </div>

      <!-- Grid / Agenda toggle -->
      <div class="flex rounded-lg border border-slate-700 bg-slate-900 p-0.5">
        <button
          type="button"
          onclick={() => (viewMode = "grid")}
          class={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${
            viewMode === "grid" ? "bg-sky-500 text-white" : "text-slate-400 hover:text-slate-200"
          }`}
        >
          {m.appts_view_grid()}
        </button>
        <button
          type="button"
          onclick={() => (viewMode = "agenda")}
          class={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${
            viewMode === "agenda" ? "bg-sky-500 text-white" : "text-slate-400 hover:text-slate-200"
          }`}
        >
          {m.appts_view_agenda()}
        </button>
      </div>
    </div>
  </div>

  <!-- Day Header -->
  <div class="flex items-center justify-between border-b border-slate-700/60 pb-2">
    <h2 class="text-lg font-bold text-slate-100 flex items-center gap-2">
      📅 {formattedDateHeading(selectedDate)}
    </h2>
    <span class="text-xs text-slate-400 font-medium">
      {filteredAppointments.length} appointment{filteredAppointments.length === 1 ? "" : "s"} scheduled
    </span>
  </div>

  {#if loading}
    <div class="flex items-center justify-center py-16 text-slate-400">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-sky-400"></div>
      <span class="ml-3 text-sm font-medium">{m.appts_loading_schedule()}</span>
    </div>
  {:else if viewMode === "grid"}
    <!-- DAY GRID VIEW -->
    <div class="rounded-xl border border-slate-700/80 bg-slate-900/80 shadow-md overflow-hidden">
      <div class="divide-y divide-slate-800">
        {#each timeSlots as slot}
          {@const slotAppts = filteredAppointments.filter(
            (a: Appointment) => getApptHour(a.start_time) === slot
          )}
          <div class="flex min-h-[96px] group hover:bg-slate-800/30 transition-colors">
            <!-- Hour label -->
            <div
              class="w-24 flex-shrink-0 border-r border-slate-800 p-3 text-xs font-semibold text-slate-400 bg-slate-900/50"
            >
              {slot}
            </div>

            <!-- Appointments for this time slot -->
            <div class="flex-1 p-2 flex flex-wrap gap-3 items-start">
              {#if slotAppts.length === 0}
                <div
                  class="h-full w-full flex items-center justify-start text-xs text-slate-600 italic px-2 py-4"
                >
                  {m.appts_no_appts_slot()}
                </div>
              {:else}
                {#each slotAppts as appt}
                  {@const badge = statusBadges[appt.status] || statusBadges.scheduled}
                  <div
                    class="group/card relative w-full sm:w-[320px] rounded-xl border border-l-4 p-3.5 shadow-md transition-all duration-150 hover:shadow-sky-500/10 hover:border-sky-500/50 bg-slate-800/90 text-left cursor-pointer"
                    style="border-left-color: {appt.color || '#3b82f6'};"
                    onclick={() => oneditappointment(appt)}
                    role="button"
                    tabindex="0"
                    onkeydown={(e) => e.key === "Enter" && oneditappointment(appt)}
                  >
                    <div class="flex items-start justify-between">
                      <div>
                        <div class="text-sm font-bold text-white flex items-center gap-1.5">
                          {getPatientName(appt.patient_id)}
                        </div>
                        <div class="text-xs text-slate-400 mt-0.5 flex items-center gap-2">
                          <span>⏱ {formatTime(appt.start_time)} - {formatTime(appt.end_time)}</span>
                          {#if getPatientPhone(appt.patient_id)}
                            <span>📞 {getPatientPhone(appt.patient_id)}</span>
                          {/if}
                        </div>
                      </div>

                      <span
                        class={`rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider ${badge.bg} ${badge.text} ${badge.border}`}
                      >
                        {badge.label}
                      </span>
                    </div>

                    {#if appt.reason}
                      <div
                        class="mt-2 text-xs font-medium text-sky-200/90 bg-slate-900/60 rounded-lg px-2.5 py-1"
                      >
                        📋 {appt.reason}
                      </div>
                    {/if}

                    <div
                      class="mt-2.5 flex items-center justify-between text-[11px] text-slate-400 pt-2 border-t border-slate-700/50"
                    >
                      <span>👤 {getProviderName(appt.provider_id)}</span>
                      <span>📍 {getOperatoryName(appt.operatory_id)}</span>
                    </div>

                    <!-- Quick Status Change Actions -->
                    <div
                      class="mt-2.5 flex items-center gap-1.5 pt-2 border-t border-slate-700/40"
                      onclick={(e) => e.stopPropagation()}
                      role="presentation"
                    >
                      {#if appt.status === "scheduled"}
                        <button
                          type="button"
                          onclick={() => onupdatestatus(appt.id, "confirmed")}
                          class="px-2 py-0.5 text-[10px] font-semibold text-sky-400 bg-sky-500/10 hover:bg-sky-500/20 rounded border border-sky-500/30"
                        >
                          {m.appts_action_confirm()}
                        </button>
                      {/if}
                      {#if appt.status === "scheduled" || appt.status === "confirmed"}
                        <button
                          type="button"
                          onclick={() => onupdatestatus(appt.id, "arrived")}
                          class="px-2 py-0.5 text-[10px] font-semibold text-amber-400 bg-amber-500/10 hover:bg-amber-500/20 rounded border border-amber-500/30"
                        >
                          {m.appts_action_arrived()}
                        </button>
                      {/if}
                      {#if appt.status === "arrived"}
                        <button
                          type="button"
                          onclick={() => onupdatestatus(appt.id, "in_chair")}
                          class="px-2 py-0.5 text-[10px] font-semibold text-purple-400 bg-purple-500/10 hover:bg-purple-500/20 rounded border border-purple-500/30"
                        >
                          {m.appts_action_seat()}
                        </button>
                      {/if}
                      {#if appt.status === "in_chair"}
                        <button
                          type="button"
                          onclick={() => onupdatestatus(appt.id, "completed")}
                          class="px-2 py-0.5 text-[10px] font-semibold text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 rounded border border-emerald-500/30"
                        >
                          {m.appts_action_complete()}
                        </button>
                      {/if}
                    </div>
                  </div>
                {/each}
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  {:else}
    <!-- AGENDA VIEW TABLE -->
    <div class="rounded-xl border border-slate-700/80 bg-slate-900/80 shadow-md overflow-hidden">
      {#if filteredAppointments.length === 0}
        <div class="py-16 text-center text-slate-400">
          <p class="text-base font-semibold">{m.appts_no_appts_date()}</p>
          <p class="text-xs text-slate-500 mt-1">{m.appts_no_appts_sub()}</p>
        </div>
      {:else}
        <table class="w-full text-left text-sm text-slate-200">
          <thead
            class="bg-slate-800/90 text-xs font-semibold uppercase tracking-wider text-slate-400 border-b border-slate-700"
          >
            <tr>
              <th class="px-4 py-3">{m.appt_label_start_time()}</th>
              <th class="px-4 py-3">{m.patients_th_name()}</th>
              <th class="px-4 py-3">{m.appt_label_reason()}</th>
              <th class="px-4 py-3">{m.appt_label_provider()}</th>
              <th class="px-4 py-3">{m.appt_label_operatory()}</th>
              <th class="px-4 py-3">{m.appt_label_status()}</th>
              <th class="px-4 py-3 text-right">{m.patients_th_actions()}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800">
            {#each filteredAppointments as appt}
              {@const badge = statusBadges[appt.status] || statusBadges.scheduled}
              <tr class="hover:bg-slate-800/50 transition-colors">
                <td class="px-4 py-3 font-medium whitespace-nowrap text-sky-400">
                  {formatTime(appt.start_time)} - {formatTime(appt.end_time)}
                </td>
                <td class="px-4 py-3 font-semibold text-white">
                  {getPatientName(appt.patient_id)}
                </td>
                <td class="px-4 py-3 text-slate-300">
                  {appt.reason || "—"}
                </td>
                <td class="px-4 py-3 text-slate-400 text-xs">
                  {getProviderName(appt.provider_id)}
                </td>
                <td class="px-4 py-3 text-slate-400 text-xs">
                  {getOperatoryName(appt.operatory_id)}
                </td>
                <td class="px-4 py-3">
                  <select
                    value={appt.status}
                    onchange={(e) => onupdatestatus(appt.id, (e.target as HTMLSelectElement).value)}
                    class={`rounded-lg border px-2.5 py-1 text-xs font-semibold focus:outline-none ${badge.bg} ${badge.text} ${badge.border}`}
                  >
                    <option value="scheduled">{m.appts_status_scheduled()}</option>
                    <option value="confirmed">{m.appts_status_confirmed()}</option>
                    <option value="arrived">{m.appts_status_arrived()}</option>
                    <option value="in_chair">{m.appts_status_in_chair()}</option>
                    <option value="completed">{m.appts_status_completed()}</option>
                    <option value="cancelled">{m.appts_status_cancelled()}</option>
                    <option value="no_show">{m.appts_status_no_show()}</option>
                  </select>
                </td>
                <td class="px-4 py-3 text-right space-x-2">
                  <button
                    type="button"
                    onclick={() => oneditappointment(appt)}
                    class="text-xs font-medium text-sky-400 hover:text-sky-300"
                  >
                    {m.patients_btn_edit()}
                  </button>
                  <button
                    type="button"
                    onclick={() => ondeleteappointment(appt.id)}
                    class="text-xs font-medium text-rose-400 hover:text-rose-300"
                  >
                    {m.appt_delete()}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  {/if}
</div>
