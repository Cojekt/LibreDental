<script lang="ts">
  import { onMount, untrack } from "svelte";
  import {
    PatientService,
    PracticeConfigService,
    AppointmentService,
    SystemSettingsService,
  } from "@bindings/services/index.js";
  import { initLocale, getLocaleVersion } from "$lib/locale.svelte.js";
  import {
    getTodayDateString,
    getLocalDateString,
    getDateOnlyString,
    dateOnlyToISO,
  } from "$lib/date.js";
  import { handleError } from "$lib/error.js";
  import { m } from "./paraglide/messages.js";
  import type {
    Patient,
    PracticeConfig,
    CountryConfig,
    Appointment,
    Provider,
    Operatory,
  } from "@bindings/domain/index.js";
  import { Sex, Status, AppointmentStatus } from "@bindings/domain/index.js";
  import { auth } from "./stores/auth.svelte.js";

  import Header from "./components/Header.svelte";
  import OnboardingModal from "./components/OnboardingModal.svelte";
  import PatientModal from "./components/PatientModal.svelte";
  import AppointmentModal from "./components/AppointmentModal.svelte";
  import SettingsModal from "./components/SettingsModal.svelte";
  import StaffLoginModal from "./components/StaffLoginModal.svelte";
  import ClinicView from "./views/ClinicView.svelte";
  import PatientsView from "./views/PatientsView.svelte";
  import AppointmentsView from "./views/AppointmentsView.svelte";
  import ChartingView from "./views/ChartingView.svelte";
  import BillingView from "./views/BillingView.svelte";
  import AccountingView from "./views/AccountingView.svelte";
  import AuditView from "./views/AuditView.svelte";
  import ConfirmModal from "./components/ui/ConfirmModal.svelte";

  // App Navigation (Default to "clinic" landing tab on far left)
  let activeTab = $state("clinic");

  // Settings & Theme
  type ThemeMode = "dark" | "light" | "system";
  let showSettingsModal = $state(false);
  let showStaffLoginModal = $state(false);
  let theme = $state<ThemeMode>("system");

  async function getSystemOSTheme(): Promise<"dark" | "light"> {
    // The backend reports the OS preference of the machine it runs on, which is only
    // this client's OS in desktop mode; LAN clients of a server build use their own.
    try {
      const isDesktop = await SystemSettingsService.IsDesktopMode().catch(() => false);
      if (isDesktop) {
        const isDark = await SystemSettingsService.IsSystemDarkMode();
        if (typeof isDark === "boolean") {
          return isDark ? "dark" : "light";
        }
      }
    } catch (e) {
      console.warn("Could not query OS dark mode from backend:", e);
    }
    if (typeof window !== "undefined" && window.matchMedia) {
      return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
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
      const isDesktop = await SystemSettingsService.IsDesktopMode().catch(() => false);
      if (isDesktop) {
        await SystemSettingsService.SetTheme(newTheme);
      }
    } catch (e) {
      console.warn("Failed to persist theme in config:", e);
    }
  }

  async function loadTheme() {
    try {
      const isDesktop = await SystemSettingsService.IsDesktopMode().catch(() => false);
      if (isDesktop) {
        const dbTheme = await SystemSettingsService.GetTheme();
        if (dbTheme === "light" || dbTheme === "dark" || dbTheme === "system") {
          await applyTheme(dbTheme as ThemeMode);
          return;
        }
      }
    } catch (e) {
      console.warn("Could not load theme from DB, fallback to localStorage:", e);
    }
    const savedTheme = (localStorage.getItem("theme") as ThemeMode) || "system";
    await applyTheme(savedTheme);
  }

  // Clinic providers and operatories state
  let providers = $state<Provider[]>([]);
  let operatories = $state<Operatory[]>([]);

  // Patients state: `patients` backs the Patients tab and follows its search/status filter;
  // `directoryPatients` is every active patient, for pickers and name lookups elsewhere.
  let patients = $state<Patient[]>([]);
  let directoryPatients = $state<Patient[]>([]);
  let searchQuery = $state("");
  let statusFilter = $state("active");
  let loadingPatients = $state(false);

  // Configuration & Onboarding
  let practiceConfig = $state<PracticeConfig | null>(null);
  let countryMeta = $state<CountryConfig | null>(null);
  let supportedCountries = $state<CountryConfig[]>([]);
  let showOnboarding = $state(false);
  let onboardingStep = $state<1 | 2>(1);

  // Patient Modal states
  let showPatientModal = $state(false);
  let patientError = $state("");
  let isEditingPatient = $state(false);
  let editingPatientId = $state("");

  // Patient form fields
  let firstName = $state("");
  let lastName = $state("");
  let sex = $state<any>(Sex.SexMale);
  let email = $state("");
  let phone = $state("");
  let phoneSecondary = $state("");
  let dob = $state("");
  let nationalId = $state("");
  let addressLine1 = $state("");
  let addressLine2 = $state("");
  let city = $state("");
  let stateProvince = $state("");
  let postalCode = $state("");
  let emergencyName = $state("");
  let emergencyRel = $state("");
  let emergencyPhone = $state("");
  let guarantorName = $state("");
  let guarantorRel = $state("");
  let guarantorPhone = $state("");
  let insuranceCarrier = $state("");
  let insurancePolicy = $state("");
  let insuranceGroup = $state("");
  let insuranceIsSubscriber = $state(true);
  let insuranceSubscriberId = $state("");
  let preferredContactMethod = $state("phone");
  let preferredLanguage = $state("en");
  let reminderOptIn = $state(true);
  let preferredProviderId = $state("");
  let referralSource = $state("");
  let medicalAlerts = $state("");

  // Appointments state
  let appointments = $state<Appointment[]>([]);
  let loadingAppointments = $state(false);
  let selectedDate = $state(getLocalDateString());
  let selectedProvider = $state("all");
  let viewMode = $state<"calendar" | "grid" | "agenda">("calendar");

  // Appointment Modal states
  let showApptModal = $state(false);
  let apptError = $state("");
  let isEditingAppt = $state(false);
  let editingApptId = $state("");

  // Appointment form fields
  let apptPatientId = $state("");
  let apptProviderId = $state("");
  let apptOperatoryId = $state("");
  let apptStartDateStr = $state(getTodayDateString());
  let apptStartTimeStr = $state("09:00");
  let apptEndTimeStr = $state("10:00");
  let apptStatus = $state("scheduled");
  let apptReason = $state("");
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
        onboardingStep = 1;
        await loadCountryMeta("");
      } else {
        practiceConfig = cfg;
        await loadCountryMeta(cfg.country_code);
      }
    } catch (err) {
      console.error("Failed to check practice config:", err);
      showOnboarding = true;
      onboardingStep = 1;
      await loadCountryMeta("");
    }
  }

  // Covers installs that finished country setup but quit before creating the first provider.
  async function checkInitialProvider() {
    if (showOnboarding) return;
    try {
      if (await PracticeConfigService.NeedsInitialProvider()) {
        onboardingStep = 2;
        showOnboarding = true;
      }
    } catch (err) {
      // Fall back to step 2: if a provider does exist, creating one routes to sign-in instead.
      console.error("Failed to check for initial provider:", err);
      onboardingStep = 2;
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
      const cfg = await PracticeConfigService.SetConfig(auth.token, countryCode);
      practiceConfig = cfg;
      await loadCountryMeta(countryCode);
      if (await PracticeConfigService.NeedsInitialProvider()) {
        onboardingStep = 2;
        return;
      }
      // Another client created the first provider while this one was on step one.
      if (!auth.token) {
        await handleAlreadyInitialized();
        return;
      }
      showOnboarding = false;
      await refreshPatientLists();
      await loadAppointments();
    } catch (err: any) {
      // A sessionless save is rejected once another client has created the first provider.
      if (!auth.token && err?.message?.includes("unauthorized")) {
        await handleAlreadyInitialized();
        return;
      }
      console.error("Failed to save onboarding practice config:", err);
    }
  }

  async function handleInitialProviderCreated() {
    showOnboarding = false;
    onboardingStep = 1;
    await loadClinicData();
  }

  async function handleAlreadyInitialized() {
    showOnboarding = false;
    onboardingStep = 1;
    await loadClinicData();
    showStaffLoginModal = true;
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

  let patientsRequestGen = 0;

  async function loadPatients() {
    if (!auth.token) return;
    const gen = ++patientsRequestGen;
    loadingPatients = true;
    try {
      const query = searchQuery;
      const status = statusFilter;
      const res = await PatientService.ListPatients(auth.token, query, status);
      if (gen !== patientsRequestGen) return; // a newer search superseded this one
      patients = (res?.filter(Boolean) as Patient[]) || [];
      if (isUnfilteredPatientList(query, status)) {
        directoryRequestGen++; // supersede any in-flight directory load with this result
        directoryPatients = patients;
      }
    } catch (err) {
      console.error("Failed to load patients:", err);
    } finally {
      if (gen === patientsRequestGen) {
        loadingPatients = false;
      }
    }
  }

  let directoryRequestGen = 0;

  function isUnfilteredPatientList(query: string, status: string): boolean {
    return !query && status === "active";
  }

  // Refresh both lists, reusing the Patients tab result as the directory when that tab is
  // unfiltered, so it isn't fetched (and audit-logged) twice.
  async function refreshPatientLists() {
    const unfiltered = isUnfilteredPatientList(searchQuery, statusFilter);
    await Promise.all([loadPatients(), unfiltered ? null : loadDirectoryPatients()]);
  }

  async function loadDirectoryPatients() {
    if (!auth.token) return;
    const gen = ++directoryRequestGen;
    try {
      const res = await PatientService.ListPatients(auth.token, "", "active");
      if (gen !== directoryRequestGen) return;
      directoryPatients = (res?.filter(Boolean) as Patient[]) || [];
    } catch (err) {
      console.error("Failed to load patient directory:", err);
    }
  }

  let appointmentsRequestGen = 0;

  async function loadAppointments() {
    if (!auth.token) return;
    const gen = ++appointmentsRequestGen;
    // Only block the view on the first load; later refreshes (e.g. date navigation)
    // swap data in place instead of flashing a spinner over the calendar.
    if (appointments.length === 0) loadingAppointments = true;
    try {
      const res = await AppointmentService.ListAppointments(auth.token, {} as any);
      if (gen !== appointmentsRequestGen) return;
      appointments = (res?.filter(Boolean) as Appointment[]) || [];
    } catch (err) {
      console.error("Failed to load appointments:", err);
    } finally {
      if (gen === appointmentsRequestGen) loadingAppointments = false;
    }
  }

  function parseMedicalAlerts(value: string): string[] {
    return value
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
  }

  // Patient Actions
  function openAddPatientModal() {
    isEditingPatient = false;
    editingPatientId = "";
    firstName = "";
    lastName = "";
    sex = Sex.SexMale;
    email = "";
    phone = "";
    phoneSecondary = "";
    dob = "";
    nationalId = "";
    addressLine1 = "";
    addressLine2 = "";
    city = "";
    stateProvince = "";
    postalCode = "";
    emergencyName = "";
    emergencyRel = "";
    emergencyPhone = "";
    guarantorName = "";
    guarantorRel = "";
    guarantorPhone = "";
    insuranceCarrier = "";
    insurancePolicy = "";
    insuranceGroup = "";
    insuranceIsSubscriber = true;
    insuranceSubscriberId = "";
    preferredContactMethod = "phone";
    preferredLanguage = "en";
    reminderOptIn = true;
    preferredProviderId = "";
    referralSource = "";
    medicalAlerts = "";
    patientError = "";
    showPatientModal = true;
  }

  function openEditPatientModal(p: Patient) {
    isEditingPatient = true;
    editingPatientId = p.id;
    firstName = p.first_name;
    lastName = p.last_name;
    sex = p.sex || Sex.SexMale;
    email = p.email || "";
    phone = p.phone_primary || "";
    phoneSecondary = p.phone_secondary || "";
    dob = getDateOnlyString(p.date_of_birth);
    nationalId = p.national_id || "";
    addressLine1 = p.address_line1 || "";
    addressLine2 = p.address_line2 || "";
    city = p.city || "";
    stateProvince = p.state_province || "";
    postalCode = p.postal_code || "";
    emergencyName = p.emergency_contact_name || "";
    emergencyRel = p.emergency_contact_rel || "";
    emergencyPhone = p.emergency_contact_phone || "";
    guarantorName = p.guarantor_name || "";
    guarantorRel = p.guarantor_rel || "";
    guarantorPhone = p.guarantor_phone || "";
    insuranceCarrier = p.insurance_carrier || "";
    insurancePolicy = p.insurance_policy_number || "";
    insuranceGroup = p.insurance_group_number || "";
    insuranceIsSubscriber = p.insurance_is_subscriber;
    insuranceSubscriberId = p.insurance_subscriber_id || "";
    preferredContactMethod = p.preferred_contact_method || "phone";
    preferredLanguage = p.preferred_language || "en";
    reminderOptIn = p.reminder_opt_in !== false;
    preferredProviderId = p.preferred_provider_id || "";
    referralSource = p.referral_source || "";
    medicalAlerts = p.medical_alerts ? p.medical_alerts.join(", ") : "";
    patientError = "";
    showPatientModal = true;
  }

  async function handleSavePatient(e: Event) {
    e.preventDefault();
    if (!firstName || !lastName || !dob || !phone) return;

    if (!countryMeta || !countryMeta.code) {
      patientError = m.alert_practice_country_required();
      return;
    }
    if (dob > getTodayDateString()) {
      patientError = m.patient_err_dob_future();
      return;
    }
    patientError = "";

    try {
      if (isEditingPatient) {
        const p = await PatientService.GetPatient(auth.token, editingPatientId);
        if (p) {
          p.first_name = firstName;
          p.last_name = lastName;
          p.sex = sex;
          p.email = email;
          p.phone_primary = phone;
          p.phone_secondary = phoneSecondary;
          p.date_of_birth = dateOnlyToISO(dob);
          p.national_id = nationalId;
          p.national_id_type = countryMeta.national_id_type;
          p.address_line1 = addressLine1;
          p.address_line2 = addressLine2;
          p.city = city;
          p.state_province = stateProvince;
          p.postal_code = postalCode;
          p.country_code = countryMeta.code;
          p.emergency_contact_name = emergencyName;
          p.emergency_contact_rel = emergencyRel;
          p.emergency_contact_phone = emergencyPhone;
          p.guarantor_name = guarantorName;
          p.guarantor_rel = guarantorRel;
          p.guarantor_phone = guarantorPhone;
          p.insurance_carrier = insuranceCarrier;
          p.insurance_policy_number = insurancePolicy;
          p.insurance_group_number = insuranceGroup;
          p.insurance_is_subscriber = insuranceIsSubscriber;
          p.insurance_subscriber_id = insuranceSubscriberId;
          p.preferred_contact_method = preferredContactMethod;
          p.preferred_language = preferredLanguage;
          p.reminder_opt_in = reminderOptIn;
          p.preferred_provider_id = preferredProviderId;
          p.referral_source = referralSource;
          p.medical_alerts = parseMedicalAlerts(medicalAlerts);
          await PatientService.UpdatePatient(auth.token, p);
        }
      } else {
        const newPatient: Omit<Patient, "created_at" | "updated_at"> = {
          id: "pat_" + Date.now(),
          first_name: firstName,
          last_name: lastName,
          preferred_name: "",
          date_of_birth: dateOnlyToISO(dob),
          sex: sex,
          email: email,
          phone_primary: phone,
          phone_secondary: phoneSecondary,
          emergency_contact_name: emergencyName,
          emergency_contact_rel: emergencyRel,
          emergency_contact_phone: emergencyPhone,
          guarantor_name: guarantorName,
          guarantor_rel: guarantorRel,
          guarantor_phone: guarantorPhone,
          insurance_carrier: insuranceCarrier,
          insurance_policy_number: insurancePolicy,
          insurance_group_number: insuranceGroup,
          insurance_is_subscriber: insuranceIsSubscriber,
          insurance_subscriber_id: insuranceSubscriberId,
          preferred_contact_method: preferredContactMethod,
          preferred_language: preferredLanguage,
          reminder_opt_in: reminderOptIn,
          preferred_provider_id: preferredProviderId,
          referral_source: referralSource,
          address_line1: addressLine1,
          address_line2: addressLine2,
          city: city,
          state_province: stateProvince,
          postal_code: postalCode,
          country_code: countryMeta.code,
          national_id_type: countryMeta.national_id_type,
          national_id: nationalId,
          medical_alerts: parseMedicalAlerts(medicalAlerts),
          allergies: [],
          notes: "",
          version: 1,
          status: Status.StatusActive,
        };
        await PatientService.CreatePatient(auth.token, newPatient as unknown as Patient);
      }
      showPatientModal = false;
      await refreshPatientLists();
    } catch (err) {
      console.error("Failed to save patient:", err);
      patientError = handleError(err, m.patient_err_save());
    }
  }

  let showConfirmArchivePatient = $state(false);
  let patientToArchive = $state<Patient | null>(null);

  async function handleArchivePatient(p: Patient) {
    patientToArchive = p;
    showConfirmArchivePatient = true;
  }

  async function executeArchivePatient() {
    if (!patientToArchive) return;
    try {
      await PatientService.ArchivePatient(auth.token, patientToArchive.id);
      await refreshPatientLists();
    } catch (err) {
      console.error("Failed to archive patient:", err);
    } finally {
      patientToArchive = null;
    }
  }

  // Appointment Actions
  function openAddApptModal() {
    isEditingAppt = false;
    editingApptId = "";
    // No default patient: silently preselecting the first one makes it easy to book the
    // wrong person, so the picker starts on its "select a patient" placeholder.
    apptPatientId = "";
    apptProviderId = providers.length > 0 ? providers[0].id : "";
    apptOperatoryId = operatories.length > 0 ? operatories[0].id : "";
    apptStartDateStr = selectedDate;
    apptStartTimeStr = "09:00";
    apptEndTimeStr = "10:00";
    apptStatus = "scheduled";
    apptReason = "";
    apptNotes = "";
    apptError = "";
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
      apptStartDateStr = getLocalDateString(d);
      apptStartTimeStr = `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
    }
    if (appt.end_time) {
      const d = new Date(appt.end_time);
      apptEndTimeStr = `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
    }
    apptStatus = appt.status || "scheduled";
    apptReason = appt.reason || "";
    apptNotes = appt.notes || "";
    apptError = "";
    showApptModal = true;
  }

  async function handleSaveAppt(e: Event) {
    e.preventDefault();
    if (!apptPatientId || !apptProviderId || !apptOperatoryId) {
      apptError = m.alert_appointment_validation();
      return;
    }
    if (apptEndTimeStr <= apptStartTimeStr) {
      apptError = m.appt_err_end_before_start();
      return;
    }
    apptError = "";

    try {
      const startTimeISO = new Date(`${apptStartDateStr}T${apptStartTimeStr}:00`).toISOString();
      const endTimeISO = new Date(`${apptStartDateStr}T${apptEndTimeStr}:00`).toISOString();

      if (isEditingAppt) {
        const existing = await AppointmentService.GetAppointment(auth.token, editingApptId);
        if (existing) {
          existing.patient_id = apptPatientId;
          existing.provider_id = apptProviderId;
          existing.operatory_id = apptOperatoryId;
          existing.start_time = startTimeISO;
          existing.end_time = endTimeISO;
          existing.status = apptStatus as any;
          existing.reason = apptReason;
          existing.notes = apptNotes;
          await AppointmentService.UpdateAppointment(auth.token, existing);
        }
      } else {
        const newAppt: Omit<Appointment, "created_at" | "updated_at"> = {
          id: "appt_" + Date.now(),
          patient_id: apptPatientId,
          provider_id: apptProviderId,
          operatory_id: apptOperatoryId,
          start_time: startTimeISO,
          end_time: endTimeISO,
          status: apptStatus as any,
          reason: apptReason,
          notes: apptNotes,
          version: 1,
        };
        await AppointmentService.CreateAppointment(auth.token, newAppt as unknown as Appointment);
      }
      showApptModal = false;
      await loadAppointments();
    } catch (err) {
      console.error("Failed to save appointment:", err);
      apptError = handleError(err, m.appt_err_save());
    }
  }

  async function handleUpdateApptStatus(id: string, status: string) {
    try {
      await AppointmentService.UpdateAppointmentStatus(auth.token, id, status);
      await loadAppointments();
    } catch (err) {
      console.error("Failed to update status:", err);
    }
  }

  let showConfirmDeleteAppt = $state(false);
  let apptToDelete = $state("");

  async function handleDeleteAppt(id?: string) {
    const apptId = id || editingApptId;
    if (!apptId) return;
    apptToDelete = apptId;
    showConfirmDeleteAppt = true;
  }

  async function executeDeleteAppt() {
    if (!apptToDelete) return;
    try {
      await AppointmentService.DeleteAppointment(auth.token, apptToDelete);
      showApptModal = false;
      await loadAppointments();
    } catch (err) {
      console.error("Failed to delete appointment:", err);
    } finally {
      apptToDelete = "";
    }
  }

  // Load patient data once per session and refresh appointments whenever the session or the
  // selected date changes. A single effect keeps login from firing (and audit-logging) the
  // appointment list twice.
  let loadedToken = "";
  $effect(() => {
    const token = auth.token;
    void selectedDate;
    untrack(() => {
      if (!token) {
        // Drop responses still in flight from the previous session.
        patientsRequestGen++;
        directoryRequestGen++;
        appointmentsRequestGen++;
        loadingPatients = false;
        loadingAppointments = false;
        loadedToken = "";
        patients = [];
        directoryPatients = [];
        appointments = [];
        return;
      }
      if (token !== loadedToken) {
        loadedToken = token;
        refreshPatientLists();
      }
      loadAppointments();
    });
  });

  onMount(async () => {
    if (typeof window !== "undefined" && window.matchMedia) {
      window.matchMedia("(prefers-color-scheme: light)").addEventListener("change", () => {
        if (theme === "system") {
          applyTheme("system");
        }
      });
    }

    await loadTheme();

    // Initialize i18n via Go backend locale resolution
    await initLocale();

    await checkConfig();
    await checkInitialProvider();
    await loadClinicData();
  });
