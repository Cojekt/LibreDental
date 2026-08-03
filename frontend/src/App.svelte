<script lang="ts">
  import { onMount } from "svelte";
  import { PatientService, ConfigService } from "../bindings/github.com/LibreDental/libredental/pkg/services/index.js";
  import type { Patient, PracticeConfig, CountryConfig } from "../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import { FALLBACK_COUNTRIES } from "./lib/country.js";

  import Header from "./components/Header.svelte";
  import StatsGrid from "./components/StatsGrid.svelte";
  import FilterBar from "./components/FilterBar.svelte";
  import PatientTable from "./components/PatientTable.svelte";
  import PatientModal from "./components/PatientModal.svelte";
  import OnboardingModal from "./components/OnboardingModal.svelte";

  let patients = $state<Patient[]>([]);
  let searchQuery = $state("");
  let statusFilter = $state("active");
  let loading = $state(false);

  // Configuration & Onboarding
  let practiceConfig = $state<PracticeConfig | null>(null);
  let countryMeta = $state<CountryConfig | null>(null);
  let supportedCountries = $state<CountryConfig[]>(FALLBACK_COUNTRIES);
  let showOnboarding = $state(false);

  // Modal states
  let showPatientModal = $state(false);
  let isEditing = $state(false);
  let editingPatientId = $state("");

  // Form fields
  let firstName = $state("");
  let lastName = $state("");
  let email = $state("");
  let phone = $state("");
  let dob = $state("1990-01-01");
  let nationalId = $state("");
  let stateProvince = $state("");
  let postalCode = $state("");
  let medicalAlerts = $state("");

  async function checkConfig() {
    try {
      const countries = await ConfigService.GetSupportedCountries();
      if (countries && countries.length > 0) {
        supportedCountries = countries;
      }
    } catch (e) {
      console.warn("Using fallback countries:", e);
    }

    try {
      const cfg = await ConfigService.GetConfig();
      if (!cfg || !cfg.country_code) {
        showOnboarding = true;
      } else {
        practiceConfig = cfg;
        await loadCountryMeta(cfg.country_code);
      }
    } catch (err) {
      console.error("Failed to check practice config:", err);
      showOnboarding = true;
    }
  }

  async function loadCountryMeta(countryCode: string) {
    try {
      const meta = await ConfigService.GetCountryConfig(countryCode);
      if (meta) {
        countryMeta = meta;
        return;
      }
    } catch (e) {
      console.warn("Could not fetch country meta from backend, using fallback:", e);
    }
    const found = supportedCountries.find((c) => c.code === countryCode);
    countryMeta = found || supportedCountries[0];
  }

  async function handleOnboardingComplete(countryCode: string) {
    try {
      const cfg = await ConfigService.SetConfig(countryCode);
      practiceConfig = cfg;
      await loadCountryMeta(countryCode);
      showOnboarding = false;
      await loadPatients();
    } catch (err) {
      console.error("Failed to save onboarding practice config:", err);
    }
  }

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
    nationalId = "";
    stateProvince = "";
    postalCode = "";
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
    nationalId = p.national_id || "";
    stateProvince = p.state_province || "";
    postalCode = p.postal_code || "";
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
          p.national_id = nationalId;
          p.national_id_type = countryMeta?.national_id_type || "national_id";
          p.state_province = stateProvince;
          p.postal_code = postalCode;
          p.country_code = countryMeta?.code || "US";
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
          gender: Gender.GenderUndisclosed,
          status: Status.StatusActive,
          national_id: nationalId,
          national_id_type: countryMeta?.national_id_type || "national_id",
          state_province: stateProvince,
          postal_code: postalCode,
          country_code: countryMeta?.code || "US",
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

  onMount(async () => {
    await checkConfig();
    await loadPatients();
  });
</script>

<div class="min-h-screen flex flex-col">
  <Header {countryMeta} onnewpatient={openAddModal} />

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
      {countryMeta}
      onaddpatient={openAddModal}
      oneditpatient={openEditModal}
      onarchivepatient={handleArchivePatient}
    />
  </main>
</div>

<OnboardingModal
  bind:showOnboarding
  {supportedCountries}
  oncomplete={handleOnboardingComplete}
/>

<PatientModal
  bind:showPatientModal
  {isEditing}
  bind:firstName
  bind:lastName
  bind:email
  bind:phone
  bind:dob
  bind:nationalId
  bind:stateProvince
  bind:postalCode
  bind:medicalAlerts
  {countryMeta}
  onsave={handleSavePatient}
/>
