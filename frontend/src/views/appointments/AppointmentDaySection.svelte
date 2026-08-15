<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import type { Appointment } from "@bindings/domain/models.js";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import { getLocalDateString } from "$lib/date.js";

  let {
    selectedDate = "",
    timeSlots = [],
    filteredAppointments = [],
    getApptHour,
    formatSlotLabel,
    formatTime,
    getPatientName,
    getPatientPhone,
    getProviderName,
    getOperatoryName,
    statusBadges,
    oneditappointment,
    onupdatestatus,
    noApptsLabel = "No appointments scheduled for this slot",
    confirmLabel = "Confirm",
    arrivedLabel = "Arrived",
    seatLabel = "Seat",
    completeLabel = "Complete",
  } = $props<{
    selectedDate: string;
    timeSlots: string[];
    filteredAppointments: Appointment[];
    getApptHour: (isoStr: string) => string;
    formatSlotLabel: (slot: string) => string;
    formatTime: (isoStr: string) => string;
    getPatientName: (id: string) => string;
    getPatientPhone: (id: string) => string;
    getProviderName: (id: string) => string;
    getOperatoryName: (id: string) => string;
    statusBadges: Record<string, { label: string; bg: string; text: string; border: string }>;
    oneditappointment: (appt: Appointment) => void;
    onupdatestatus: (id: string, status: string) => void;
    noApptsLabel?: string;
    confirmLabel?: string;
    arrivedLabel?: string;
    seatLabel?: string;
    completeLabel?: string;
  }>();

  let now = $state(new Date());
  let timer: any;

  onMount(() => {
    timer = setInterval(() => {
      now = new Date();
    }, 60000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  let isToday = $derived(selectedDate === getLocalDateString(now));

  function getIndicatorTopPos(slotStr: string): number {
    if (!isToday) return -1;
    try {
      const [hStr] = slotStr.split(":");
      const slotHour = parseInt(hStr, 10);
      if (slotHour === now.getHours()) {
        return (now.getMinutes() / 60) * 100;
      }
    } catch {
      // ignore
    }
    return -1;
  }
</script>

<div class="rounded-xl border border-slate-700/80 bg-slate-900/80 shadow-md overflow-hidden">
  <div class="divide-y divide-slate-800">
    {#each timeSlots as slot}
      {@const slotAppts = filteredAppointments.filter(
        (a: Appointment) => getApptHour(a.start_time) === slot
      )}
      <div class="flex min-h-[96px] group hover:bg-slate-800/30 transition-colors relative">
        {#if getIndicatorTopPos(slot) >= 0}
          <div
            class="absolute left-0 right-0 z-10 border-t-2 border-rose-500 pointer-events-none"
            style="top: {getIndicatorTopPos(slot)}%;"
          >
            <div
              class="absolute -top-1.5 left-[5.7rem] w-3 h-3 bg-rose-500 rounded-full shadow-md shadow-rose-500/50"
            ></div>
          </div>
        {/if}
        <div
          class="w-24 flex-shrink-0 border-r border-slate-800 p-3 text-xs font-semibold text-slate-400 bg-slate-900/50"
        >
          {formatSlotLabel(slot)}
        </div>

        <div class="flex-1 p-2 flex flex-wrap gap-3 items-start">
          {#if slotAppts.length === 0}
            <div
              class="h-full w-full flex items-center justify-start text-xs text-slate-600 italic px-2 py-4"
            >
              {noApptsLabel}
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

                  <StatusBadge variant={appt.status} label={badge.label} />
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

                <!-- Quick Status Actions -->
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
                      {confirmLabel}
                    </button>
                  {/if}
                  {#if appt.status === "scheduled" || appt.status === "confirmed"}
                    <button
                      type="button"
                      onclick={() => onupdatestatus(appt.id, "arrived")}
                      class="px-2 py-0.5 text-[10px] font-semibold text-amber-400 bg-amber-500/10 hover:bg-amber-500/20 rounded border border-amber-500/30"
                    >
                      {arrivedLabel}
                    </button>
                  {/if}
                  {#if appt.status === "arrived"}
                    <button
                      type="button"
                      onclick={() => onupdatestatus(appt.id, "in_chair")}
                      class="px-2 py-0.5 text-[10px] font-semibold text-purple-400 bg-purple-500/10 hover:bg-purple-500/20 rounded border border-purple-500/30"
                    >
                      {seatLabel}
                    </button>
                  {/if}
                  {#if appt.status === "in_chair"}
                    <button
                      type="button"
                      onclick={() => onupdatestatus(appt.id, "completed")}
                      class="px-2 py-0.5 text-[10px] font-semibold text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 rounded border border-emerald-500/30"
                    >
                      {completeLabel}
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