</script>

<div
  class="min-h-screen flex flex-col bg-slate-950 text-slate-100 dark-app-wrapper"
  data-locale={getLocaleVersion()}
>
  <Header
    bind:activeTab
    onnewpatient={openAddPatientModal}
    onnewappointment={openAddApptModal}
    onopensettings={() => (showSettingsModal = true)}
    onopenstafflogin={() => (showStaffLoginModal = true)}
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
        onrequestlogin={() => (showStaffLoginModal = true)}
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
        patients={directoryPatients}
        {providers}
        {operatories}
        businessHours={practiceConfig?.business_hours}
        loading={loadingAppointments}
        bind:selectedDate
        bind:selectedProvider
        bind:viewMode
        onnewappointment={openAddApptModal}
        oneditappointment={openEditApptModal}
        onupdatestatus={handleUpdateApptStatus}
        ondeleteappointment={handleDeleteAppt}
      />
    {:else if activeTab === "charting"}
      <ChartingView patients={directoryPatients} {countryMeta} />
    {:else if activeTab === "billing"}
      <BillingView patients={directoryPatients} {providers} {countryMeta} />
    {:else if activeTab === "accounting"}
      <AccountingView {providers} {countryMeta} />
    {:else if activeTab === "audit"}
      <AuditView patients={directoryPatients} />
    {/if}
  </main>
