<script lang="ts">
  import type { Appointment } from "@bindings/domain/models.js";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";

  let { appointments = [], compact = false } = $props<{
    appointments: Appointment[];
    compact?: boolean;
  }>();

  let scheduledCount = $derived(
    appointments.filter((a: Appointment) => a.status === "scheduled").length
  );
  let confirmedCount = $derived(
    appointments.filter((a: Appointment) => a.status === "confirmed").length
  );
  let arrivedCount = $derived(
    appointments.filter((a: Appointment) => a.status === "arrived").length
  );
  let inChairCount = $derived(
    appointments.filter((a: Appointment) => a.status === "in_chair").length
  );
  let completedCount = $derived(
    appointments.filter((a: Appointment) => a.status === "completed").length
  );
  let cancelledCount = $derived(
    appointments.filter((a: Appointment) => a.status === "cancelled").length
  );
  let noShowCount = $derived(
    appointments.filter((a: Appointment) => a.status === "no_show").length
  );
</script>

<div class={`grid grid-cols-2 sm:grid-cols-4 md:grid-cols-7 ${compact ? "gap-2" : "gap-3"}`}>
  <!-- Scheduled -->
  <div
    class={`rounded-lg border-l-4 border border-slate-500/40 border-l-slate-400 bg-slate-500/15 shadow-sm backdrop-blur ${compact ? "p-2" : "p-3.5"}`}
  >
    <div class="flex items-center justify-between text-[11px] font-semibold text-slate-300">
      <span>{(getLocaleVersion(), m.appts_status_scheduled())}</span>
      <span class="h-1.5 w-1.5 rounded-full bg-slate-400"></span>
    </div>
    <div class={`font-bold text-slate-100 ${compact ? "mt-0.5 text-base" : "mt-2 text-2xl"}`}>
      {scheduledCount}
    </div>
  </div>

  <!-- Confirmed -->
  <div
    class={`rounded-lg border-l-4 border border-blue-500/40 border-l-blue-400 bg-blue-500/15 shadow-sm backdrop-blur ${compact ? "p-2" : "p-3.5"}`}
  >
    <div class="flex items-center justify-between text-[11px] font-semibold text-blue-300">
      <span>{m.appts_status_confirmed()}</span>
      <span class="h-1.5 w-1.5 rounded-full bg-blue-400"></span>
    </div>
    <div class={`font-bold text-blue-100 ${compact ? "mt-0.5 text-base" : "mt-2 text-2xl"}`}>
      {confirmedCount}
    </div>
  </div>

  <!-- Arrived / Waiting -->
  <div
    class={`rounded-lg border-l-4 border border-amber-500/40 border-l-amber-400 bg-amber-500/15 shadow-sm backdrop-blur ${compact ? "p-2" : "p-3.5"}`}
  >
    <div class="flex items-center justify-between text-[11px] font-semibold text-amber-300">
      <span>{m.appts_status_arrived()}</span>
      <span class="h-1.5 w-1.5 rounded-full bg-amber-400 animate-pulse"></span>
    </div>
    <div class={`font-bold text-amber-100 ${compact ? "mt-0.5 text-base" : "mt-2 text-2xl"}`}>
      {arrivedCount}
    </div>
  </div>

  <!-- In Chair -->
  <div
    class={`rounded-lg border-l-4 border border-purple-500/40 border-l-purple-400 bg-purple-500/15 shadow-sm backdrop-blur ${compact ? "p-2" : "p-3.5"}`}
  >
    <div class="flex items-center justify-between text-[11px] font-semibold text-purple-300">
      <span>{m.appts_status_in_chair()}</span>
      <span class="h-1.5 w-1.5 rounded-full bg-purple-400 animate-pulse"></span>
    </div>
    <div class={`font-bold text-purple-100 ${compact ? "mt-0.5 text-base" : "mt-2 text-2xl"}`}>
      {inChairCount}
    </div>
  </div>

  <!-- Completed -->
  <div
    class={`rounded-lg border-l-4 border border-emerald-500/40 border-l-emerald-400 bg-emerald-500/15 shadow-sm backdrop-blur ${compact ? "p-2" : "p-3.5"}`}
  >
    <div class="flex items-center justify-between text-[11px] font-semibold text-emerald-300">
      <span>{m.appts_status_completed()}</span>
      <span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span>
    </div>
    <div class={`font-bold text-emerald-100 ${compact ? "mt-0.5 text-base" : "mt-2 text-2xl"}`}>
      {completedCount}
    </div>
  </div>

  <!-- Cancelled -->
  <div
    class={`rounded-lg border-l-4 border border-rose-500/40 border-l-rose-400 bg-rose-500/15 shadow-sm backdrop-blur ${compact ? "p-2" : "p-3.5"}`}
  >
    <div class="flex items-center justify-between text-[11px] font-semibold text-rose-300">
      <span>{m.appts_status_cancelled()}</span>
      <span class="h-1.5 w-1.5 rounded-full bg-rose-400"></span>
    </div>
    <div class={`font-bold text-rose-100 ${compact ? "mt-0.5 text-base" : "mt-2 text-2xl"}`}>
      {cancelledCount}
    </div>
  </div>

  <!-- No Show -->
  <div
    class={`rounded-lg border-l-4 border border-cyan-500/40 border-l-cyan-400 bg-cyan-500/15 shadow-sm backdrop-blur ${compact ? "p-2" : "p-3.5"}`}
  >
    <div class="flex items-center justify-between text-[11px] font-semibold text-cyan-300">
      <span>{m.appts_status_no_show()}</span>
      <span class="h-1.5 w-1.5 rounded-full bg-cyan-400"></span>
    </div>
    <div class={`font-bold text-cyan-100 ${compact ? "mt-0.5 text-base" : "mt-2 text-2xl"}`}>
      {noShowCount}
    </div>
  </div>
</div>
