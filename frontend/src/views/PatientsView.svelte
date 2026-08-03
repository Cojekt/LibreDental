<script lang="ts">
  import type { Patient, CountryConfig } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import StatsGrid from "../components/StatsGrid.svelte";
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

<div class="space-y-6">
  <StatsGrid patientCount={patients.length} />

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
