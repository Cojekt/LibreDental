<script lang="ts">
  import type { Appointment } from "@bindings/domain/models.js";

  let {
    monthGrid = [],
    filteredAppointments = [],
    isSameDay,
    selectDateAndJumpToDay,
    oneditappointment,
    formatTime,
    getPatientName,
  } = $props<{
    monthGrid: {
      dateStr: string;
      dayNum: number;
      isCurrentMonth: boolean;
      isToday: boolean;
      isSelected: boolean;
    }[];
    filteredAppointments: Appointment[];
    isSameDay: (d1: string, d2: string) => boolean;
    selectDateAndJumpToDay: (dateStr: string) => void;
    oneditappointment: (appt: Appointment) => void;
    formatTime: (isoStr: string) => string;
    getPatientName: (id: string) => string;
  }>();
</script>

<div class="rounded-xl border border-slate-700/80 bg-slate-900/80 shadow-md overflow-hidden">
  <!-- Day Names Header -->
  <div
    class="grid grid-cols-7 border-b border-slate-800 bg-slate-800/80 text-center text-xs font-bold text-slate-400 py-2.5"
  >
    <div>Sun</div>
    <div>Mon</div>
    <div>Tue</div>
    <div>Wed</div>
    <div>Thu</div>
    <div>Fri</div>
    <div>Sat</div>
  </div>

  <!-- 42 Calendar Cells -->
  <div class="grid grid-cols-7 divide-x divide-y divide-slate-800/80">
    {#each monthGrid as cell}
      {@const cellAppts = filteredAppointments.filter((a: Appointment) =>
        isSameDay(a.start_time, cell.dateStr)
      )}
      <div
        class={`min-h-[110px] p-1.5 flex flex-col transition-colors group ${
          !cell.isCurrentMonth
            ? "bg-slate-950/40 text-slate-600 opacity-50"
            : cell.isSelected
              ? "bg-sky-950/20 text-slate-200"
              : "bg-slate-900/40 text-slate-300 hover:bg-slate-800/30"
        }`}
      >
        <!-- Cell Header -->
        <div class="flex items-center justify-between mb-1 px-1">
          <button
            type="button"
            onclick={() => selectDateAndJumpToDay(cell.dateStr)}
            class={`text-xs font-bold px-1.5 py-0.5 rounded transition-all hover:bg-sky-500/20 hover:text-sky-400 ${
              cell.isToday
                ? "bg-amber-500 text-slate-950 font-black shadow-sm"
                : cell.isSelected
                  ? "bg-sky-500 text-white font-bold"
                  : "text-slate-400"
            }`}
          >
            {cell.dayNum}
          </button>
          {#if cellAppts.length > 0}
            <span
              class="text-[10px] font-semibold text-slate-400 bg-slate-800 px-1.5 py-0.2 rounded-full border border-slate-700"
            >
              {cellAppts.length}
            </span>
          {/if}
        </div>

        <!-- Appointment List Chips -->
        <div class="space-y-1 flex-1 overflow-y-auto max-h-[95px]">
          {#each cellAppts.slice(0, 3) as appt}
            <button
              type="button"
              onclick={() => oneditappointment(appt)}
              class="w-full text-left rounded px-1.5 py-0.5 text-[11px] font-medium border border-l-2 bg-slate-800/90 hover:bg-slate-700/80 truncate flex items-center gap-1 transition-all"
              style="border-left-color: {appt.color || '#3b82f6'};"
            >
              <span class="text-sky-400 text-[10px] font-semibold">{formatTime(appt.start_time)}</span>
              <span class="font-semibold text-white truncate">{getPatientName(appt.patient_id)}</span>
            </button>
          {/each}

          {#if cellAppts.length > 3}
            <button
              type="button"
              onclick={() => selectDateAndJumpToDay(cell.dateStr)}
              class="w-full text-center text-[10px] font-bold text-sky-400 hover:underline pt-0.5"
            >
              +{cellAppts.length - 3} more
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>
</div>
