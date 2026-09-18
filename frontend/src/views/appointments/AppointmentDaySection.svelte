<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import type { Appointment } from "@bindings/domain/models.js";
  import { getLocalDateString } from "$lib/date.js";
  import { m } from "../../paraglide/messages.js";

  let {
    selectedDate = "",
    timeSlots = [],
    workingIntervals = null,
    workdayStartMinute = 8 * 60,
    filteredAppointments = [],
    formatSlotLabel,
    formatTime,
    getPatientName,
    getPatientPhone,
    getProviderName,
    getOperatoryName,
    statusBadges,
    getStatusColor,
    oneditappointment,
    onupdatestatus,
    confirmLabel = m.appts_action_confirm(),
    arrivedLabel = m.appts_action_arrived(),
    seatLabel = m.appts_action_seat(),
    completeLabel = m.appts_action_complete(),
    cancelLabel = m.appts_action_cancel(),
    noShowLabel = m.appts_action_no_show(),
  } = $props<{
    selectedDate: string;
    timeSlots: string[];
    workingIntervals?: [number, number][] | null;
    workdayStartMinute?: number;
    filteredAppointments: Appointment[];
    formatSlotLabel: (slot: string) => string;
    formatTime: (isoStr: string) => string;
    getPatientName: (id: string) => string;
    getPatientPhone: (id: string) => string;
    getProviderName: (id: string) => string;
    getOperatoryName: (id: string) => string;
    statusBadges: Record<string, { label: string; bg: string; text: string; border: string }>;
    getStatusColor: (status: string) => string;
    oneditappointment: (appt: Appointment) => void;
    onupdatestatus: (id: string, status: string) => void;
    confirmLabel?: string;
    arrivedLabel?: string;
    seatLabel?: string;
    completeLabel?: string;
    cancelLabel?: string;
    noShowLabel?: string;
  }>();

  const PX_PER_MIN = 1.6; // 96px per hour
  const ROW_H = PX_PER_MIN * 60;
  const DAY_MINUTES = 24 * 60;
  const MIN_APPT_HEIGHT = 22;
  const MIN_APPT_MINUTES = MIN_APPT_HEIGHT / PX_PER_MIN;

  let now = $state(new Date());
  let timer: any;
  let containerEl: HTMLDivElement;
  let expandedId = $state<string | null>(null);

  onMount(() => {
    timer = setInterval(() => {
      now = new Date();
    }, 60000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  let lastScrolledWorkdayStart = -1;
  $effect(() => {
    if (containerEl && workdayStartMinute !== lastScrolledWorkdayStart) {
      lastScrolledWorkdayStart = workdayStartMinute;
      const bufferHour = Math.max(0, Math.floor(workdayStartMinute / 60) - 1);
      containerEl.scrollTop = bufferHour * ROW_H;
    }
  });

  let isToday = $derived(selectedDate === getLocalDateString(now));
  let nowTopPx = $derived((now.getHours() * 60 + now.getMinutes()) * PX_PER_MIN);

  let hours = $derived(timeSlots.map((s: string) => parseInt(s.split(":")[0], 10) || 0));

  // Ranges of the day (in px, top/height) that fall outside business hours.
  let nonWorkingSegments = $derived.by(() => {
    if (workingIntervals == null) return [] as { top: number; height: number }[];
    if (workingIntervals.length === 0) {
      return [{ top: 0, height: DAY_MINUTES * PX_PER_MIN }];
    }
    const working = [...workingIntervals].sort((a, b) => a[0] - b[0]);
    const segments: [number, number][] = [];
    let cursor = 0;
    for (const [s, e] of working) {
      if (s > cursor) segments.push([cursor, s]);
      cursor = Math.max(cursor, e);
    }
    if (cursor < DAY_MINUTES) segments.push([cursor, DAY_MINUTES]);
    return segments.map(([s, e]) => ({ top: s * PX_PER_MIN, height: (e - s) * PX_PER_MIN }));
  });

  type LaidOutAppt = {
    appt: Appointment;
    top: number;
    height: number;
    leftPct: number;
    widthPct: number;
  };

  // Lay out appointments as a continuous timeline: each block is sized/positioned by its
  // actual start/end time, and overlapping appointments are placed in side-by-side columns.
  let laidOutAppointments = $derived.by(() => {
    type TimedAppt = { appt: Appointment; startMin: number; endMin: number };

    const items: TimedAppt[] = [];
    for (const appt of filteredAppointments as Appointment[]) {
      const start = new Date(appt.start_time);
      const end = new Date(appt.end_time);
      if (isNaN(start.getTime()) || isNaN(end.getTime())) continue;
      const startMin = start.getHours() * 60 + start.getMinutes();
      const endMin = Math.max(end.getHours() * 60 + end.getMinutes(), startMin + MIN_APPT_MINUTES);
      items.push({ appt, startMin, endMin });
    }
    items.sort((a, b) => a.startMin - b.startMin);

    const results: LaidOutAppt[] = [];
    let cluster: TimedAppt[] = [];
    let clusterEnd = -1;

    function flushCluster() {
      if (cluster.length === 0) return;
      const colEnds: number[] = [];
      const colByItem: number[] = [];
      for (const it of cluster) {
        let col = colEnds.findIndex((end) => end <= it.startMin);
        if (col === -1) {
          col = colEnds.length;
          colEnds.push(it.endMin);
        } else {
          colEnds[col] = it.endMin;
        }
        colByItem.push(col);
      }
      const totalCols = colEnds.length;
      cluster.forEach((it, i) => {
        const widthPct = 100 / totalCols;
        results.push({
          appt: it.appt,
          top: it.startMin * PX_PER_MIN,
          height: Math.max((it.endMin - it.startMin) * PX_PER_MIN, MIN_APPT_HEIGHT),
          leftPct: colByItem[i] * widthPct,
          widthPct,
        });
      });
      cluster = [];
    }

    for (const it of items) {
      if (cluster.length > 0 && it.startMin >= clusterEnd) {
        flushCluster();
        clusterEnd = -1;
      }
      cluster.push(it);
      clusterEnd = Math.max(clusterEnd, it.endMin);
    }
    flushCluster();

    return results;
  });

  function toggleExpanded(id: string) {
    expandedId = expandedId === id ? null : id;
  }
</script>

<div
  bind:this={containerEl}
  class="border border-slate-700/80 bg-slate-900/80 shadow-md overflow-y-auto max-h-[70vh] relative"
>
  <div class="flex relative" style="height: {DAY_MINUTES * PX_PER_MIN}px;">
    <!-- Time gutter -->
    <div class="w-24 flex-shrink-0 border-r border-slate-800 bg-slate-900/50 relative">
      {#each hours as h}
        <div
          class="absolute left-0 right-0 border-t border-slate-800"
          style="top: {h * ROW_H}px;"
        ></div>
        <div
          class="absolute left-0 right-0 px-3 text-xs font-semibold text-slate-400"
          style="top: {h * ROW_H + 4}px;"
        >
          {formatSlotLabel(`${String(h).padStart(2, "0")}:00`)}
        </div>
        <div
          class="absolute left-0 right-0 border-t border-dashed border-slate-700/70 pointer-events-none"
          style="top: {h * ROW_H + ROW_H * 0.25}px;"
        >
          <span
            class="absolute right-2 -translate-y-1/2 text-[10px] text-slate-600 bg-slate-900/60 px-1"
            >:15</span
          >
        </div>
        <div
          class="absolute left-0 right-0 border-t border-dashed border-slate-700/70 pointer-events-none"
          style="top: {h * ROW_H + ROW_H * 0.5}px;"
        >
          <span
            class="absolute right-2 -translate-y-1/2 text-[10px] text-slate-600 bg-slate-900/60 px-1"
            >:30</span
          >
        </div>
        <div
          class="absolute left-0 right-0 border-t border-dashed border-slate-700/70 pointer-events-none"
          style="top: {h * ROW_H + ROW_H * 0.75}px;"
        >
          <span
            class="absolute right-2 -translate-y-1/2 text-[10px] text-slate-600 bg-slate-900/60 px-1"
            >:45</span
          >
        </div>
      {/each}
    </div>

    <!-- Timeline -->
    <div class="flex-1 relative">
      {#each hours as h}
        <div
          class="absolute left-0 right-0 border-t border-slate-800 pointer-events-none"
          style="top: {h * ROW_H}px;"
        ></div>
        <div
          class="absolute left-0 right-0 border-t border-dashed border-slate-800 pointer-events-none"
          style="top: {h * ROW_H + ROW_H * 0.25}px;"
        ></div>
        <div
          class="absolute left-0 right-0 border-t border-dashed border-slate-800 pointer-events-none"
          style="top: {h * ROW_H + ROW_H * 0.5}px;"
        ></div>
        <div
          class="absolute left-0 right-0 border-t border-dashed border-slate-800 pointer-events-none"
          style="top: {h * ROW_H + ROW_H * 0.75}px;"
        ></div>
      {/each}

      {#each nonWorkingSegments as seg}
        <div
          class="absolute left-0 right-0 bg-slate-950/45 pointer-events-none"
          style="top: {seg.top}px; height: {seg.height}px;"
        ></div>
      {/each}

      {#each laidOutAppointments as item (item.appt.id)}
        {@const appt = item.appt}
        {@const isExpanded = expandedId === appt.id}
        {@const badge = statusBadges[appt.status] || statusBadges.scheduled}
        {@const color = getStatusColor(appt.status)}
        <div
          class="absolute border-l-4 shadow-md transition-all bg-slate-800/95 hover:brightness-110 cursor-pointer overflow-hidden"
          style={isExpanded
            ? `top: ${item.top}px; left: 4px; right: 4px; min-height: ${item.height}px; height: auto; z-index: 30; border-left-color: ${color};`
            : `top: ${item.top}px; left: calc(${item.leftPct}% + 2px); width: calc(${item.widthPct}% - 4px); height: ${item.height}px; z-index: 10; border-left-color: ${color};`}
          onclick={() => toggleExpanded(appt.id)}
          role="button"
          tabindex="0"
          onkeydown={(e) => e.key === "Enter" && toggleExpanded(appt.id)}
        >
          {#if isExpanded}
            <div class="p-3">
              <div class="flex items-start justify-between gap-2">
                <div>
                  <div class="text-sm font-bold text-white">{getPatientName(appt.patient_id)}</div>
                  <div class="text-xs text-slate-400 mt-0.5 flex items-center gap-2">
                    <span>{formatTime(appt.start_time)} - {formatTime(appt.end_time)}</span>
                    {#if getPatientPhone(appt.patient_id)}
                      <span>{getPatientPhone(appt.patient_id)}</span>
                    {/if}
                  </div>
                </div>
                <span
                  class={`shrink-0 text-[10px] font-bold uppercase tracking-wide px-2 py-0.5 rounded border ${badge.bg} ${badge.text} ${badge.border}`}
                >
                  {badge.label}
                </span>
              </div>

              {#if appt.reason}
                <div
                  class="mt-2 text-xs font-medium text-sky-200/90 bg-slate-900/60 rounded-lg px-2.5 py-1"
                >
                  {appt.reason}
                </div>
              {/if}

              <div
                class="mt-2.5 flex items-center justify-between text-[11px] text-slate-400 pt-2 border-t border-slate-700/50"
              >
                <span>{getProviderName(appt.provider_id)}</span>
                <span>{getOperatoryName(appt.operatory_id)}</span>
              </div>

              <div
                class="mt-2.5 flex items-center gap-1.5 pt-2 border-t border-slate-700/40"
                onclick={(e) => e.stopPropagation()}
                onkeydown={(e) => e.stopPropagation()}
                role="presentation"
              >
                {#if appt.status === "scheduled"}
                  <button
                    type="button"
                    onclick={() => onupdatestatus(appt.id, "confirmed")}
                    class="px-2 py-0.5 text-[10px] font-semibold text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 rounded border border-blue-500/30"
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
                {#if appt.status === "scheduled" || appt.status === "confirmed"}
                  <button
                    type="button"
                    onclick={() => onupdatestatus(appt.id, "no_show")}
                    class="px-2 py-0.5 text-[10px] font-semibold text-cyan-400 bg-cyan-500/10 hover:bg-cyan-500/20 rounded border border-cyan-500/30"
                  >
                    {noShowLabel}
                  </button>
                {/if}
                {#if appt.status === "scheduled" || appt.status === "confirmed" || appt.status === "arrived"}
                  <button
                    type="button"
                    onclick={() => onupdatestatus(appt.id, "cancelled")}
                    class="px-2 py-0.5 text-[10px] font-semibold text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 rounded border border-rose-500/30"
                  >
                    {cancelLabel}
                  </button>
                {/if}
                <button
                  type="button"
                  onclick={() => oneditappointment(appt)}
                  class="ml-auto px-2.5 py-0.5 text-[10px] font-semibold text-white bg-sky-500 hover:bg-sky-400 rounded"
                >
                  {m.appts_action_edit()}
                </button>
              </div>
            </div>
          {:else}
            <div class="h-full px-2 py-1 flex flex-col justify-center gap-0.5">
              <span class="text-[11px] font-bold text-white truncate leading-tight">
                {getPatientName(appt.patient_id)}
              </span>
              {#if item.height >= 34}
                <span class="text-[10px] text-slate-300/80 truncate leading-tight">
                  {formatTime(appt.start_time)} - {formatTime(appt.end_time)}
                </span>
              {/if}
            </div>
          {/if}
        </div>
      {/each}

      {#if isToday}
        <div
          class="absolute left-0 right-0 z-20 border-t-2 border-rose-500 pointer-events-none"
          style="top: {nowTopPx}px;"
        ></div>
      {/if}
    </div>
  </div>
</div>
