<script lang="ts">
  import type {
    PracticeConfig,
    CountryConfig,
    Provider,
    Operatory,
    BusinessHourDay,
  } from "@bindings/domain/models.js";
  import { ProviderRole, OperatoryType } from "@bindings/domain/models.js";
  import { PracticeConfigService } from "@bindings/services/index.js";
  import TabNav from "../components/ui/TabNav.svelte";
  import ClinicProfileSection from "./clinic/ClinicProfileSection.svelte";
  import ClinicHoursSection from "./clinic/ClinicHoursSection.svelte";
  import ProvidersSection from "./clinic/ProvidersSection.svelte";
  import OperatoriesSection from "./clinic/OperatoriesSection.svelte";
  import DocumentsSection from "./clinic/DocumentsSection.svelte";
  import IntegrationsSection from "./clinic/IntegrationsSection.svelte";
  import { m } from "../paraglide/messages.js";

  let {
    practiceConfig = $bindable(),
    countryMeta,
    supportedCountries = [],
    providers = $bindable([]),
    operatories = $bindable([]),
    onrefresh,
  } = $props<{
    practiceConfig: PracticeConfig | null;
    countryMeta?: CountryConfig | null;
    supportedCountries?: CountryConfig[];
    providers: Provider[];
    operatories: Operatory[];
    onrefresh: () => Promise<void>;
  }>();

  let activeSubTab = $state<
    "profile" | "hours" | "providers" | "operatories" | "documents" | "integrations"
  >("profile");
  let savingProfile = $state(false);
  let profileMessage = $state<{
    text: string;
    type: "success" | "error";
  } | null>(null);

  function ensureDaySlots(hours: BusinessHourDay[]): BusinessHourDay[] {
    return hours.map((h) => {
      const slots =
        h.slots && h.slots.length > 0
          ? h.slots
          : [
              {
                open_time: h.open_time || "08:00",
                close_time: h.close_time || "17:00",
              },
            ];
      const breaks = h.breaks ? [...h.breaks] : [];
      return {
        ...h,
        slots: slots,
        breaks: breaks,
      };
    });
  }

  function addSlot(hour: BusinessHourDay) {
    if (!hour.slots) hour.slots = [];
    const lastSlot = hour.slots[hour.slots.length - 1];
    const newOpen = lastSlot ? lastSlot.close_time : "13:00";
    hour.slots = [...hour.slots, { open_time: newOpen, close_time: "17:00" }];
    syncDayBounds(hour);
  }

  function removeSlot(hour: BusinessHourDay, index: number) {
    if (hour.slots && hour.slots.length > 1) {
      hour.slots = hour.slots.filter((_, i) => i !== index);
      syncDayBounds(hour);
    }
  }

  function addBreak(hour: BusinessHourDay) {
    if (!hour.breaks) hour.breaks = [];
    hour.breaks = [...hour.breaks, { name: "Break", start_time: "12:00", end_time: "13:00" }];
  }

  function removeBreak(hour: BusinessHourDay, index: number) {
    if (hour.breaks) {
      hour.breaks = hour.breaks.filter((_, i) => i !== index);
    }
  }

  function syncDayBounds(hour: BusinessHourDay) {
    if (hour.slots && hour.slots.length > 0) {
      hour.open_time = hour.slots[0].open_time;
      hour.close_time = hour.slots[hour.slots.length - 1].close_time;
    }
  }

  function formatTime12(time24: string): string {
    if (!time24) return "";
    const [hStr, mStr] = time24.split(":");
    let h = parseInt(hStr, 10);
    if (isNaN(h)) return time24;
    const m = mStr || "00";
    const ampm = h >= 12 ? "PM" : "AM";
    h = h % 12;
    if (h === 0) h = 12;
    return `${h.toString().padStart(2, "0")}:${m} ${ampm}`;
  }

  // Form local states for Practice Config
  let clinicName = $state(practiceConfig?.clinic_name || "My Dental Clinic");
  let tagline = $state(practiceConfig?.tagline || "");
  let taxId = $state(practiceConfig?.tax_id || "");
  let licenseNumber = $state(practiceConfig?.license_number || "");
  let phone = $state(practiceConfig?.phone || "");
  let email = $state(practiceConfig?.email || "");
  let website = $state(practiceConfig?.website || "");
  let addressLine1 = $state(practiceConfig?.address_line1 || "");
  let addressLine2 = $state(practiceConfig?.address_line2 || "");
  let city = $state(practiceConfig?.city || "");
  let stateProvince = $state(practiceConfig?.state_province || "");
  let postalCode = $state(practiceConfig?.postal_code || "");
  let countryCode = $state(practiceConfig?.country_code || "");
  let currency = $state(practiceConfig?.currency || "");
  let toothSystem = $state(practiceConfig?.tooth_system || "");
  let dateFormat = $state(practiceConfig?.date_format || "");
  let businessHours = $state<BusinessHourDay[]>(
    ensureDaySlots(
      practiceConfig?.business_hours || [
        {
          day: "Monday",
          open_time: "08:00",
          close_time: "17:00",
          is_closed: false,
          slots: [{ open_time: "08:00", close_time: "17:00" }],
          breaks: [{ name: "Lunch Break", start_time: "12:00", end_time: "13:00" }],
        },
        {
          day: "Tuesday",
          open_time: "08:00",
          close_time: "17:00",
          is_closed: false,
          slots: [{ open_time: "08:00", close_time: "17:00" }],
          breaks: [{ name: "Lunch Break", start_time: "12:00", end_time: "13:00" }],
        },
        {
          day: "Wednesday",
          open_time: "08:00",
          close_time: "17:00",
          is_closed: false,
          slots: [{ open_time: "08:00", close_time: "17:00" }],
          breaks: [{ name: "Lunch Break", start_time: "12:00", end_time: "13:00" }],
        },
        {
          day: "Thursday",
          open_time: "08:00",
          close_time: "17:00",
          is_closed: false,
          slots: [{ open_time: "08:00", close_time: "17:00" }],
          breaks: [{ name: "Lunch Break", start_time: "12:00", end_time: "13:00" }],
        },
        {
          day: "Friday",
          open_time: "08:00",
          close_time: "17:00",
          is_closed: false,
          slots: [{ open_time: "08:00", close_time: "17:00" }],
          breaks: [{ name: "Lunch Break", start_time: "12:00", end_time: "13:00" }],
        },
        {
          day: "Saturday",
          open_time: "08:00",
          close_time: "17:00",
          is_closed: true,
          slots: [{ open_time: "08:00", close_time: "17:00" }],
          breaks: [],
        },
        {
          day: "Sunday",
          open_time: "08:00",
          close_time: "17:00",
          is_closed: true,
          slots: [{ open_time: "08:00", close_time: "17:00" }],
          breaks: [],
        },
      ]
    )
  );

  $effect(() => {
    if (practiceConfig) {
      clinicName = practiceConfig.clinic_name || "My Dental Clinic";
      tagline = practiceConfig.tagline || "";
      taxId = practiceConfig.tax_id || "";
      licenseNumber = practiceConfig.license_number || "";
      phone = practiceConfig.phone || "";
      email = practiceConfig.email || "";
      website = practiceConfig.website || "";
      addressLine1 = practiceConfig.address_line1 || "";
      addressLine2 = practiceConfig.address_line2 || "";
      city = practiceConfig.city || "";
      stateProvince = practiceConfig.state_province || "";
      postalCode = practiceConfig.postal_code || "";
      countryCode = practiceConfig.country_code || countryMeta?.code || "";
      currency = practiceConfig.currency || countryMeta?.default_currency || "";
      toothSystem = practiceConfig.tooth_system || countryMeta?.default_tooth_system || "";
      dateFormat = practiceConfig.date_format || countryMeta?.date_format || "";
      if (practiceConfig.business_hours && practiceConfig.business_hours.length > 0) {
        businessHours = ensureDaySlots(practiceConfig.business_hours);
      }
    } else if (countryMeta) {
      if (!countryCode) countryCode = countryMeta.code;
      if (!currency) currency = countryMeta.default_currency;
      if (!toothSystem) toothSystem = countryMeta.default_tooth_system;
      if (!dateFormat) dateFormat = countryMeta.date_format;
    }
  });

  // Modal / Form state for Provider
  let showProviderModal = $state(false);
  let isEditingProvider = $state(false);
  let provId = $state("");
  let provName = $state("");
  let provRole = $state<string>("dentist");
  let provSpecialty = $state("");
  let provLicense = $state("");
  let provEmail = $state("");
  let provPhone = $state("");
  let provColor = $state("#3b82f6");
  let provPin = $state("");
  let provIsActive = $state(true);
  let provHourlyRate = $state(0.0);

  // Modal / Form state for Operatory
  let showOperatoryModal = $state(false);
  let isEditingOperatory = $state(false);
  let opId = $state("");
  let opName = $state("");
  let opRoomCode = $state("");
  let opType = $state<string>("general");
  let opIsActive = $state(true);

  let triggerUploadDocument = $state<(() => void) | undefined>();

  let isEditingProfile = $state(false);
  let toastTimeout: any = null;

  function setProfileMessage(text: string, type: "success" | "error") {
    if (toastTimeout) clearTimeout(toastTimeout);
    profileMessage = { text, type };
    toastTimeout = setTimeout(() => {
      profileMessage = null;
    }, 3000);
  }

  function resetFormFields() {
    if (practiceConfig) {
      clinicName = practiceConfig.clinic_name || "My Dental Clinic";
      tagline = practiceConfig.tagline || "";
      taxId = practiceConfig.tax_id || "";
      licenseNumber = practiceConfig.license_number || "";
      phone = practiceConfig.phone || "";
      email = practiceConfig.email || "";
      website = practiceConfig.website || "";
      addressLine1 = practiceConfig.address_line1 || "";
      addressLine2 = practiceConfig.address_line2 || "";
      city = practiceConfig.city || "";
      stateProvince = practiceConfig.state_province || "";
      postalCode = practiceConfig.postal_code || "";
      countryCode = practiceConfig.country_code || countryMeta?.code || "";
      currency = practiceConfig.currency || countryMeta?.default_currency || "";
      toothSystem = practiceConfig.tooth_system || countryMeta?.default_tooth_system || "";
      dateFormat = practiceConfig.date_format || countryMeta?.date_format || "";
      if (practiceConfig.business_hours && practiceConfig.business_hours.length > 0) {
        businessHours = practiceConfig.business_hours;
      }
    }
  }

  function cancelEditProfile() {
    resetFormFields();
    isEditingProfile = false;
    if (toastTimeout) clearTimeout(toastTimeout);
    profileMessage = null;
  }

  async function handleSaveConfig() {
    savingProfile = true;
    profileMessage = null;

    try {
      const updatedConfig: PracticeConfig = {
        id: 1,
        clinic_name: clinicName,
        tagline: tagline,
        tax_id: taxId,
        license_number: licenseNumber,
        phone: phone,
        email: email,
        website: website,
        address_line1: addressLine1,
        address_line2: addressLine2,
        city: city,
        state_province: stateProvince,
        postal_code: postalCode,
        country_code: countryCode as any,
        currency: currency,
        tooth_system: toothSystem as any,
        date_format: dateFormat,
        business_hours: businessHours,
        created_at: practiceConfig?.created_at || new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      const res = await PracticeConfigService.UpdatePracticeConfig(updatedConfig);
      if (res) {
        practiceConfig = res;
        setProfileMessage("Clinic settings saved!", "success");
        isEditingProfile = false;
        await onrefresh();
      }
    } catch (err) {
      console.error("Failed to update clinic config:", err);
      setProfileMessage("Failed to save settings.", "error");
    } finally {
      savingProfile = false;
    }
  }

  // Provider Handlers
  function openAddProviderModal() {
    isEditingProvider = false;
    provId = "";
    provName = "";
    provRole = "dentist";
    provSpecialty = "General Dentistry";
    provLicense = "";
    provEmail = "";
    provPhone = "";
    provColor = "#3b82f6";
    provPin = "";
    provIsActive = true;
    provHourlyRate = 0.0;
    showProviderModal = true;
  }

  function openEditProviderModal(p: Provider) {
    isEditingProvider = true;
    provId = p.id;
    provName = p.name;
    provRole = p.role || "dentist";
    provSpecialty = p.specialty || "";
    provLicense = p.license_number || "";
    provEmail = p.email || "";
    provPhone = p.phone || "";
    provColor = p.color || "#3b82f6";
    provPin = p.pin || "";
    provIsActive = p.is_active;
    provHourlyRate = (p.hourly_rate || 0.0) / 100;
    showProviderModal = true;
  }

  async function handleSaveProvider(e: Event) {
    e.preventDefault();
    if (!provName) return;

    try {
      const p: Provider = {
        id: provId,
        name: provName,
        role: provRole as ProviderRole,
        specialty: provSpecialty,
        license_number: provLicense,
        email: provEmail,
        phone: provPhone,
        color: provColor,
        pin: provPin,
        is_active: provIsActive,
        hourly_rate: Math.round(provHourlyRate * 100),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      await PracticeConfigService.SaveProvider(p);
      showProviderModal = false;
      await onrefresh();
    } catch (err) {
      console.error("Failed to save provider:", err);
    }
  }

  async function handleDeleteProvider(id: string) {
    if (confirm(m.confirm_delete_provider())) {
      try {
        await PracticeConfigService.DeleteProvider(id);
        await onrefresh();
      } catch (err) {
        console.error("Failed to delete provider:", err);
      }
    }
  }

  // Operatory Handlers
  function openAddOperatoryModal() {
    isEditingOperatory = false;
    opId = "";
    opName = "";
    opRoomCode = "";
    opType = "general";
    opIsActive = true;
    showOperatoryModal = true;
  }

  function openEditOperatoryModal(op: Operatory) {
    isEditingOperatory = true;
    opId = op.id;
    opName = op.name;
    opRoomCode = op.room_code || "";
    opType = op.type || "general";
    opIsActive = op.is_active;
    showOperatoryModal = true;
  }

  async function handleSaveOperatory(e: Event) {
    e.preventDefault();
    if (!opName) return;

    try {
      const op: Operatory = {
        id: opId,
        name: opName,
        room_code: opRoomCode,
        type: opType as OperatoryType,
        is_active: opIsActive,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      await PracticeConfigService.SaveOperatory(op);
      showOperatoryModal = false;
      await onrefresh();
    } catch (err) {
      console.error("Failed to save operatory:", err);
    }
  }

  async function handleDeleteOperatory(id: string) {
    if (confirm(m.confirm_delete_operatory())) {
      try {
        await PracticeConfigService.DeleteOperatory(id);
        await onrefresh();
      } catch (err) {
        console.error("Failed to delete operatory:", err);
      }
    }
  }

  let tabs = $derived([
    { id: "profile", label: "Practice Profile & Standards" },
    { id: "hours", label: "Operating Hours" },
    { id: "providers", label: "Providers & Staff", count: providers.length },
    { id: "operatories", label: "Operatories & Chairs", count: operatories.length },
    { id: "documents", label: "Clinic Documents" },
    { id: "integrations", label: "Integrations" },
  ]);
</script>

<div class="space-y-6">
  <div class="flex flex-wrap items-center justify-between border-b border-slate-800 gap-4">
    <TabNav {tabs} bind:activeTab={activeSubTab} />

    <div class="flex items-center gap-3">
      {#if profileMessage}
        <div
          class={`text-xs font-semibold px-3 py-1.5 rounded-xl border flex items-center gap-1.5 transition-all shadow-sm ${
            profileMessage.type === "success"
              ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-400"
              : "border-rose-500/30 bg-rose-500/10 text-rose-400"
          }`}
        >
          <span>{profileMessage.type === "success" ? "✓" : "⚠️"}</span>
          <span>{profileMessage.text}</span>
        </div>
      {/if}

      {#if activeSubTab === "profile" || activeSubTab === "hours"}
        <div class="flex items-center gap-2">
          {#if !isEditingProfile}
            <button
              type="button"
              onclick={() => (isEditingProfile = true)}
              class="btn btn-primary text-xs shadow-md shadow-sky-500/20 flex items-center gap-1.5 px-4 py-2"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
              </svg>
              <span>{activeSubTab === "hours" ? "Edit Hours" : "Edit Practice Info"}</span>
            </button>
          {:else}
            <button
              type="button"
              onclick={cancelEditProfile}
              disabled={savingProfile}
              class="rounded-xl border border-slate-700 bg-slate-900 px-4 py-2 text-xs font-semibold text-slate-300 hover:bg-slate-800 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onclick={handleSaveConfig}
              disabled={savingProfile}
              class="btn btn-primary text-xs shadow-md shadow-sky-500/20 flex items-center gap-1.5 px-4 py-2"
            >
              {#if savingProfile}
                <div
                  class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent"
                ></div>
                <span>Saving...</span>
              {:else}
                <svg
                  class="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                  <polyline points="17 21 17 13 7 13 7 21" />
                  <polyline points="7 3 7 8 15 8" />
                </svg>
                <span>Save Changes</span>
              {/if}
            </button>
          {/if}
        </div>
      {:else if activeSubTab === "providers"}
        <button
          type="button"
          onclick={openAddProviderModal}
          class="btn btn-primary text-xs shadow-md shadow-sky-500/20 flex items-center gap-1.5 px-4 py-2"
        >
          <svg
            class="h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          {m.prov_add_btn()}
        </button>
      {:else if activeSubTab === "operatories"}
        <button
          type="button"
          onclick={openAddOperatoryModal}
          class="btn btn-primary text-xs shadow-md shadow-sky-500/20 flex items-center gap-1.5 px-4 py-2"
        >
          <svg
            class="h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          {m.clinic_op_add_btn()}
        </button>
      {:else if activeSubTab === "documents"}
        <button
          type="button"
          onclick={() => triggerUploadDocument && triggerUploadDocument()}
          class="btn btn-primary text-xs shadow-md shadow-sky-500/20 flex items-center gap-1.5 px-4 py-2"
        >
          <svg
            class="h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          {m.doc_btn_upload()}
        </button>
      {/if}
    </div>
  </div>

  <div>
    {#if activeSubTab === "profile"}
      <ClinicProfileSection
        {practiceConfig}
        {countryMeta}
        {isEditingProfile}
        bind:clinicName
        bind:tagline
        bind:taxId
        bind:licenseNumber
        bind:phone
        bind:email
        bind:website
        bind:addressLine1
        bind:city
        bind:stateProvince
        bind:postalCode
        bind:countryCode
        bind:currency
        bind:toothSystem
        bind:dateFormat
      />
    {:else if activeSubTab === "hours"}
      <ClinicHoursSection
        bind:businessHours
        {isEditingProfile}
        {formatTime12}
        {addSlot}
        {removeSlot}
        {addBreak}
        {removeBreak}
        {syncDayBounds}
      />
    {:else if activeSubTab === "providers"}
      <ProvidersSection
        {providers}
        {openAddProviderModal}
        {openEditProviderModal}
        {handleDeleteProvider}
        {handleSaveProvider}
        bind:showProviderModal
        {isEditingProvider}
        bind:provName
        bind:provRole
        bind:provSpecialty
        bind:provLicense
        bind:provEmail
        bind:provPhone
        bind:provColor
        bind:provPin
        bind:provIsActive
        bind:provHourlyRate
      />
    {:else if activeSubTab === "operatories"}
      <OperatoriesSection
        {operatories}
        {openAddOperatoryModal}
        {openEditOperatoryModal}
        {handleDeleteOperatory}
        {handleSaveOperatory}
        bind:showOperatoryModal
        {isEditingOperatory}
        bind:opName
        bind:opRoomCode
        bind:opType
        bind:opIsActive
      />
    {:else if activeSubTab === "documents"}
      <DocumentsSection bind:openUploadModal={triggerUploadDocument} />
    {:else if activeSubTab === "integrations"}
      <IntegrationsSection />
    {/if}
  </div>
</div>
