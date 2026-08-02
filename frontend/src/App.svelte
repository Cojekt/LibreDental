<script lang="ts">
  import { onMount } from "svelte";
  import { PatientService } from "../bindings/github.com/LibreDental/libredental/pkg/services/index.js";
  import type { Patient } from "../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";

  import Header from "./components/Header.svelte";
  import StatsGrid from "./components/StatsGrid.svelte";
  import FilterBar from "./components/FilterBar.svelte";
  import PatientTable from "./components/PatientTable.svelte";
  import PatientModal from "./components/PatientModal.svelte";

  let patients = $state<Patient[]>([]);
  let searchQuery = $state("");
  let statusFilter = $state("active");
  let loading = $state(false);
  let showPatientModal = $state(false);
  let isEditing = $state(false);
  let editingPatientId = $state("");

  // Form fields
  let firstName = $state("");
  let lastName = $state("");
  let email = $state("");
  let phone = $state("");
  let dob = $state("1990-01-01");
  let medicalAlerts = $state("");

  async function loadPatients() {
    loading = true;
    try {
      const res = await PatientService.ListPatients(searchQuery, statusFilter);
      patients = res || [];
    } catch (err) {
      console.error("Failed to load patients:", err);
    } finally {
      loading = false;
    }
  }

  function openAddModal() {
    isEditing = false;
    editingPatientId = "";
    firstName = "";
    lastName = "";
    email = "";
    phone = "";
    dob = "1990-01-01";
    medicalAlerts = "";
    showPatientModal = true;
  }

  function openEditModal(p: Patient) {
    isEditing = true;
    editingPatientId = p.id;
    firstName = p.first_name;
    lastName = p.last_name;
    email = p.email || "";
    phone = p.phone_primary || "";
    dob = p.date_of_birth
      ? new Date(p.date_of_birth).toISOString().split("T")[0]
      : "1990-01-01";
    medicalAlerts = p.medical_alerts ? p.medical_alerts.join(", ") : "";
    showPatientModal = true;
  }

  async function handleSavePatient(e: Event) {
    e.preventDefault();
    if (!firstName || !lastName) return;

    try {
      if (isEditing) {
        const p = await PatientService.GetPatient(editingPatientId);
        if (p) {
          p.first_name = firstName;
          p.last_name = lastName;
          p.email = email;
          p.phone_primary = phone;
          p.date_of_birth = new Date(dob).toISOString();
          p.medical_alerts = medicalAlerts
            ? medicalAlerts.split(",").map((s) => s.trim())
            : [];
          await PatientService.UpdatePatient(p);
        }
      } else {
        const newPatient: Patient = {
          id: "pat_" + Date.now(),
          first_name: firstName,
          last_name: lastName,
          email: email,
          phone_primary: phone,
          date_of_birth: new Date(dob).toISOString(),
          gender: "undisclosed",
          status: "active",
          medical_alerts: medicalAlerts
            ? medicalAlerts.split(",").map((s) => s.trim())
            : [],
          allergies: [],
          version: 1,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          middle_name: undefined,
          preferred_name: undefined,
          phone_secondary: undefined,
          address_line1: undefined,
          address_line2: undefined,
          city: undefined,
          state: undefined,
          zip_code: undefined,
          notes: undefined,
        };
        await PatientService.CreatePatient(newPatient);
      }
      showPatientModal = false;
      await loadPatients();
    } catch (err) {
      console.error("Failed to save patient:", err);
    }
  }

  async function handleArchivePatient(p: Patient) {
    if (
      confirm(
        `Are you sure you want to archive ${p.first_name} ${p.last_name}?`,
      )
    ) {
      try {
        await PatientService.ArchivePatient(p.id);
        await loadPatients();
      } catch (err) {
        console.error("Failed to archive patient:", err);
      }
    }
  }

  onMount(() => {
    loadPatients();
  });
</script>

<div class="min-h-screen flex flex-col">
  <Header onnewpatient={openAddModal} />

  <main class="p-6 max-w-[1200px] mx-auto w-full box-border">
    <StatsGrid patientCount={patients.length} />

    <FilterBar
      bind:searchQuery
      bind:statusFilter
      onloadpatients={loadPatients}
    />

    <PatientTable
      {patients}
      {loading}
      {statusFilter}
      onaddpatient={openAddModal}
      oneditpatient={openEditModal}
      onarchivepatient={handleArchivePatient}
    />
  </main>
</div>

<PatientModal
  bind:showPatientModal
  {isEditing}
  bind:firstName
  bind:lastName
  bind:email
  bind:phone
  bind:dob
  bind:medicalAlerts
  onsave={handleSavePatient}
/>


