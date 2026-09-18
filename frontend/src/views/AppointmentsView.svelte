<script lang="ts">
  import type {
    Patient,
    Appointment,
    Provider,
    Operatory,
    BusinessHourDay,
    TimeSlot,
  } from "@bindings/domain/models.js";
  import AppointmentStats from "../components/AppointmentStats.svelte";
  import AppointmentDaySection from "./appointments/AppointmentDaySection.svelte";
  import AppointmentWeekSection from "./appointments/AppointmentWeekSection.svelte";
  import AppointmentMonthSection from "./appointments/AppointmentMonthSection.svelte";
  import AppointmentAgendaSection from "./appointments/AppointmentAgendaSection.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocalDateString, isSameDay, parseLocalDate } from "$lib/date.js";
  import { currentLocale, getLocaleVersion } from "$lib/locale.svelte.js";

  let {
    appointments = [],
    patients = [],
    providers = [],
    operatories = [],
    businessHours = null,
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
    businessHours?: BusinessHourDay[] | null;
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

  // Appointment filters, shared by calendar and agenda views. The date range
  // only applies to agenda (calendar navigation already picks the date), and
  // defaults to "today and after" so agenda opens on upcoming appointments
  // rather than the full history.
  let filterPatient = $state("all");
  let filterOperatory = $state("all");
  let filterStatus = $state("all");
  let filterDateFrom = $state(getLocalDateString());
  let filterDateTo = $state("");

  function clearFilters() {
    selectedProvider = "all";
    filterPatient = "all";
    filterOperatory = "all";
    filterStatus = "all";
    filterDateFrom = getLocalDateString();
    filterDateTo = "";
  }

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
        bg: "bg-slate-500/15",
        text: "text-slate-300",
        border: "border-slate-500/30",
      },
      confirmed: {
        label: m.appts_status_confirmed(),
        bg: "bg-blue-500/15",
        text: "text-blue-400",
        border: "border-blue-500/30",
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
        bg: "bg-cyan-500/15",
        text: "text-cyan-400",
        border: "border-cyan-500/30",
      },
    });

  const statusColors: Record<string, string> = {
    scheduled: "#94a3b8",
    confirmed: "#3b82f6",
    arrived: "#f59e0b",
    in_chair: "#a855f7",
    completed: "#10b981",
    cancelled: "#f43f5e",
    no_show: "#06b6d4",
  };

  function getStatusColor(status: string): string {
    return statusColors[status] || statusColors.scheduled;
  }

  let timeSlots = $derived.by(() => {
    const slots: string[] = [];
    for (let h = 0; h < 24; h++) {
      slots.push(`${String(h).padStart(2, "0")}:00`);
    }
    return slots;
  });

  const WEEKDAY_NAMES = [
    "Sunday",
    "Monday",
    "Tuesday",
    "Wednesday",
    "Thursday",
    "Friday",
    "Saturday",
  ];

  function parseTimeToMinutes(time: string): number {
    if (!time) return 0;
    const [hStr, mStr] = time.split(":");
    const h = parseInt(hStr, 10) || 0;
    const m = parseInt(mStr, 10) || 0;
    return h * 60 + m;
  }

  let selectedDayBusinessHours = $derived.by(() => {
    if (!businessHours || businessHours.length === 0) return null;
    const d = parseLocalDate(selectedDate);
    if (isNaN(d.getTime())) return null;
    const dayName = WEEKDAY_NAMES[d.getDay()];
    return businessHours.find((bh: BusinessHourDay) => bh.day === dayName) || null;
  });

  // Ranges of the day (in minutes since midnight) the practice is open, with breaks removed.
  // `null` means "no business-hours data configured" - callers should not grey anything out.
  let workingIntervals = $derived.by(() => {
    const day = selectedDayBusinessHours;
    if (!day) return null;
    if (day.is_closed) return [] as [number, number][];

    let intervals: [number, number][] =
      day.slots && day.slots.length > 0
        ? day.slots.map((s: TimeSlot): [number, number] => [
            parseTimeToMinutes(s.open_time),
            parseTimeToMinutes(s.close_time),
          ])
        : [[parseTimeToMinutes(day.open_time), parseTimeToMinutes(day.close_time)]];

    for (const brk of day.breaks || []) {
      const bStart = parseTimeToMinutes(brk.start_time);
      const bEnd = parseTimeToMinutes(brk.end_time);
      const next: [number, number][] = [];
      for (const [s, e] of intervals) {
        if (bEnd <= s || bStart >= e) {
          next.push([s, e]);
          continue;
        }
        if (bStart > s) next.push([s, bStart]);
        if (bEnd < e) next.push([bEnd, e]);
      }
      intervals = next;
    }

    return intervals;
  });

  let workdayStartMinute = $derived(
    workingIntervals && workingIntervals.length > 0
      ? Math.min(...workingIntervals.map(([s]) => s))
      : 8 * 60
  );

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
    const day = curr.getDate();
    curr.setDate(1); // avoid rolling into the wrong month while day is still out of range
    curr.setMonth(curr.getMonth() + delta);
    const lastDayOfTargetMonth = new Date(curr.getFullYear(), curr.getMonth() + 1, 0).getDate();
    curr.setDate(Math.min(day, lastDayOfTargetMonth));
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
      const dayName = d.toLocaleDateString(currentLocale(), { weekday: "short" });
      const label = d.toLocaleDateString(currentLocale(), { month: "short", day: "numeric" });

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
      return d.toLocaleDateString(currentLocale(), { month: "long", year: "numeric" });
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
        if (filterPatient !== "all" && a.patient_id !== filterPatient) return false;
        if (filterOperatory !== "all" && a.operatory_id !== filterOperatory) return false;
        if (filterStatus !== "all" && a.status !== filterStatus) return false;
        const isCalendar = viewMode === "calendar" || viewMode === "grid";
        if (isCalendar && calendarView === "day") {
          return isSameDay(a.start_time, selectedDate);
        }
        if (viewMode === "agenda") {
          const apptDateStr = getLocalDateString(a.start_time);
          if (filterDateFrom && apptDateStr < filterDateFrom) return false;
          if (filterDateTo && apptDateStr > filterDateTo) return false;
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

  function formattedDateHeading(dateStr: string): string {
    try {
      const d = parseLocalDate(dateStr);
      return d.toLocaleDateString(currentLocale(), {
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
      return d.toLocaleDateString(currentLocale(), {
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

<div class="space-y-6" data-locale={getLocaleVersion()}>
  <!-- Control Bar -->
  <div
    class="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-slate-700/80 bg-slate-800/80 p-4 shadow-sm backdrop-blur"
  >
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
          {m.appts_view_agenda()}
        </button>
      </div>

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
    {/if}
  </div>

  <!-- Appointment Filters -->
  <div
    class="flex flex-wrap items-end gap-4 rounded-xl border border-slate-700/80 bg-slate-800/80 p-4 shadow-sm backdrop-blur"
  >
    <div class="flex flex-col gap-1">
      <label for="appt-filter-patient" class="text-xs font-medium text-slate-400">
        {m.appts_filter_patient()}
      </label>
      <select
        id="appt-filter-patient"
        bind:value={filterPatient}
        class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-200 focus:border-sky-500 focus:outline-none"
      >
        <option value="all">{m.appts_filter_all_patients()}</option>
        {#each patients as p}
          <option value={p.id}>{p.first_name} {p.last_name}</option>
        {/each}
      </select>
    </div>

    <div class="flex flex-col gap-1">
      <label for="appt-filter-provider" class="text-xs font-medium text-slate-400">
        {m.appts_provider_filter()}
      </label>
      <select
        id="appt-filter-provider"
        bind:value={selectedProvider}
        class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-200 focus:border-sky-500 focus:outline-none"
      >
        <option value="all">{m.appts_all_providers()}</option>
        {#each providers as p}
          <option value={p.id}>{p.name} ({p.role})</option>
        {/each}
      </select>
    </div>

    <div class="flex flex-col gap-1">
      <label for="appt-filter-operatory" class="text-xs font-medium text-slate-400">
        {m.appts_filter_operatory()}
      </label>
      <select
        id="appt-filter-operatory"
        bind:value={filterOperatory}
        class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-200 focus:border-sky-500 focus:outline-none"
      >
        <option value="all">{m.appts_filter_all_operatories()}</option>
        {#each operatories as o}
          <option value={o.id}>{o.name}</option>
        {/each}
      </select>
    </div>

    <div class="flex flex-col gap-1">
      <label for="appt-filter-status" class="text-xs font-medium text-slate-400">
        {m.appts_filter_status()}
      </label>
      <select
        id="appt-filter-status"
        bind:value={filterStatus}
        class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-200 focus:border-sky-500 focus:outline-none"
      >
        <option value="all">{m.appts_filter_all_statuses()}</option>
        <option value="scheduled">{m.appts_status_scheduled()}</option>
        <option value="confirmed">{m.appts_status_confirmed()}</option>
        <option value="arrived">{m.appts_status_arrived()}</option>
        <option value="in_chair">{m.appts_status_in_chair()}</option>
        <option value="completed">{m.appts_status_completed()}</option>
        <option value="cancelled">{m.appts_status_cancelled()}</option>
        <option value="no_show">{m.appts_status_no_show()}</option>
      </select>
    </div>

    {#if viewMode === "agenda"}
      <div class="flex flex-col gap-1">
        <label for="appt-filter-date-from" class="text-xs font-medium text-slate-400">
          {m.appts_filter_date_from()}
        </label>
        <input
          id="appt-filter-date-from"
          type="date"
          bind:value={filterDateFrom}
          class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-slate-200 focus:border-sky-500 focus:outline-none"
        />
      </div>

      <div class="flex flex-col gap-1">
        <label for="appt-filter-date-to" class="text-xs font-medium text-slate-400">
          {m.appts_filter_date_to()}
        </label>
        <input
          id="appt-filter-date-to"
          type="date"
          bind:value={filterDateTo}
          class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-slate-200 focus:border-sky-500 focus:outline-none"
        />
      </div>
    {/if}

    <button
      type="button"
      onclick={clearFilters}
      class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-300 hover:border-slate-600 hover:text-white"
    >
      {m.appts_filter_clear()}
    </button>
  </div>

  {#if viewMode === "calendar" || viewMode === "grid"}
    <!-- Stats Summary -->
    <AppointmentStats appointments={filteredAppointments} compact />
  {/if}

  <!-- Header Title -->
  <div class="flex items-center justify-between border-b border-slate-700/60 pb-2">
    <h2 class="text-lg font-bold text-slate-100 flex items-center gap-2">
      {#if viewMode === "calendar" || viewMode === "grid"}
        {#if calendarView === "day"}
          {formattedDateHeading(selectedDate)}
        {:else if calendarView === "week"}
          Week of {weekRangeHeading}
        {:else if calendarView === "month"}
          {monthYearHeading}
        {/if}
      {:else}
        {m.appts_agenda_title()}
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
    {#if calendarView === "day"}
      <AppointmentDaySection
        {selectedDate}
        {timeSlots}
        {workingIntervals}
        {workdayStartMinute}
        {filteredAppointments}
        {formatSlotLabel}
        {formatTime}
        {getPatientName}
        {getPatientPhone}
        {getProviderName}
        {getOperatoryName}
        {statusBadges}
        {getStatusColor}
        {oneditappointment}
        {onupdatestatus}
        confirmLabel={m.appts_action_confirm()}
        arrivedLabel={m.appts_action_arrived()}
        seatLabel={m.appts_action_seat()}
        completeLabel={m.appts_action_complete()}
        cancelLabel={m.appts_action_cancel()}
        noShowLabel={m.appts_action_no_show()}
      />
    {:else if calendarView === "week"}
      <AppointmentWeekSection
        {weekDays}
        {filteredAppointments}
        {isSameDay}
        {selectDateAndJumpToDay}
        {oneditappointment}
        {formatTime}
        {getPatientName}
        {getProviderName}
        {statusBadges}
        {getStatusColor}
      />
    {:else if calendarView === "month"}
      <AppointmentMonthSection
        {monthGrid}
        {filteredAppointments}
        {isSameDay}
        {selectDateAndJumpToDay}
        {oneditappointment}
        {formatTime}
        {getPatientName}
        {getStatusColor}
      />
    {/if}
  {:else}
    <AppointmentAgendaSection
      {filteredAppointments}
      {jumpToDateFromAgenda}
      {formatApptDate}
      {formatTime}
      {getPatientName}
      {getProviderName}
      {getOperatoryName}
      {onupdatestatus}
      {oneditappointment}
      {ondeleteappointment}
    />
  {/if}
</div>
