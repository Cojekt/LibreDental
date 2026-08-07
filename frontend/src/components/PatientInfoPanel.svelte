<script lang="ts">
  import type {
    Patient,
    CountryConfig,
  } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../lib/locale.svelte.js";

  let {
    patient = null,
    countryMeta = null,
    oneditpatient,
    onarchivepatient,
    onclearselection,
  } = $props<{
    patient: Patient | null;
    countryMeta?: CountryConfig | null;
    oneditpatient?: (p: Patient) => void;
    onarchivepatient?: (p: Patient) => void;
    onclearselection?: () => void;
  }>();

  let showSSN = $state(false);

  // Reset SSN unhide state when switching selected patient
  $effect(() => {
    if (patient) {
      showSSN = false;
    }
  });

  const idLabel = $derived.by(() => {
    getLocaleVersion();
    return countryMeta?.national_id_name || m.patient_info_ssn_default();
  });

  const maskedSSN = $derived.by(() => {
    getLocaleVersion();
    if (!patient?.national_id) return m.patients_unassigned_id();
    const raw = patient.national_id;
    if (raw.length === 9 && !raw.includes("-")) {
      return "•••-••-••••";
    }
    return raw.replace(/[a-zA-Z0-9]/g, "•");
  });

  function calculateAge(dobStr?: string): number | null {
    if (!dobStr) return null;
    const dob = new Date(dobStr);
    if (isNaN(dob.getTime())) return null;
    const diffMs = Date.now() - dob.getTime();
    const ageDate = new Date(diffMs);
    return Math.abs(ageDate.getUTCFullYear() - 1970);
  }

  const age = $derived(patient?.date_of_birth ? calculateAge(patient.date_of_birth) : null);

  function formatSex(sex?: string): string {
    getLocaleVersion();
    if (!sex) return "N/A";
    switch (sex) {
      case "male":
        return m.sex_male();
      case "female":
        return m.sex_female();
      case "other":
        return m.sex_other();
      default:
        return m.sex_undisclosed();
    }
  }

  const formattedAddress = $derived.by(() => {
    if (!patient) return "";
    const parts = [
      patient.address_line1,
      patient.address_line2,
      patient.city,
      patient.state_province,
      patient.postal_code,
      patient.country_code,
    ].filter(Boolean);
    return parts.join(", ");
  });
</script>

<div
  class="flex flex-col h-full rounded-xl border border-slate-700 bg-slate-800 shadow-xl overflow-hidden"