</div>

<SettingsModal bind:showModal={showSettingsModal} bind:theme onchangetheme={applyTheme} />

<StaffLoginModal
  bind:showModal={showStaffLoginModal}
  {providers}
  onlogout={() => (activeTab = "clinic")}
/>

<OnboardingModal
  bind:showOnboarding
  bind:step={onboardingStep}
  {supportedCountries}
  savedCountry={practiceConfig?.country_code}
  oncomplete={handleOnboardingComplete}
  onprovidercreated={handleInitialProviderCreated}
  onalreadyinitialized={handleAlreadyInitialized}
/>

<PatientModal
  bind:showPatientModal
  isEditing={isEditingPatient}
  bind:firstName
  bind:lastName
  bind:sex
  bind:dob
  bind:email
  bind:phone
  bind:phoneSecondary
  bind:nationalId
  bind:addressLine1
  bind:addressLine2
  bind:city
  bind:stateProvince
  bind:postalCode
  bind:emergencyName
  bind:emergencyRel
  bind:emergencyPhone
  bind:guarantorName
  bind:guarantorRel
  bind:guarantorPhone
  bind:insuranceCarrier
  bind:insurancePolicy
  bind:insuranceGroup
  bind:insuranceIsSubscriber
  bind:insuranceSubscriberId
  bind:preferredContactMethod
  bind:preferredLanguage
  bind:reminderOptIn
  bind:preferredProviderId
  bind:referralSource
  bind:medicalAlerts
  {countryMeta}
  configuredProviders={providers}
  errorMsg={patientError}
  onsave={handleSavePatient}
/>

<AppointmentModal
  bind:showModal={showApptModal}
  isEditing={isEditingAppt}
  patients={directoryPatients}
  errorMsg={apptError}
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
  bind:notes={apptNotes}
  onsave={handleSaveAppt}
  ondelete={() => handleDeleteAppt(editingApptId)}
/>

<ConfirmModal
  bind:showModal={showConfirmArchivePatient}
  title={m.common_confirm()}
  message={patientToArchive
    ? m.confirm_archive_patient({
        firstName: patientToArchive.first_name,
        lastName: patientToArchive.last_name,
      })
    : ""}
  onConfirm={executeArchivePatient}
/>

<ConfirmModal
  bind:showModal={showConfirmDeleteAppt}
  title={m.common_confirm()}
  message={m.confirm_delete_appointment()}
  onConfirm={executeDeleteAppt}
/>
