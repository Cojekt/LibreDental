<script lang="ts">
  import type {
    Patient,
    CountryConfig,
  } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import FilterBar from "../components/FilterBar.svelte";
  import PatientTable from "../components/PatientTable.svelte";
  import PatientInfoPanel from "../components/PatientInfoPanel.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../lib/locale.svelte.js";

  let {
    patients,
    loading,
    searchQuery = $bindable(""),
    statusFilter = $bindable("active"),
    countryMeta,
    onloadpatients,
    onaddpatient,
    oneditpatient,
    onarchivepatient,
  } = $props<{
    patients: Patient[];
    loading: boolean;
    searchQuery: string;
    statusFilter: string;
    countryMeta?: CountryConfig | null;
    onloadpatients: () => void;
    onaddpatient: () => void;
    oneditpatient: (p: Patient) => void;
    onarchivepatient: (p: Patient) => void;
  }>();

  let selectedPatient = $state<Patient | null>(null);

  // Synchronize selected patient object when patients array changes
  $effect(() => {
    if (selectedPatient) {
      const current = patients.find((p: Patient) => p.id === selectedPatient?.id);
      selectedPatient = current || null;
    }
  });

  function handleSelectPatient(p: Patient) {
    selectedPatient = p;
  }
</script>

<div class="space-y-5">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-xl font-bold text-slate-100">
        {(getLocaleVersion(), m.patients_directory_title())}
      </h2>
      <p class="text-xs text-slate-400 mt-0.5">
        {m.patients_directory_subtitle()}
      </p>
    </div>
    <div
      class="text-xs font-semibold text-slate-300 bg-slate-800/90 px-3.5 py-1.5 rounded-xl border border-slate-700/80 shadow-sm"
    >
      {m.patients_total_count()}
      <span class="text-sky-400 font-bold ml-1">{patients.length}</span>
    </div>
  </div>

  <FilterBar bind:searchQuery bind:statusFilter {onloadpatients} />

  <div class="grid grid-cols-1 lg:grid-cols-3 xl:grid-cols-4 gap-6 items-start">
    <div class="lg:col-span-2 xl:col-span-3">
      <PatientTable
        {patients}
        {loading}
        {statusFilter}
        {countryMeta}
        selectedPatientId={selectedPatient?.id}
        onselectpatient={handleSelectPatient}
        {onaddpatient}
        {oneditpatient}
        {onarchivepatient}
      />
    </div>

    <div class="lg:col-span-1 xl:col-span-1">
      <PatientInfoPanel
        patient={selectedPatient}
        {countryMeta}
        {oneditpatient}
        {onarchivepatient}
        onclearselection={() => (selectedPatient = null)}
      />
    </div>
  </div>
</div>