>
  <!-- Header -->
  <div class="flex items-center justify-between border-b border-slate-700 bg-slate-900 px-5 py-4">
    <div class="flex items-center gap-2.5">
      <div
        class="flex h-8 w-8 items-center justify-center rounded-lg bg-sky-500/15 text-sky-400 border border-sky-500/20"
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="h-4 w-4"
        >
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
          <circle cx="12" cy="7" r="4"></circle>
        </svg>
      </div>
      <div>
        <h3 class="text-base font-bold text-slate-100">
          {(getLocaleVersion(), m.patient_info_title())}
        </h3>
        {#if patient}
          <p class="text-xs text-sky-400 font-medium truncate max-w-[200px]">
            {patient.first_name}
            {patient.last_name}
          </p>
        {/if}
      </div>
    </div>

    {#if patient && onclearselection}
      <button
        type="button"
        onclick={onclearselection}
        class="text-slate-400 hover:text-slate-200 p-1.5 rounded-lg hover:bg-slate-800 transition-colors cursor-pointer"
        title={m.patient_info_deselect()}
        aria-label={m.patient_info_deselect()}
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="h-4 w-4"
        >
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      </button>
    {/if}
  </div>

  <!-- Body -->
  <div class="p-5 flex-1 overflow-y-auto">
    {#if !patient}
      <!-- Empty Placeholder State -->
      <div class="flex flex-col items-center justify-center py-12 text-center h-full">
        <div
          class="mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-900/80 border border-slate-700/80 text-slate-500 shadow-inner"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="h-7 w-7"
          >
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path>
            <circle cx="9" cy="7" r="4"></circle>
            <path d="M22 21v-2a4 4 0 0 0-3-3.87"></path>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
          </svg>
        </div>
        <p class="text-sm font-semibold text-slate-300">
          {(getLocaleVersion(), m.patient_info_no_selected_title())}
        </p>
        <p class="mt-1 text-xs text-slate-400 max-w-[220px]">
          {m.patient_info_no_selected_desc()}
        </p>
      </div>
    {:else}
      <!-- Full Patient Info View -->
      <div class="space-y-5">
        <!-- Patient Header Badge / Name -->
        <div class="flex items-start justify-between border-b border-slate-700/70 pb-4">
          <div>
            <h4 class="text-lg font-bold text-slate-50">
              {patient.first_name}
              {patient.last_name}
            </h4>
            {#if patient.preferred_name}
              <p class="text-xs text-slate-400 italic">"{patient.preferred_name}"</p>
            {/if}
            <div class="mt-1.5 flex items-center gap-2 flex-wrap">
              {#if patient.status === "archived"}
                <span
                  class="rounded bg-rose-500/20 px-2 py-0.5 text-[11px] font-semibold text-rose-400 border border-rose-500/30"
                >
                  {(getLocaleVersion(), m.patients_filter_archived())}
                </span>
              {:else}
                <span
                  class="rounded bg-emerald-500/20 px-2 py-0.5 text-[11px] font-semibold text-emerald-400 border border-emerald-500/30"
                >
                  {(getLocaleVersion(), m.patient_info_active())}
                </span>
              {/if}
              {#if age !== null}
                <span class="text-xs text-slate-400">{m.patient_info_age({ age })}</span>
              {/if}
              <span
                class="rounded bg-sky-500/15 px-2 py-0.5 text-[11px] font-medium text-sky-300 border border-sky-500/20"
              >
                {formatSex(patient.sex)}
              </span>
            </div>
          </div>
        </div>

        <!-- National ID / SSN Section (with Eye icon toggle) -->
        <div class="rounded-xl border border-slate-700/80 bg-slate-900/60 p-3.5 shadow-sm">
          <div class="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span class="font-medium">{idLabel}</span>
            <button
              type="button"
              onclick={() => (showSSN = !showSSN)}
              class="flex items-center gap-1 text-sky-400 hover:text-sky-300 text-xs font-semibold px-2 py-0.5 rounded hover:bg-slate-800 transition-colors cursor-pointer"
              title={showSSN ? m.patient_info_hide_ssn() : m.patient_info_unhide_ssn()}
              aria-label={showSSN ? m.patient_info_hide_ssn() : m.patient_info_unhide_ssn()}
            >
              {#if showSSN}
                <!-- Eye Off Icon -->
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  class="h-3.5 w-3.5"
                >
                  <path
                    d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"
                  ></path>
                  <line x1="1" y1="1" x2="23" y2="23"></line>
                </svg>
                <span>{m.patient_info_hide()}</span>
              {:else}
                <!-- Eye Icon -->
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  class="h-3.5 w-3.5"
                >
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                  <circle cx="12" cy="12" r="3"></circle>
                </svg>
                <span>{m.patient_info_unhide()}</span>
              {/if}
            </button>
          </div>
          <div
            class="font-mono text-sm font-semibold tracking-wide text-sky-400 bg-sky-500/10 px-3 py-1.5 rounded border border-blue-500/20 flex items-center justify-between"
          >
            <span>{showSSN ? patient.national_id || m.patients_unassigned_id() : maskedSSN}</span>
          </div>
        </div>

        <!-- Contact Information -->
        <div class="space-y-2 text-xs">
          <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
            {m.patient_info_contact_title()}
          </p>
          <div class="rounded-lg border border-slate-700/60 bg-slate-900/40 p-3 space-y-2">
            <div class="flex items-center gap-2 text-slate-200">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="h-3.5 w-3.5 text-slate-400 shrink-0"
              >
                <path
                  d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"
                ></path>
              </svg>
              <span class="font-mono text-slate-100"
                >{patient.phone_primary || m.patients_no_phone()}</span
              >
            </div>
            {#if patient.phone_secondary}
              <div class="flex items-center gap-2 text-slate-300 pl-5 text-[11px]">
                <span>{m.patient_info_sec_phone()}</span>
                <span class="font-mono text-slate-300">{patient.phone_secondary}</span>
              </div>
            {/if}
            <div class="flex items-center gap-2 text-slate-200">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="h-3.5 w-3.5 text-slate-400 shrink-0"
              >
                <path
                  d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"
                ></path>
                <polyline points="22,6 12,13 2,6"></polyline>
              </svg>
              <span class="truncate text-slate-100">{patient.email || m.patients_no_email()}</span>
            </div>
          </div>
        </div>

        <!-- Demographics & Location -->
        <div class="space-y-2 text-xs">
          <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
            {m.patient_info_demographics_title()}
          </p>
          <div
            class="rounded-lg border border-slate-700/60 bg-slate-900/40 p-3 space-y-2 text-slate-300"
          >
            <div class="flex justify-between">
              <span class="text-slate-400">{m.patient_sex()}:</span>
              <span class="font-medium text-slate-100">{formatSex(patient.sex)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-slate-400">{m.patient_dob()}:</span>
              <span class="font-medium text-slate-100">
                {patient.date_of_birth
                  ? new Date(patient.date_of_birth).toLocaleDateString()
                  : "N/A"}
              </span>
            </div>
            {#if formattedAddress}
              <div class="flex justify-between">
                <span class="text-slate-400 shrink-0 mr-2">{m.patient_info_region_address()}</span>
                <span class="font-medium text-slate-100 text-right">
                  {formattedAddress}
                </span>
              </div>
            {/if}
          </div>
        </div>

        <!-- Emergency Contact -->
        {#if patient.emergency_contact_name || patient.emergency_contact_phone}
          <div class="space-y-2 text-xs">
            <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
              🆘 {m.patient_info_emergency_title()}
            </p>
            <div
              class="rounded-lg border border-amber-500/20 bg-amber-500/5 p-3 space-y-1 text-slate-300"
            >
              <div class="font-semibold text-amber-300 flex items-center justify-between">
                <span>{patient.emergency_contact_name}</span>
                {#if patient.emergency_contact_rel}
                  <span class="text-[11px] font-normal text-amber-400/80"
                    >({patient.emergency_contact_rel})</span
                  >
                {/if}
              </div>
              {#if patient.emergency_contact_phone}
                <p class="font-mono text-xs text-slate-200">{patient.emergency_contact_phone}</p>
              {/if}
            </div>
          </div>
        {/if}

        <!-- Primary Insurance -->
        {#if patient.insurance_carrier || patient.insurance_policy_number}
          <div class="space-y-2 text-xs">
            <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
              🛡️ {m.patient_info_insurance_title()}
            </p>
            <div
              class="rounded-lg border border-slate-700/60 bg-slate-900/40 p-3 space-y-1.5 text-slate-300"
            >
              <p class="font-semibold text-slate-100">
                {patient.insurance_carrier || "Primary Dental Insurance"}
              </p>
              {#if patient.insurance_policy_number}
                <div class="flex justify-between text-[11px]">
                  <span class="text-slate-400">Policy / ID:</span>
                  <span class="font-mono text-sky-400">{patient.insurance_policy_number}</span>
                </div>
              {/if}
              {#if patient.insurance_group_number}
                <div class="flex justify-between text-[11px]">
                  <span class="text-slate-400">Group #:</span>
                  <span class="font-mono text-slate-200">{patient.insurance_group_number}</span>
                </div>
              {/if}
            </div>
          </div>
        {/if}

        <!-- Guarantor -->
        {#if patient.guarantor_name}
          <div class="space-y-2 text-xs">
            <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
              💳 {m.patient_info_guarantor_title()}
            </p>
            <div
              class="rounded-lg border border-slate-700/60 bg-slate-900/40 p-3 space-y-1 text-slate-300"
            >
              <div class="flex justify-between font-semibold text-slate-100">
                <span>{patient.guarantor_name}</span>
                {#if patient.guarantor_rel}
                  <span class="text-[11px] font-normal text-slate-400"
                    >({patient.guarantor_rel})</span
                  >
                {/if}
              </div>
              {#if patient.guarantor_phone}
                <p class="font-mono text-xs text-slate-300">{patient.guarantor_phone}</p>
              {/if}
            </div>
          </div>
        {/if}

        <!-- Preferences & Reminders -->
        <div class="space-y-2 text-xs">
          <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
            ⚙️ {m.patient_info_preferences_title()}
          </p>
          <div
            class="rounded-lg border border-slate-700/60 bg-slate-900/40 p-3 space-y-1.5 text-slate-300"
          >
            <div class="flex justify-between items-center">
              <span class="text-slate-400">Contact Preference:</span>
              <span class="font-semibold text-sky-400 capitalize"
                >{patient.preferred_contact_method || "phone"}</span
              >
            </div>
            <div class="flex justify-between items-center">
              <span class="text-slate-400">Reminders:</span>
              {#if patient.reminder_opt_in !== false}
                <span
                  class="text-[11px] font-medium text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20"
                >
                  ✓ {m.patient_info_reminders_enabled()}
                </span>
              {:else}
                <span
                  class="text-[11px] font-medium text-slate-400 bg-slate-800 px-2 py-0.5 rounded"
                >
                  ✕ {m.patient_info_reminders_disabled()}
                </span>
              {/if}
            </div>
            {#if patient.referral_source}
              <div class="flex justify-between items-center text-[11px]">
                <span class="text-slate-400">Referral:</span>
                <span class="text-slate-200">{patient.referral_source}</span>
              </div>
            {/if}
          </div>
        </div>

        <!-- Medical Alerts -->
        <div class="space-y-2 text-xs">
          <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
            {m.patients_th_alerts()}
          </p>
          {#if patient.medical_alerts && patient.medical_alerts.length > 0}
            <div class="flex flex-wrap gap-1.5">
              {#each patient.medical_alerts as alert}
                <span
                  class="rounded-md border border-amber-400/30 bg-amber-500/15 px-2.5 py-1 text-xs font-medium text-amber-400 flex items-center gap-1"
                >
                  ⚠️ {alert}
                </span>
              {/each}
            </div>
          {:else}
            <span
              class="inline-block rounded-md bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-400"
            >
              {m.patients_clean_record()}
            </span>
          {/if}
        </div>

        <!-- Actions -->
        <div class="pt-3 border-t border-slate-700/70 flex gap-2">
          {#if oneditpatient}
            <button
              class="btn btn-secondary btn-sm flex-1 justify-center cursor-pointer"
              onclick={() => oneditpatient(patient)}
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="h-3.5 w-3.5"
              >
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
              </svg>
              {m.patients_btn_edit()}
            </button>
          {/if}
          {#if onarchivepatient && patient.status !== "archived"}
            <button
              class="btn btn-ghost btn-danger btn-sm flex-1 justify-center cursor-pointer"
              onclick={() => onarchivepatient(patient)}
            >
              {m.patients_btn_archive()}
            </button>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>
