<script lang="ts">
  import type { Appointment } from "@bindings/domain/models.js";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";

  let { appointments = [] } = $props<{
    appointments: Appointment[];
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
</script>

<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
  <!-- Scheduled -->
  <div class="rounded-xl border border-slate-700/80 bg-slate-800/80 p-3.5 shadow-sm backdrop-blur">
    <div class="flex items-center justify-between text-xs font-semibold text-slate-400">
      <span>{(getLocaleVersion(), m.appts_status_scheduled())}</span>
      <span class="h-2 w-2 rounded-full bg-blue-500"></span>
    </div>
    <div class="mt-2 text-2xl font-bold text-slate-100">{scheduledCount}</div>
  </div>

  <!-- Confirmed -->
  <div class="rounded-xl border border-slate-700/80 bg-slate-800/80 p-3.5 shadow-sm backdrop-blur">
    <div class="flex items-center justify-between text-xs font-semibold text-sky-400">
      <span>{m.appts_status_confirmed()}</span>
      <span class="h-2 w-2 rounded-full bg-sky-400"></span>
    </div>
    <div class="mt-2 text-2xl font-bold text-sky-300">{confirmedCount}</div>
  </div>

  <!-- Arrived / Waiting -->
  <div class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-3.5 shadow-sm backdrop-blur">
    <div class="flex items-center justify-between text-xs font-semibold text-amber-400">
      <span>{m.appts_status_arrived()}</span>
      <span class="h-2 w-2 rounded-full bg-amber-400 animate-pulse"></span>
    </div>
    <div class="mt-2 text-2xl font-bold text-amber-300">{arrivedCount}</div>
  </div>

  <!-- In Chair -->
  <div
    class="rounded-xl border border-purple-500/30 bg-purple-500/10 p-3.5 shadow-sm backdrop-blur"
  >
    <div class="flex items-center justify-between text-xs font-semibold text-purple-400">
      <span>{m.appts_status_in_chair()}</span>
      <span class="h-2 w-2 rounded-full bg-purple-400 animate-pulse"></span>
    </div>
    <div class="mt-2 text-2xl font-bold text-purple-300">{inChairCount}</div>
  </div>

  <!-- Completed -->
  <div
    class="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-3.5 shadow-sm backdrop-blur"
  >
    <div class="flex items-center justify-between text-xs font-semibold text-emerald-400">
      <span>{m.appts_status_completed()}</span>
      <span class="h-2 w-2 rounded-full bg-emerald-400"></span>
    </div>
    <div class="mt-2 text-2xl font-bold text-emerald-300">{completedCount}</div>
  </div>
</div>
