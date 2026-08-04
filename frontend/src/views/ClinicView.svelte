<script lang="ts">
  import type {
    PracticeConfig,
    CountryConfig,
    Provider,
    Operatory,
    BusinessHourDay,
  } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import {
    Gender,
    ProviderRole,
    OperatoryType,
  } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import { PracticeConfigService } from "../../bindings/github.com/LibreDental/libredental/pkg/services/index.js";

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

  let activeSubTab = $state<"profile" | "hours" | "providers" | "operatories">("profile");
  let savingProfile = $state(false);
  let profileMessage = $state<{ text: string; type: "success" | "error" } | null>(null);

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
  let countryCode = $state(practiceConfig?.country_code || "US");
  let currency = $state(practiceConfig?.currency || "USD");
  let toothSystem = $state(practiceConfig?.tooth_system || "universal");
  let dateFormat = $state(practiceConfig?.date_format || "YYYY-MM-DD");
  let businessHours = $state<BusinessHourDay[]>(
    practiceConfig?.business_hours || [
      { day: "Monday", open_time: "08:00", close_time: "17:00", is_closed: false },
      { day: "Tuesday", open_time: "08:00", close_time: "17:00", is_closed: false },
      { day: "Wednesday", open_time: "08:00", close_time: "17:00", is_closed: false },
      { day: "Thursday", open_time: "08:00", close_time: "17:00", is_closed: false },
      { day: "Friday", open_time: "08:00", close_time: "17:00", is_closed: false },
      { day: "Saturday", open_time: "08:00", close_time: "17:00", is_closed: true },
      { day: "Sunday", open_time: "08:00", close_time: "17:00", is_closed: true },
    ]
  );

  // Reactively sync when practiceConfig changes
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
      countryCode = practiceConfig.country_code || "US";
      currency = practiceConfig.currency || "USD";
      toothSystem = practiceConfig.tooth_system || "universal";
      dateFormat = practiceConfig.date_format || "YYYY-MM-DD";
      if (practiceConfig.business_hours && practiceConfig.business_hours.length > 0) {
        businessHours = practiceConfig.business_hours;
      }
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
  let provIsActive = $state(true);

  // Modal / Form state for Operatory
  let showOperatoryModal = $state(false);
  let isEditingOperatory = $state(false);
  let opId = $state("");
  let opName = $state("");
  let opRoomCode = $state("");
  let opType = $state<string>("general");
  let opIsActive = $state(true);

  let isEditingProfile = $state(false);
  let toastTimeout: any = null;

  function setProfileMessage(text: string, type: "success" | "error") {
    if (toastTimeout) clearTimeout(toastTimeout);
    profileMessage = { text, type };
    toastTimeout = setTimeout(() => {
      profileMessage = null;
    }, 3000);
  }

  function selectSubTab(tab: "profile" | "hours" | "providers" | "operatories") {
    activeSubTab = tab;
    if (toastTimeout) clearTimeout(toastTimeout);
    profileMessage = null;
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
      countryCode = practiceConfig.country_code || "US";
      currency = practiceConfig.currency || "USD";
      toothSystem = practiceConfig.tooth_system || "universal";
      dateFormat = practiceConfig.date_format || "YYYY-MM-DD";
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

  function handleCountryChange(code: string) {
    countryCode = code as any;
    const found = supportedCountries.find((c: CountryConfig) => c.code === code);
    if (found) {
      currency = found.default_currency;
      toothSystem = found.default_tooth_system;
      dateFormat = found.date_format;
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
    provIsActive = true;
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
    provIsActive = p.is_active;
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
        is_active: provIsActive,
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
    if (confirm("Are you sure you want to delete this provider?")) {
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
    if (confirm("Are you sure you want to delete this operatory?")) {
      try {
        await PracticeConfigService.DeleteOperatory(id);
        await onrefresh();
      } catch (err) {
        console.error("Failed to delete operatory:", err);
      }
    }
  }
</script>

<div class="space-y-6">
  <!-- Sub Navigation Tabs with Edit / Save / Cancel controls & Minimalist Notification -->
  <div class="flex flex-wrap items-center justify-between border-b border-slate-800 pb-2 gap-4">
    <div class="flex border-b border-transparent gap-2">
      <button
        type="button"
        onclick={() => selectSubTab("profile")}
        class={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-colors ${
          activeSubTab === "profile"
            ? "border-sky-400 text-sky-400"
            : "border-transparent text-slate-400 hover:text-slate-200"
        }`}
      >
        Practice Profile & Standards
      </button>
      <button
        type="button"
        onclick={() => selectSubTab("hours")}
        class={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-colors ${
          activeSubTab === "hours"
            ? "border-sky-400 text-sky-400"
            : "border-transparent text-slate-400 hover:text-slate-200"
        }`}
      >
        Operating Hours
      </button>
      <button
        type="button"
        onclick={() => selectSubTab("providers")}
        class={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-colors ${
          activeSubTab === "providers"
            ? "border-sky-400 text-sky-400"
            : "border-transparent text-slate-400 hover:text-slate-200"
        }`}
      >
        Providers & Staff ({providers.length})
      </button>
      <button
        type="button"
        onclick={() => selectSubTab("operatories")}
        class={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-colors ${
          activeSubTab === "operatories"
            ? "border-sky-400 text-sky-400"
            : "border-transparent text-slate-400 hover:text-slate-200"
        }`}
      >
        Operatories & Chairs ({operatories.length})
      </button>
    </div>

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
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
              </svg>
              <span>Edit Practice Info</span>
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
                <div class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
                <span>Saving...</span>
              {:else}
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                  <polyline points="17 21 17 13 7 13 7 21" />
                  <polyline points="7 3 7 8 15 8" />
                </svg>
                <span>Save Changes</span>
              {/if}
            </button>
          {/if}
        </div>
      {/if}
    </div>
  </div>

  <!-- TAB 1: PRACTICE PROFILE & STANDARDS -->
  {#if activeSubTab === "profile"}
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- General Demographics -->
      <div class="lg:col-span-2 space-y-6">
        <div class="rounded-xl border border-slate-800 bg-slate-900/70 p-6 space-y-4">
          <h3 class="text-base font-bold text-slate-100 flex items-center gap-2 border-b border-slate-800 pb-3">
            🏢 Clinic Identity & Information
          </h3>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label for="clinic-name" class="block text-xs font-semibold text-slate-400 mb-1">Clinic Name *</label>
              <input
                id="clinic-name"
                type="text"
                bind:value={clinicName}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. Bright Smile Dental Clinic"
              />
            </div>
            <div>
              <label for="clinic-tagline" class="block text-xs font-semibold text-slate-400 mb-1">Tagline / Subtitle</label>
              <input
                id="clinic-tagline"
                type="text"
                bind:value={tagline}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. Comprehensive Family & Cosmetic Dentistry"
              />
            </div>

            <div>
              <label for="clinic-tax-id" class="block text-xs font-semibold text-slate-400 mb-1">
                {countryMeta?.national_id_type ? `${countryMeta.national_id_type.toUpperCase()} / Tax ID` : "Tax ID / Business Registration #"}
              </label>
              <input
                id="clinic-tax-id"
                type="text"
                bind:value={taxId}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. 12-3456789"
              />
            </div>

            <div>
              <label for="clinic-license" class="block text-xs font-semibold text-slate-400 mb-1">Practice License Number</label>
              <input
                id="clinic-license"
                type="text"
                bind:value={licenseNumber}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. DEN-LIC-8829"
              />
            </div>

            <div>
              <label for="clinic-phone" class="block text-xs font-semibold text-slate-400 mb-1">Primary Phone</label>
              <input
                id="clinic-phone"
                type="text"
                bind:value={phone}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. +1 (555) 234-5678"
              />
            </div>

            <div>
              <label for="clinic-email" class="block text-xs font-semibold text-slate-400 mb-1">Email Address</label>
              <input
                id="clinic-email"
                type="email"
                bind:value={email}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. contact@brightsmiledental.com"
              />
            </div>

            <div class="sm:col-span-2">
              <label for="clinic-website" class="block text-xs font-semibold text-slate-400 mb-1">Website URL</label>
              <input
                id="clinic-website"
                type="url"
                bind:value={website}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. https://www.brightsmiledental.com"
              />
            </div>
          </div>
        </div>

        <!-- Address Information -->
        <div class="rounded-xl border border-slate-800 bg-slate-900/70 p-6 space-y-4">
          <h3 class="text-base font-bold text-slate-100 flex items-center gap-2 border-b border-slate-800 pb-3">
            📍 Physical Address
          </h3>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="sm:col-span-2">
              <label for="addr-1" class="block text-xs font-semibold text-slate-400 mb-1">Street Address</label>
              <input
                id="addr-1"
                type="text"
                bind:value={addressLine1}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
                placeholder="e.g. 100 Main Street, Suite 400"
              />
            </div>

            <div>
              <label for="addr-city" class="block text-xs font-semibold text-slate-400 mb-1">City / Municipality</label>
              <input
                id="addr-city"
                type="text"
                bind:value={city}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
              />
            </div>

            <div>
              <label for="addr-state" class="block text-xs font-semibold text-slate-400 mb-1">State / Province / Region</label>
              <input
                id="addr-state"
                type="text"
                bind:value={stateProvince}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
              />
            </div>

            <div>
              <label for="addr-zip" class="block text-xs font-semibold text-slate-400 mb-1">ZIP / Postal Code</label>
              <input
                id="addr-zip"
                type="text"
                bind:value={postalCode}
                disabled={!isEditingProfile}
                class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- Regional & Practice Standards (Read-Only) -->
      <div class="space-y-6">
        <div class="rounded-xl border border-slate-800 bg-slate-900/70 p-6 space-y-4">
          <h3 class="text-base font-bold text-slate-100 flex items-center gap-2 border-b border-slate-800 pb-3">
            🌐 Regional & Dental Standards
          </h3>

          <div class="space-y-3">
            <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
              <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Practice Country</span>
              <span class="text-sm font-medium text-slate-200 mt-0.5 flex items-center gap-2">
                <span>{countryMeta?.flag || "📍"}</span>
                <span>{countryMeta?.name || countryCode} ({countryCode})</span>
              </span>
            </div>

            <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
              <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Tooth Notation System</span>
              <span class="text-sm font-medium text-slate-200 mt-0.5 capitalize">
                {toothSystem === "fdi" ? "FDI World Dental Federation Notation (#11 - #48)" : toothSystem === "palmer" ? "Palmer Notation Method" : "Universal Numbering System (#1 - #32)"}
              </span>
            </div>

            <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
              <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Default Practice Currency</span>
              <span class="text-sm font-medium text-slate-200 mt-0.5 font-mono">
                {currency || "USD"}
              </span>
            </div>

            <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
              <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider">System Date Format</span>
              <span class="text-sm font-medium text-slate-200 mt-0.5 font-mono">
                {dateFormat || "YYYY-MM-DD"}
              </span>
            </div>
          </div>
          <p class="text-[11px] text-slate-500 italic">
            Regional standards are established during setup to preserve clinical record integrity.
          </p>
        </div>
      </div>
    </div>
  {/if}

  <!-- TAB 2: OPERATING HOURS -->
  {#if activeSubTab === "hours"}
    <div class="rounded-xl border border-slate-800 bg-slate-900/70 p-6 space-y-6">
      <div class="flex items-center justify-between border-b border-slate-800 pb-4">
        <div>
          <h3 class="text-lg font-bold text-slate-100">⏰ Practice Operating Schedule</h3>
          <p class="text-xs text-slate-400 mt-0.5">Configure your weekly operating hours for appointment scheduling.</p>
        </div>
      </div>

      <div class="space-y-3">
        {#each businessHours as hour, idx}
          <div class={`flex flex-wrap items-center justify-between gap-4 p-4 rounded-xl border transition-colors ${
            hour.is_closed ? "border-slate-800/60 bg-slate-950/40 opacity-70" : "border-slate-800 bg-slate-950/80"
          }`}>
            <div class="w-32 flex items-center gap-3">
              <input
                type="checkbox"
                id={`closed-${idx}`}
                bind:checked={hour.is_closed}
                disabled={!isEditingProfile}
                class="rounded border-slate-700 text-sky-500 focus:ring-sky-500 h-4 w-4 disabled:opacity-50 disabled:cursor-not-allowed"
              />
              <label for={`closed-${idx}`} class="text-sm font-semibold text-slate-200 cursor-pointer select-none">
                {hour.day}
              </label>
            </div>

            {#if hour.is_closed}
              <span class="text-xs font-semibold text-rose-400 bg-rose-500/10 border border-rose-500/20 px-3 py-1 rounded-full">
                CLOSED
              </span>
            {:else}
              <div class="flex items-center gap-3 text-xs font-medium text-slate-300">
                <span>Opens:</span>
                <input
                  type="time"
                  bind:value={hour.open_time}
                  disabled={!isEditingProfile}
                  class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
                />
                <span>Closes:</span>
                <input
                  type="time"
                  bind:value={hour.close_time}
                  disabled={!isEditingProfile}
                  class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
                />
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- TAB 3: PROVIDERS & STAFF -->
  {#if activeSubTab === "providers"}
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-lg font-bold text-slate-100">👨‍⚕️ Practice Staff & Providers</h3>
          <p class="text-xs text-slate-400 mt-0.5">Manage dentists, hygienists, and support staff assigned to patient appointments.</p>
        </div>
        <button
          type="button"
          onclick={openAddProviderModal}
          class="btn btn-primary shadow-md shadow-sky-500/20 text-xs flex items-center gap-1.5"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          Add Provider / Staff
        </button>
      </div>

      {#if providers.length === 0}
        <div class="rounded-2xl border border-dashed border-slate-800 bg-slate-900/40 py-12 text-center">
          <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-800 text-slate-400 mb-3">
            👨‍⚕️
          </div>
          <p class="text-base font-semibold text-slate-300">No staff members or providers added yet</p>
          <p class="text-xs text-slate-500 mt-1">Click "Add Provider / Staff" above to configure practice dentists and hygienists.</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each providers as p}
            <div class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 relative group hover:border-slate-700 transition-colors">
              <div class="flex items-start justify-between">
                <div class="flex items-center gap-3">
                  <div
                    class="h-10 w-10 rounded-full flex items-center justify-center text-white font-bold text-sm shadow-md"
                    style="background-color: {p.color || '#3b82f6'};"
                  >
                    {p.name.charAt(0)}
                  </div>
                  <div>
                    <h4 class="text-sm font-bold text-slate-100">{p.name}</h4>
                    <p class="text-xs text-sky-400 capitalize font-medium">{p.role} {p.specialty ? `• ${p.specialty}` : ""}</p>
                  </div>
                </div>
                <span class={`px-2 py-0.5 text-[10px] font-bold rounded-full uppercase ${p.is_active ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" : "bg-slate-800 text-slate-500"}`}>
                  {p.is_active ? "Active" : "Inactive"}
                </span>
              </div>

              {#if p.license_number || p.email || p.phone}
                <div class="text-xs text-slate-400 space-y-1 pt-2 border-t border-slate-800">
                  {#if p.license_number}
                    <div>🪪 License: <span class="text-slate-300 font-mono">{p.license_number}</span></div>
                  {/if}
                  {#if p.email}
                    <div>✉️ {p.email}</div>
                  {/if}
                  {#if p.phone}
                    <div>📞 {p.phone}</div>
                  {/if}
                </div>
              {/if}

              <div class="flex items-center justify-end gap-2 pt-2 border-t border-slate-800/60 text-xs">
                <button
                  type="button"
                  onclick={() => openEditProviderModal(p)}
                  class="text-sky-400 hover:text-sky-300 font-semibold"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onclick={() => handleDeleteProvider(p.id)}
                  class="text-rose-400 hover:text-rose-300 font-semibold"
                >
                  Delete
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  <!-- TAB 4: OPERATORIES & CHAIRS -->
  {#if activeSubTab === "operatories"}
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-lg font-bold text-slate-100">🚪 Treatment Operatories & Chairs</h3>
          <p class="text-xs text-slate-400 mt-0.5">Manage treatment rooms, hygiene bays, and surgical suites.</p>
        </div>
        <button
          type="button"
          onclick={openAddOperatoryModal}
          class="btn btn-primary shadow-md shadow-sky-500/20 text-xs flex items-center gap-1.5"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          Add Operatory / Room
        </button>
      </div>

      {#if operatories.length === 0}
        <div class="rounded-2xl border border-dashed border-slate-800 bg-slate-900/40 py-12 text-center">
          <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-800 text-slate-400 mb-3">
            🚪
          </div>
          <p class="text-base font-semibold text-slate-300">No operatories or treatment rooms configured</p>
          <p class="text-xs text-slate-500 mt-1">Click "Add Operatory / Room" above to set up treatment chairs for scheduling.</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each operatories as op}
            <div class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 relative group hover:border-slate-700 transition-colors">
              <div class="flex items-start justify-between">
                <div>
                  <h4 class="text-sm font-bold text-slate-100">{op.name}</h4>
                  <p class="text-xs text-sky-400 capitalize font-medium mt-0.5">
                    {op.type} room {op.room_code ? `• Code: ${op.room_code}` : ""}
                  </p>
                </div>
                <span class={`px-2 py-0.5 text-[10px] font-bold rounded-full uppercase ${op.is_active ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" : "bg-slate-800 text-slate-500"}`}>
                  {op.is_active ? "Active" : "Inactive"}
                </span>
              </div>

              <div class="flex items-center justify-end gap-2 pt-2 border-t border-slate-800/60 text-xs">
                <button
                  type="button"
                  onclick={() => openEditOperatoryModal(op)}
                  class="text-sky-400 hover:text-sky-300 font-semibold"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onclick={() => handleDeleteOperatory(op.id)}
                  class="text-rose-400 hover:text-rose-300 font-semibold"
                >
                  Delete
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- PROVIDER MODAL -->
{#if showProviderModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4 backdrop-blur-sm">
    <div class="w-full max-w-md rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl space-y-5">
      <div class="flex items-center justify-between border-b border-slate-800 pb-3">
        <h3 class="text-lg font-bold text-white">
          {isEditingProvider ? "Edit Staff / Provider" : "Add New Staff / Provider"}
        </h3>
        <button
          type="button"
          onclick={() => (showProviderModal = false)}
          class="text-slate-400 hover:text-white text-lg font-bold"
        >
          ✕
        </button>
      </div>

      <form onsubmit={handleSaveProvider} class="space-y-4">
        <div>
          <label for="prov-name" class="block text-xs font-semibold text-slate-400 mb-1">Full Name *</label>
          <input
            id="prov-name"
            type="text"
            bind:value={provName}
            required
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
            placeholder="e.g. Dr. Sarah Smith"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="prov-role" class="block text-xs font-semibold text-slate-400 mb-1">Role</label>
            <select
              id="prov-role"
              bind:value={provRole}
              class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
            >
              <option value="dentist">Dentist</option>
              <option value="hygienist">Dental Hygienist</option>
              <option value="assistant">Dental Assistant</option>
              <option value="staff">Administrative Staff</option>
            </select>
          </div>

          <div>
            <label for="prov-specialty" class="block text-xs font-semibold text-slate-400 mb-1">Specialty</label>
            <input
              id="prov-specialty"
              type="text"
              bind:value={provSpecialty}
              class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
              placeholder="e.g. Orthodontics"
            />
          </div>
        </div>

        <div>
          <label for="prov-license" class="block text-xs font-semibold text-slate-400 mb-1">License Number</label>
          <input
            id="prov-license"
            type="text"
            bind:value={provLicense}
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
            placeholder="e.g. DENT-88912"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="prov-email" class="block text-xs font-semibold text-slate-400 mb-1">Email</label>
            <input
              id="prov-email"
              type="email"
              bind:value={provEmail}
              class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
            />
          </div>

          <div>
            <label for="prov-phone" class="block text-xs font-semibold text-slate-400 mb-1">Phone</label>
            <input
              id="prov-phone"
              type="text"
              bind:value={provPhone}
              class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
            />
          </div>
        </div>

        <div class="flex items-center justify-between pt-2">
          <div>
            <label for="prov-color" class="block text-xs font-semibold text-slate-400 mb-1">Schedule Color Badge</label>
            <input
              id="prov-color"
              type="color"
              bind:value={provColor}
              class="h-9 w-16 cursor-pointer rounded border border-slate-700 bg-slate-950 p-1"
            />
          </div>

          <div class="flex items-center gap-2 pt-4">
            <input
              type="checkbox"
              id="prov-active"
              bind:checked={provIsActive}
              class="rounded border-slate-700 text-sky-500 focus:ring-sky-500 h-4 w-4"
            />
            <label for="prov-active" class="text-xs font-semibold text-slate-300 cursor-pointer">Active Provider</label>
          </div>
        </div>

        <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
          <button
            type="button"
            onclick={() => (showProviderModal = false)}
            class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white"
          >
            Cancel
          </button>
          <button type="submit" class="btn btn-primary text-xs px-5 py-2">
            Save Staff Member
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- OPERATORY MODAL -->
{#if showOperatoryModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4 backdrop-blur-sm">
    <div class="w-full max-w-md rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl space-y-5">
      <div class="flex items-center justify-between border-b border-slate-800 pb-3">
        <h3 class="text-lg font-bold text-white">
          {isEditingOperatory ? "Edit Treatment Operatory" : "Add New Operatory"}
        </h3>
        <button
          type="button"
          onclick={() => (showOperatoryModal = false)}
          class="text-slate-400 hover:text-white text-lg font-bold"
        >
          ✕
        </button>
      </div>

      <form onsubmit={handleSaveOperatory} class="space-y-4">
        <div>
          <label for="op-name" class="block text-xs font-semibold text-slate-400 mb-1">Operatory Name *</label>
          <input
            id="op-name"
            type="text"
            bind:value={opName}
            required
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
            placeholder="e.g. Operatory 1 (General Dentistry)"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="op-code" class="block text-xs font-semibold text-slate-400 mb-1">Room Code / ID</label>
            <input
              id="op-code"
              type="text"
              bind:value={opRoomCode}
              class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
              placeholder="e.g. ROOM-A"
            />
          </div>

          <div>
            <label for="op-type" class="block text-xs font-semibold text-slate-400 mb-1">Type</label>
            <select
              id="op-type"
              bind:value={opType}
              class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
            >
              <option value="general">General Dentistry</option>
              <option value="hygiene">Hygiene Bay</option>
              <option value="surgery">Oral Surgery Suite</option>
            </select>
          </div>
        </div>

        <div class="flex items-center gap-2 pt-2">
          <input
            type="checkbox"
            id="op-active"
            bind:checked={opIsActive}
            class="rounded border-slate-700 text-sky-500 focus:ring-sky-500 h-4 w-4"
          />
          <label for="op-active" class="text-xs font-semibold text-slate-300 cursor-pointer">Active Operatory</label>
        </div>

        <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
          <button
            type="button"
            onclick={() => (showOperatoryModal = false)}
            class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white"
          >
            Cancel
          </button>
          <button type="submit" class="btn btn-primary text-xs px-5 py-2">
            Save Operatory
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
