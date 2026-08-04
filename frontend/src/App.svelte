<script lang="ts">
  import { onMount } from "svelte";
  import {
    PatientService,
    PracticeConfigService,
    AppointmentService,
    SystemSettingsService,
  } from "../bindings/github.com/LibreDental/libredental/pkg/services/index.js";
  import type {
    Patient,
    PracticeConfig,
    CountryConfig,
    Appointment,
    Provider,
    Operatory,
  } from "../bindings/github.com/LibreDental/libredental/pkg/domain/index.js";
  import {
    Gender,
    Status,
    AppointmentStatus,
  } from "../bindings/github.com/LibreDental/libredental/pkg/domain/index.js";

  import Header from "./components/Header.svelte";
  import OnboardingModal from "./components/OnboardingModal.svelte";
  import PatientModal from "./components/PatientModal.svelte";
  import AppointmentModal from "./components/AppointmentModal.svelte";
  import SettingsModal from "./components/SettingsModal.svelte";
  import ClinicView from "./views/ClinicView.svelte";
  import PatientsView from "./views/PatientsView.svelte";
  import AppointmentsView from "./views/AppointmentsView.svelte";

  // App Navigation (Default to "clinic" landing tab on far left)
  let activeTab = $state("clinic");

  // Settings & Theme
  type ThemeMode = "dark" | "light" | "system";
  let showSettingsModal = $state(false);
  let theme = $state<ThemeMode>("system");

  async function getSystemOSTheme(): Promise<"dark" | "light"> {
    try {
      const isDark = await SystemSettingsService.IsSystemDarkMode();
      if (typeof isDark === "boolean") {
        return isDark ? "dark" : "light";
      }
    } catch (e) {
      console.warn("Could not query OS dark mode from backend:", e);
    }
    if (
      typeof window !== "undefined" &&
      window.matchMedia &&
      window.matchMedia("(prefers-color-scheme: dark)").matches
    ) {
      return "dark";
    }
    return "dark";
  }

  async function applyTheme(newTheme: ThemeMode) {
    theme = newTheme;
    localStorage.setItem("theme", newTheme);

    const effective = newTheme === "system" ? await getSystemOSTheme() : newTheme;
    if (effective === "light") {
      document.documentElement.classList.add("light");
    } else {
      document.documentElement.classList.remove("light");
    }

    try {
      await SystemSettingsService.SetTheme(newTheme);
    } catch (e) {
      console.warn("Failed to persist theme in SQLite:", e);
    }
  }

  async function loadTheme() {
    try {
      const dbTheme = await SystemSettingsService.GetTheme();
      if (
        dbTheme === "light" ||
        dbTheme === "dark" ||
        dbTheme === "system"
      ) {
        await applyTheme(dbTheme as ThemeMode);
        return;
      }
    } catch (e) {
      console.warn("Could not load theme from DB, fallback to localStorage:", e);
    }
    const savedTheme =
      (localStorage.getItem("theme") as ThemeMode) || "system";
    await applyTheme(savedTheme);
  }

  // Clinic providers and operatories state
  let providers = $state<Provider[]>([]);
  let operatories = $state<Operatory[]>([]);

  // Patients state
  let patients = $state<Patient[]>([]);
  let searchQuery = $state("");
  let statusFilter = $state("active");
  let loadingPatients = $state(false);

  // Configuration & Onboarding
  let practiceConfig = $state<PracticeConfig | null>(null);
  let countryMeta = $state<CountryConfig | null>(null);
  let supportedCountries = $state<CountryConfig[]>([]);
  let showOnboarding = $state(false);

  // Patient Modal states
  let showPatientModal = $state(false);
  let isEditingPatient = $state(false);
  let editingPatientId = $state("");

  // Patient form fields
  let firstName = $state("");
  let lastName = $state("");
  let email = $state("");
  let phone = $state("");
  let dob = $state("1990-01-01");
  let nationalId = $state("");
  let stateProvince = $state("");
  let postalCode = $state("");
  let medicalAlerts = $state("");

  // Appointments state
  let appointments = $state<Appointment[]>([]);
  let loadingAppointments = $state(false);
  let selectedDate = $state(new Date().toISOString().split("T")[0]);
  let selectedProvider = $state("all");
  let viewMode = $state<"grid" | "agenda">("grid");

  // Appointment Modal states
  let showApptModal = $state(false);
  let isEditingAppt = $state(false);
  let editingApptId = $state("");

  // Appointment form fields
  let apptPatientId = $state("");
  let apptProviderId = $state("");
  let apptOperatoryId = $state("");
  let apptStartDateStr = $state(new Date().toISOString().split("T")[0]);
  let apptStartTimeStr = $state("09:00");
  let apptEndTimeStr = $state("10:00");
  let apptStatus = $state("scheduled");
  let apptReason = $state("");
  let apptColor = $state("#3b82f6");
  let apptNotes = $state("");

  async function checkConfig() {
    try {
      supportedCountries = (await PracticeConfigService.GetSupportedCountries()) || [];
    } catch (e) {
      console.error("Failed to fetch supported countries from backend:", e);
    }

    try {
      const cfg = await PracticeConfigService.GetConfig();
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
      const meta = await PracticeConfigService.GetCountryConfig(countryCode);
      countryMeta = meta || null;
    } catch (e) {
      console.error("Could not fetch country meta from backend:", e);
      countryMeta = null;
    }
  }

  async function handleOnboardingComplete(countryCode: string) {
    try {
      const cfg = await PracticeConfigService.SetConfig(countryCode);
      practiceConfig = cfg;
      await loadCountryMeta(countryCode);
      showOnboarding = false;
      await loadPatients();
      await loadAppointments();
    } catch (err) {
      console.error("Failed to save onboarding practice config:", err);
    }
  }

  async function loadClinicData() {
    try {
      const provList = await PracticeConfigService.ListProviders();
      providers = (provList?.filter(Boolean) as Provider[]) || [];

      const opList = await PracticeConfigService.ListOperatories();
      operatories = (opList?.filter(Boolean) as Operatory[]) || [];
    } catch (err) {
      console.error("Failed to load clinic providers/operatories:", err);
    }
  }

  async function refreshClinic() {
    await checkConfig();
    await loadClinicData();
  }

  async function loadPatients() {
    loadingPatients = true;
    try {
      const res = await PatientService.ListPatients(searchQuery, statusFilter);
      patients = (res?.filter(Boolean) as Patient[]) || [];
    } catch (err) {
      console.error("Failed to load patients:", err);
    } finally {
      loadingPatients = false;
    }
  }

  async function loadAppointments() {
    loadingAppointments = true;
    try {
      const start = `${selectedDate}T00:00:00Z`;
      const end = `${selectedDate}T23:59:59Z`;
      const res = await AppointmentService.ListAppointments({
        start_date: start,
        end_date: end,
      } as any);
      appointments = (res?.filter(Boolean) as Appointment[]) || [];
    } catch (err) {
      console.error("Failed to load appointments:", err);
    } finally {
      loadingAppointments = false;
    }
  }

  // Patient Actions
  function openAddPatientModal() {
    isEditingPatient = false;
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

  function openEditPatientModal(p: Patient) {
    isEditingPatient = true;
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

    if (!countryMeta || !countryMeta.code) {
      alert("Practice country configuration is required before managing patient records.");
      return;
    }

    try {
      if (isEditingPatient) {
        const p = await PatientService.GetPatient(editingPatientId);
        if (p) {
          p.first_name = firstName;
          p.last_name = lastName;
          p.email = email;
          p.phone_primary = phone;
          p.date_of_birth = new Date(dob).toISOString();
          p.national_id = nationalId;
          p.national_id_type = countryMeta.national_id_type;
          p.state_province = stateProvince;
          p.postal_code = postalCode;
          p.country_code = countryMeta.code;
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
          middle_name: "",
          preferred_name: "",
          date_of_birth: new Date(dob).toISOString(),
          gender: Gender.GenderUndisclosed,
          email: email,
          phone_primary: phone,
          phone_secondary: "",
          address_line1: "",
          address_line2: "",
          city: "",
          state_province: stateProvince,
          postal_code: postalCode,
          country_code: countryMeta.code,
          national_id_type: countryMeta.national_id_type,
          national_id: nationalId,
          medical_alerts: medicalAlerts
            ? medicalAlerts.split(",").map((s) => s.trim())
            : [],
          allergies: [],
          notes: "",
          version: 1,
          status: Status.StatusActive,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
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

  // Appointment Actions
  function openAddApptModal() {
    isEditingAppt = false;
    editingApptId = "";
    apptPatientId = patients.length > 0 ? patients[0].id : "";
    apptProviderId = providers.length > 0 ? providers[0].id : "";
    apptOperatoryId = operatories.length > 0 ? operatories[0].id : "";
    apptStartDateStr = selectedDate;
    apptStartTimeStr = "09:00";
    apptEndTimeStr = "10:00";
    apptStatus = "scheduled";
    apptReason = "Routine Dental Examination & Cleaning";
    apptColor = "#3b82f6";
    apptNotes = "";
    showApptModal = true;
  }

  function openEditApptModal(appt: Appointment) {
    isEditingAppt = true;
    editingApptId = appt.id;
    apptPatientId = appt.patient_id;
    apptProviderId = appt.provider_id;
    apptOperatoryId = appt.operatory_id;
    if (appt.start_time) {
      const d = new Date(appt.start_time);
      apptStartDateStr = d.toISOString().split("T")[0];
      apptStartTimeStr = `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
    }
    if (appt.end_time) {
      const d = new Date(appt.end_time);
      apptEndTimeStr = `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
    }
    apptStatus = appt.status || "scheduled";
    apptReason = appt.reason || "";
    apptColor = appt.color || "#3b82f6";
    apptNotes = appt.notes || "";
    showApptModal = true;
  }

  async function handleSaveAppt(e: Event) {
    e.preventDefault();
    if (!apptPatientId || !apptProviderId || !apptOperatoryId) {
      alert("A valid patient, provider, and operatory room must be selected for the appointment.");
      return;
    }

    try {
      const startTimeISO = new Date(
        `${apptStartDateStr}T${apptStartTimeStr}:00`,
      ).toISOString();
      const endTimeISO = new Date(
        `${apptStartDateStr}T${apptEndTimeStr}:00`,
      ).toISOString();

      if (isEditingAppt) {
        const existing = await AppointmentService.GetAppointment(editingApptId);
        if (existing) {
          existing.patient_id = apptPatientId;
          existing.provider_id = apptProviderId;
          existing.operatory_id = apptOperatoryId;
          existing.start_time = startTimeISO;
          existing.end_time = endTimeISO;
          existing.status = apptStatus as AppointmentStatus;
          existing.reason = apptReason;
          existing.color = apptColor;
          existing.notes = apptNotes;
          await AppointmentService.UpdateAppointment(existing);
        }
      } else {
        const newAppt: Appointment = {
          id: "appt_" + Date.now(),
          patient_id: apptPatientId,
          provider_id: apptProviderId,
          operatory_id: apptOperatoryId,
          start_time: startTimeISO,
          end_time: endTimeISO,
          status: apptStatus as AppointmentStatus,
          reason: apptReason,
          color: apptColor,
          notes: apptNotes,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          version: 1,
        };
        await AppointmentService.CreateAppointment(newAppt);
      }
      showApptModal = false;
      await loadAppointments();
    } catch (err) {
      console.error("Failed to save appointment:", err);
    }
  }

  async function handleUpdateApptStatus(id: string, status: string) {
    try {
      await AppointmentService.UpdateAppointmentStatus(id, status);
      await loadAppointments();
    } catch (err) {
      console.error("Failed to update status:", err);
    }
  }

  async function handleDeleteAppt(id?: string) {
    const apptId = id || editingApptId;
    if (!apptId) return;
    if (confirm("Are you sure you want to delete this appointment?")) {
      try {
        await AppointmentService.DeleteAppointment(apptId);
        showApptModal = false;
        await loadAppointments();
      } catch (err) {
        console.error("Failed to delete appointment:", err);
      }
    }
  }

  // Reactivity: Reload appointments when selected date changes
  $effect(() => {
    if (selectedDate) {
      loadAppointments();
    }
  });

  onMount(async () => {
    if (typeof window !== "undefined" && window.matchMedia) {
      window
        .matchMedia("(prefers-color-scheme: light)")
        .addEventListener("change", () => {
          if (theme === "system") {
            applyTheme("system");
          }
        });
    }

    await loadTheme();
    await checkConfig();
    await loadClinicData();
    await loadPatients();
    await loadAppointments();
  });
</script>

<div class="min-h-screen flex flex-col bg-slate-950 text-slate-100 dark-app-wrapper">
  <Header
    bind:activeTab
    {countryMeta}
    onnewpatient={openAddPatientModal}
    onnewappointment={openAddApptModal}
    onopensettings={() => (showSettingsModal = true)}
  />

  <main class="w-full p-6 sm:p-8 box-border flex-1">
    {#if activeTab === "clinic"}
      <ClinicView
        bind:practiceConfig
        {countryMeta}
        {supportedCountries}
        bind:providers
        bind:operatories
        onrefresh={refreshClinic}
      />
    {:else if activeTab === "patients"}
      <PatientsView
        {patients}
        loading={loadingPatients}
        bind:searchQuery
        bind:statusFilter
        {countryMeta}
        onloadpatients={loadPatients}
        onaddpatient={openAddPatientModal}
        oneditpatient={openEditPatientModal}
        onarchivepatient={handleArchivePatient}
      />
    {:else if activeTab === "appointments"}
      <AppointmentsView
        {appointments}
        {patients}
        {providers}
        {operatories}
        loading={loadingAppointments}
        bind:selectedDate
        bind:selectedProvider
        bind:viewMode
        onnewappointment={openAddApptModal}
        oneditappointment={openEditApptModal}
        onupdatestatus={handleUpdateApptStatus}
        ondeleteappointment={handleDeleteAppt}
      />
    {/if}
  </main>
</div>

<SettingsModal
  bind:showModal={showSettingsModal}
  bind:theme
  onchangetheme={applyTheme}
/>

<OnboardingModal
  bind:showOnboarding
  {supportedCountries}
  oncomplete={handleOnboardingComplete}
/>

<PatientModal
  bind:showPatientModal
  isEditing={isEditingPatient}
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

<AppointmentModal
  bind:showModal={showApptModal}
  isEditing={isEditingAppt}
  {patients}
  configuredProviders={providers}
  configuredOperatories={operatories}
  bind:selectedPatientId={apptPatientId}
  bind:providerId={apptProviderId}
  bind:operatoryId={apptOperatoryId}
  bind:startDateStr={apptStartDateStr}
  bind:startTimeStr={apptStartTimeStr}
  bind:endTimeStr={apptEndTimeStr}
  bind:status={apptStatus}
  bind:reason={apptReason}
  bind:color={apptColor}
  bind:notes={apptNotes}
  onsave={handleSaveAppt}
  ondelete={() => handleDeleteAppt(editingApptId)}
/>
