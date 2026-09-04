<script lang="ts">
  import type { Appointment } from "@bindings/domain/models.js";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

  let {
    filteredAppointments = [],
    jumpToDateFromAgenda,
    formatApptDate,
    formatTime,
    getPatientName,
    getProviderName,
    getOperatoryName,
    onupdatestatus,
    oneditappointment,
    ondeleteappointment,
  } = $props<{
    filteredAppointments: Appointment[];
    jumpToDateFromAgenda: (isoStr: string) => void;
    formatApptDate: (isoStr: string) => string;
    formatTime: (isoStr: string) => string;
    getPatientName: (id: string) => string;
    getProviderName: (id: string) => string;
    getOperatoryName: (id: string) => string;
    onupdatestatus: (id: string, status: string) => void;
    oneditappointment: (appt: Appointment) => void;
    ondeleteappointment: (id: string) => void;
  }>();
</script>

<div class="rounded-xl border border-slate-700/80 bg-slate-900/80 shadow-md overflow-hidden">
  {#if filteredAppointments.length === 0}
    <EmptyState title={m.appts_no_appts_date()} subtitle={m.appts_no_appts_sub()} />
  {:else}
    <table class="w-full text-left text-sm text-slate-200">
      <thead
        class="bg-slate-800/90 text-xs font-semibold uppercase tracking-wider text-slate-400 border-b border-slate-700"
      >
        <tr>
          <th class="px-4 py-3">{m.appt_label_date_time()}</th>
          <th class="px-4 py-3">{m.appts_th_patient_proc()}</th>
          <th class="px-4 py-3">{m.appt_label_reason()}</th>
          <th class="px-4 py-3">{m.appts_th_prov_chair()}</th>
          <th class="px-4 py-3">{m.appt_label_operatory()}</th>
          <th class="px-4 py-3">{m.appts_th_status()}</th>
          <th class="px-4 py-3 text-right">{m.patients_th_actions()}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-800">
        {#each filteredAppointments as appt}
          <tr class="hover:bg-slate-800/50 transition-colors">
            <td class="px-4 py-3 font-medium whitespace-nowrap">
              <button
                type="button"
                onclick={() => jumpToDateFromAgenda(appt.start_time)}
                class="text-left group/date focus:outline-none transition-colors"
                title="Click to view this date in Calendar View"
              >
                <div
                  class="text-slate-200 font-semibold text-xs flex items-center gap-1.5 group-hover/date:text-sky-400 group-hover/date:underline"
                >
                  <span class="text-slate-400"></span>
                  {formatApptDate(appt.start_time)}
                </div>
                <div class="text-sky-400 text-xs mt-1 flex items-center gap-1.5">
                  <span class="text-slate-400"></span>
                  {formatTime(appt.start_time)} - {formatTime(appt.end_time)}
                </div>
              </button>
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
                class="rounded-lg border px-2.5 py-1 text-xs font-semibold focus:outline-none bg-slate-900 text-slate-200 border-slate-700"
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
