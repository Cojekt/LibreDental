<script lang="ts">
  import type { Patient, Appointment, Provider, Operatory } from "@bindings/domain/models.js";
  import AppointmentStats from "../components/AppointmentStats.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";
  import { getLocalDateString, isSameDay, parseLocalDate } from "$lib/date.js";

  let {
    appointments = [],
    patients = [],
    providers = [],
    operatories = [],
    loading = false,
    selectedDate = $bindable(getLocalDateString()),
    selectedProvider = $bindable("all"),
    viewMode = $bindable("calendar"),
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
    viewMode: "calendar" | "grid" | "agenda";
    onnewappointment: () => void;
    oneditappointment: (appt: Appointment) => void;
    onupdatestatus: (id: string, status: string) => void;
    ondeleteappointment: (id: string) => void;
  }>();

  let calendarView = $state<"day" | "week" | "month">("day");

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

  // Dynamic time slots derived from filtered appointments (default 07:00 to 18:00)
  let timeSlots = $derived.by(() => {
    let minH = 7;
    let maxH = 18;

    if (filteredAppointments.length > 0) {
      for (const a of filteredAppointments) {
        if (!a.start_time) continue;
        try {
          const d = new Date(a.start_time);
          const h = d.getHours();
          if (!isNaN(h)) {
            if (h < minH) minH = h;
            if (h > maxH) maxH = h;
          }
        } catch {
          // ignore
        }
      }
    }

    const slots: string[] = [];
    for (let h = minH; h <= maxH; h++) {
      slots.push(`${String(h).padStart(2, "0")}:00`);
    }
    return slots;
  });

  function formatSlotLabel(slot: string): string {
    try {
      const [hStr] = slot.split(":");
      const h = parseInt(hStr, 10);
      const ampm = h >= 12 ? "PM" : "AM";
      const h12 = h % 12 === 0 ? 12 : h % 12;
      return `${String(h12).padStart(2, "0")}:00 ${ampm}`;
    } catch {
      return slot;
    }
  }

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
    const curr = parseLocalDate(selectedDate);
    curr.setDate(curr.getDate() + delta);
    selectedDate = getLocalDateString(curr);
  }

  function changeWeek(delta: number) {
    const curr = parseLocalDate(selectedDate);
    curr.setDate(curr.getDate() + delta * 7);
    selectedDate = getLocalDateString(curr);
  }

  function changeMonth(delta: number) {
    const curr = parseLocalDate(selectedDate);
    curr.setMonth(curr.getMonth() + delta);
    selectedDate = getLocalDateString(curr);
  }

  function setToday() {
    selectedDate = getLocalDateString();
  }

  function selectDateAndJumpToDay(dateStr: string) {
    selectedDate = dateStr;
    calendarView = "day";
  }

  function jumpToDateFromAgenda(isoStr: string) {
    if (!isoStr) return;
    try {
      const localDate = getLocalDateString(isoStr);
      if (localDate) {
        selectedDate = localDate;
        viewMode = "calendar";
        calendarView = "day";
      }
    } catch (e) {
      console.error("Failed to parse date for jumping:", e);
    }
  }

  // Week Days (Sun - Sat) based on selectedDate
  let weekDays = $derived.by(() => {
    const curr = parseLocalDate(selectedDate);
    const dayOfWeek = isNaN(curr.getTime()) ? 0 : curr.getDay();
    const sunday = new Date(curr);
    sunday.setDate(curr.getDate() - dayOfWeek);

    const todayStr = getLocalDateString();
    const days: {
      dateStr: string;
      label: string;
      dayName: string;
      isToday: boolean;
      isSelected: boolean;
    }[] = [];

    for (let i = 0; i < 7; i++) {
      const d = new Date(sunday);
      d.setDate(sunday.getDate() + i);
      const dateStr = getLocalDateString(d);
      const dayName = d.toLocaleDateString(undefined, { weekday: "short" });
      const label = d.toLocaleDateString(undefined, { month: "short", day: "numeric" });

      days.push({
        dateStr,
        label,
        dayName,
        isToday: dateStr === todayStr,
        isSelected: dateStr === selectedDate,
      });
    }
    return days;
  });

  // Month Grid (42 cells) based on selectedDate
  let monthGrid = $derived.by(() => {
    const curr = parseLocalDate(selectedDate);
    const year = isNaN(curr.getTime()) ? new Date().getFullYear() : curr.getFullYear();
    const month = isNaN(curr.getTime()) ? new Date().getMonth() : curr.getMonth();

    const firstOfMonth = new Date(year, month, 1);
    const startDayOfWeek = firstOfMonth.getDay();

    const startDate = new Date(firstOfMonth);
    startDate.setDate(startDate.getDate() - startDayOfWeek);

    const todayStr = getLocalDateString();
    const grid: {
      dateStr: string;
      dayNum: number;
      isCurrentMonth: boolean;
      isToday: boolean;
      isSelected: boolean;
    }[] = [];

    for (let i = 0; i < 42; i++) {
      const d = new Date(startDate);
      d.setDate(startDate.getDate() + i);
      const dateStr = getLocalDateString(d);

      grid.push({
        dateStr,
        dayNum: d.getDate(),
        isCurrentMonth: d.getMonth() === month,
        isToday: dateStr === todayStr,
        isSelected: dateStr === selectedDate,
      });
    }
    return grid;
  });

  let monthYearHeading = $derived.by(() => {
    try {
      const d = parseLocalDate(selectedDate);
      return d.toLocaleDateString(undefined, { month: "long", year: "numeric" });
    } catch {
      return selectedDate;
    }
  });

  let weekRangeHeading = $derived.by(() => {
    if (weekDays.length === 0) return "";
    const start = weekDays[0].label;
    const end = weekDays[6].label;
    const year = parseLocalDate(selectedDate).getFullYear();
    return `${start} - ${end}, ${year}`;
  });

  let filteredAppointments = $derived(
    appointments
      .filter((a: Appointment) => {
        if (selectedProvider !== "all" && a.provider_id !== selectedProvider) return false;
        const isCalendar = viewMode === "calendar" || viewMode === "grid";
        if (isCalendar && calendarView === "day") {
          return isSameDay(a.start_time, selectedDate);
        }
        return true;
      })
      .sort((a: Appointment, b: Appointment) => {
        return new Date(a.start_time).getTime() - new Date(b.start_time).getTime();
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
      const d = parseLocalDate(dateStr);
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

  function formatApptDate(isoStr: string): string {
    if (!isoStr) return "";
    try {
      const d = new Date(isoStr);
      return d.toLocaleDateString(undefined, {
        weekday: "short",
        month: "short",
        day: "numeric",
        year: "numeric",
      });
    } catch {
      return isoStr;
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
    <!-- View Switcher (Calendar vs Agenda) & Sub-tabs (Day | Week | Month) -->
    <div class="flex items-center gap-3">
      <div class="flex rounded-lg border border-slate-700 bg-slate-900 p-0.5">
        <button
          type="button"
          onclick={() => (viewMode = "calendar")}
          class={`px-3 py-1.5 text-xs font-semibold rounded-md transition-colors flex items-center gap-1.5 ${
            viewMode === "calendar" || viewMode === "grid"
              ? "bg-sky-500 text-white shadow-sm"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <span>📅</span>
          {m.appts_view_grid()}
        </button>
        <button
          type="button"
          onclick={() => (viewMode = "agenda")}
          class={`px-3 py-1.5 text-xs font-semibold rounded-md transition-colors flex items-center gap-1.5 ${
            viewMode === "agenda"
              ? "bg-sky-500 text-white shadow-sm"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <span>📋</span>
          {m.appts_view_agenda()}
        </button>
      </div>

      <!-- Sub-tab Slider for Calendar View (Day | Week | Month) -->
      {#if viewMode === "calendar" || viewMode === "grid"}
        <div class="flex rounded-lg border border-slate-700/80 bg-slate-900/60 p-0.5">
          <button
            type="button"
            onclick={() => (calendarView = "day")}
            class={`px-2.5 py-1 text-xs font-semibold rounded-md transition-all ${
              calendarView === "day"
                ? "bg-slate-700 text-sky-400 shadow border border-sky-500/30"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            {m.appts_view_day()}
          </button>
          <button
            type="button"
            onclick={() => (calendarView = "week")}
            class={`px-2.5 py-1 text-xs font-semibold rounded-md transition-all ${
              calendarView === "week"
                ? "bg-slate-700 text-sky-400 shadow border border-sky-500/30"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            {m.appts_view_week()}
          </button>
          <button
            type="button"
            onclick={() => (calendarView = "month")}
            class={`px-2.5 py-1 text-xs font-semibold rounded-md transition-all ${
              calendarView === "month"
                ? "bg-slate-700 text-sky-400 shadow border border-sky-500/30"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            {m.appts_view_month()}
          </button>
        </div>
      {/if}
    </div>

    <!-- Date Navigation -->
    {#if viewMode === "calendar" || viewMode === "grid"}
      <div class="flex items-center gap-2">
        {#if calendarView === "day"}
          <button
            type="button"
            onclick={() => changeDay(-1)}
            class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
          >
            ◀ {m.appts_prev_day()}
          </button>
        {:else if calendarView === "week"}
          <button
            type="button"
            onclick={() => changeWeek(-1)}
            class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
          >
            ◀ {m.appts_prev_week()}
          </button>
        {:else if calendarView === "month"}
          <button
            type="button"
            onclick={() => changeMonth(-1)}
            class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
          >
            ◀ {m.appts_prev_month()}
          </button>
        {/if}

        <button
          type="button"
          onclick={setToday}
          class="rounded-lg border border-sky-500/30 bg-sky-500/10 px-3 py-1.5 text-xs font-semibold text-sky-400 hover:bg-sky-500/20"
        >
          {m.appts_today()}
        </button>

        {#if calendarView === "day"}
          <button
            type="button"
            onclick={() => changeDay(1)}
            class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
          >
            {m.appts_next_day()} ▶
          </button>
        {:else if calendarView === "week"}
          <button
            type="button"
            onclick={() => changeWeek(1)}
            class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
          >
            {m.appts_next_week()} ▶
          </button>
        {:else if calendarView === "month"}
          <button
            type="button"
            onclick={() => changeMonth(1)}
            class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
          >
            {m.appts_next_month()} ▶
          </button>
        {/if}

        <input
          type="date"
          bind:value={selectedDate}
          class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1 text-sm text-slate-200 focus:border-sky-500 focus:outline-none"
        />
      </div>
    {:else}
      <div
        class="flex items-center gap-2 text-xs font-semibold text-sky-400 bg-sky-500/10 border border-sky-500/20 px-3 py-1.5 rounded-lg"
      >
        <span>📋</span>
        <span>{m.appts_agenda_header()}</span>
      </div>
    {/if}

    <!-- Provider Filter -->
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
  </div>

  <!-- View Title Header -->
  <div class="flex items-center justify-between border-b border-slate-700/60 pb-2">
    <h2 class="text-lg font-bold text-slate-100 flex items-center gap-2">
      {#if viewMode === "calendar" || viewMode === "grid"}
        <span>📅</span>
        {#if calendarView === "day"}
          {formattedDateHeading(selectedDate)}
        {:else if calendarView === "week"}
          Week of {weekRangeHeading}
        {:else if calendarView === "month"}
          {monthYearHeading}
        {/if}
      {:else}
        <span>📋</span>
        Agenda (All Dates)
      {/if}
    </h2>
    <span class="text-xs text-slate-400 font-medium">
      {filteredAppointments.length}
      {#if viewMode === "calendar" || viewMode === "grid"}
        appointment{filteredAppointments.length === 1 ? "" : "s"} scheduled
      {:else}
        total appointment{filteredAppointments.length === 1 ? "" : "s"}
      {/if}
    </span>
  </div>

  {#if loading}
    <div class="flex items-center justify-center py-16 text-slate-400">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-sky-400"></div>
      <span class="ml-3 text-sm font-medium">{m.appts_loading_schedule()}</span>
    </div>
  {:else if viewMode === "calendar" || viewMode === "grid"}
    <!-- CALENDAR VIEW -->
    {#if calendarView === "day"}
      <!-- DAY VIEW GRID -->
      <div class="rounded-xl border border-slate-700/80 bg-slate-900/80 shadow-md overflow-hidden">
        <div class="divide-y divide-slate-800">
          {#each timeSlots as slot}
            {@const slotAppts = filteredAppointments.filter(
              (a: Appointment) => getApptHour(a.start_time) === slot
            )}
            <div class="flex min-h-[96px] group hover:bg-slate-800/30 transition-colors">
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
                            <span
                              >⏱ {formatTime(appt.start_time)} - {formatTime(appt.end_time)}</span
                            >
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
    {:else if calendarView === "week"}
      <!-- WEEK VIEW GRID -->
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
            <!-- Day Header (Clickable to switch to Day view) -->
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
                >
                  Today
                </span>
              {/if}
            </button>

            <!-- Appointments List for Day -->
            <div class="p-2 space-y-2 flex-1 overflow-y-auto max-h-[650px] min-h-[160px]">
              {#if dayAppts.length === 0}
                <div
                  class="h-full flex items-center justify-center text-center text-xs text-slate-600 italic py-6"
                >
                  No appointments
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
                      ⏱ {formatTime(appt.start_time)}
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
                        >👤 {getProviderName(appt.provider_id)}</span
                      >
                      <span
                        class={`rounded-full border px-1.5 py-0.2 text-[9px] font-semibold uppercase ${badge.bg} ${badge.text} ${badge.border}`}
                      >
                        {badge.label}
                      </span>
                    </div>
                  </div>
                {/each}
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {:else if calendarView === "month"}
      <!-- MONTH VIEW GRID -->
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
              <!-- Cell Header / Day Number (Clickable to switch to Day view) -->
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
                    <span class="text-sky-400 text-[10px] font-semibold"
                      >{formatTime(appt.start_time)}</span
                    >
                    <span class="font-semibold text-white truncate"
                      >{getPatientName(appt.patient_id)}</span
                    >
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
    {/if}
  {:else}
    <!-- AGENDA VIEW TABLE (WITH CLICKABLE DATES) -->
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
              <th class="px-4 py-3">{m.appt_label_date()} & {m.appt_label_start_time()}</th>
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
                <!-- Clickable Date Badge to Jump to Calendar View -->
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
                      <span class="text-slate-400">📅</span>
                      {formatApptDate(appt.start_time)}
                    </div>
                    <div class="text-sky-400 text-xs mt-1 flex items-center gap-1.5">
                      <span class="text-slate-400">⏱</span>
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
