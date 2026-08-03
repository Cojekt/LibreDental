<script lang="ts">
  import type { Patient, CountryConfig } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import FilterBar from "../components/FilterBar.svelte";
  import PatientTable from "../components/PatientTable.svelte";

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
</script>

<div class="space-y-5">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-xl font-bold text-slate-100">Patient Directory</h2>
      <p class="text-xs text-slate-400 mt-0.5">Manage patient records, contact info, and medical histories</p>
    </div>
    <div class="text-xs font-semibold text-slate-300 bg-slate-800/90 px-3.5 py-1.5 rounded-xl border border-slate-700/80 shadow-sm">
      Total Patients: <span class="text-sky-400 font-bold ml-1">{patients.length}</span>
    </div>
  </div>

  <FilterBar
    bind:searchQuery
    bind:statusFilter
    {onloadpatients}
  />

  <PatientTable
    {patients}
    {loading}
    {statusFilter}
    {countryMeta}
    {onaddpatient}
    {oneditpatient}
    {onarchivepatient}
  />
</div>
