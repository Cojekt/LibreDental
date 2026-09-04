<script lang="ts">
  import type { Appointment } from "@bindings/domain/models.js";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import { m } from "../../paraglide/messages.js";

  let {
    weekDays = [],
    filteredAppointments = [],
    isSameDay,
    selectDateAndJumpToDay,
    oneditappointment,
    formatTime,
    getPatientName,
    getProviderName,
    statusBadges,
  } = $props<{
    weekDays: {
      dateStr: string;
      label: string;
      dayName: string;
      isToday: boolean;
      isSelected: boolean;
    }[];
    filteredAppointments: Appointment[];
    isSameDay: (d1: string, d2: string) => boolean;
    selectDateAndJumpToDay: (dateStr: string) => void;
    oneditappointment: (appt: Appointment) => void;
    formatTime: (isoStr: string) => string;
    getPatientName: (id: string) => string;
    getProviderName: (id: string) => string;
    statusBadges: Record<string, { label: string; bg: string; text: string; border: string }>;
  }>();
</script>

<div class="grid grid-cols-1 md:grid-cols-7 gap-3">
  {#each weekDays as day}
    {@const dayAppts = filteredAppointments.filter((a: Appointment) =>
      isSameDay(a.start_time, day.dateStr)
    )}
    <div
      class={`flex flex-col rounded-xl border transition-all ${
        day.isSelected
          ? "border-sky-500/60 bg-slate-900/90 ring-1 ring-sky-500/30 shadow-lg"
          : day.isToday
            ? "border-amber-500/40 bg-slate-900/80"
            : "border-slate-800 bg-slate-900/50"
      }`}
    >
      <!-- Day Header -->
      <button
        type="button"
        onclick={() => selectDateAndJumpToDay(day.dateStr)}
        class="p-3 text-left border-b border-slate-800 bg-slate-800/40 hover:bg-slate-800/80 rounded-t-xl transition-colors flex items-center justify-between group/weekhead"
      >
        <div>
          <div
            class="text-xs font-bold uppercase tracking-wider text-slate-400 group-hover/weekhead:text-sky-400 transition-colors"
          >
            {day.dayName}
          </div>
          <div class="text-sm font-extrabold text-white mt-0.5">{day.label}</div>
        </div>
        {#if day.isToday}
          <span
            class="text-[10px] uppercase font-bold text-amber-400 bg-amber-500/10 px-1.5 py-0.5 rounded border border-amber-500/30"
            >{m.appts_today_badge()}
          </span>
        {/if}
      </button>

      <!-- Appointments List for Day -->
      <div class="p-2 space-y-2 flex-1 overflow-y-auto max-h-[650px] min-h-[160px]">
        {#if dayAppts.length === 0}
          <div
            class="h-full flex items-center justify-center text-center text-xs text-slate-600 italic py-6"
          >
            {m.appts_no_appts()}
          </div>
        {:else}
          {#each dayAppts as appt}
            {@const badge = statusBadges[appt.status] || statusBadges.scheduled}
            <div
              class="relative rounded-lg border border-l-4 p-2.5 shadow-sm hover:border-sky-500/50 bg-slate-800/90 text-left cursor-pointer transition-all hover:scale-[1.01]"
              style="border-left-color: {appt.color || '#3b82f6'};"
              onclick={() => oneditappointment(appt)}
              role="button"
              tabindex="0"
              onkeydown={(e) => e.key === "Enter" && oneditappointment(appt)}
            >
              <div class="text-xs font-bold text-white truncate">
                {getPatientName(appt.patient_id)}
              </div>
              <div class="text-[11px] text-sky-400 font-semibold mt-1">
                {formatTime(appt.start_time)}
              </div>
              {#if appt.reason}
                <div
                  class="text-[11px] text-slate-300 truncate mt-1 bg-slate-900/60 px-1.5 py-0.5 rounded"
                >
                  {appt.reason}
                </div>
              {/if}
              <div class="mt-2 flex items-center justify-between text-[10px]">
                <span class="text-slate-400 truncate max-w-[90px]"
                  >{getProviderName(appt.provider_id)}</span
                >
                <StatusBadge variant={appt.status} label={badge.label} size="sm" />
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/each}
</div>
