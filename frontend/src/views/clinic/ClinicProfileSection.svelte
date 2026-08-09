<script lang="ts">
  import type { PracticeConfig, CountryConfig } from "@bindings/domain/models.js";

  let {
    practiceConfig = null,
    countryMeta = null,
    isEditingProfile = false,
    clinicName = $bindable(""),
    tagline = $bindable(""),
    taxId = $bindable(""),
    licenseNumber = $bindable(""),
    phone = $bindable(""),
    email = $bindable(""),
    website = $bindable(""),
    addressLine1 = $bindable(""),
    city = $bindable(""),
    stateProvince = $bindable(""),
    postalCode = $bindable(""),
    countryCode = $bindable(""),
    currency = $bindable(""),
    toothSystem = $bindable(""),
    dateFormat = $bindable(""),
  } = $props<{
    practiceConfig: PracticeConfig | null;
    countryMeta?: CountryConfig | null;
    isEditingProfile: boolean;
    clinicName: string;
    tagline: string;
    taxId: string;
    licenseNumber: string;
    phone: string;
    email: string;
    website: string;
    addressLine1: string;
    city: string;
    stateProvince: string;
    postalCode: string;
    countryCode: string;
    currency: string;
    toothSystem: string;
    dateFormat: string;
  }>();
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
  <!-- General Demographics -->
  <div class="lg:col-span-2 space-y-6">
    <div class="rounded-xl border border-slate-800 bg-slate-900/70 p-6 space-y-4">
      <h3
        class="text-base font-bold text-slate-100 flex items-center gap-2 border-b border-slate-800 pb-3"
      >
        🏢 Clinic Identity & Information
      </h3>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label for="clinic-name" class="block text-xs font-semibold text-slate-400 mb-1"
            >Clinic Name *</label
          >
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
          <label for="clinic-tagline" class="block text-xs font-semibold text-slate-400 mb-1"
            >Tagline / Subtitle</label
          >
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
            {countryMeta?.national_id_type
              ? `${countryMeta.national_id_type.toUpperCase()} / Tax ID`
              : "Tax ID / Business Registration #"}
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
          <label for="clinic-license" class="block text-xs font-semibold text-slate-400 mb-1"
            >Practice License Number</label
          >
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
          <label for="clinic-phone" class="block text-xs font-semibold text-slate-400 mb-1"
            >Primary Phone</label
          >
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
          <label for="clinic-email" class="block text-xs font-semibold text-slate-400 mb-1"
            >Email Address</label
          >
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
          <label for="clinic-website" class="block text-xs font-semibold text-slate-400 mb-1"
            >Website URL</label
          >
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
      <h3
        class="text-base font-bold text-slate-100 flex items-center gap-2 border-b border-slate-800 pb-3"
      >
        📍 Physical Address
      </h3>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div class="sm:col-span-2">
          <label for="addr-1" class="block text-xs font-semibold text-slate-400 mb-1"
            >Street Address</label
          >
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
          <label for="addr-city" class="block text-xs font-semibold text-slate-400 mb-1"
            >City / Municipality</label
          >
          <input
            id="addr-city"
            type="text"
            bind:value={city}
            disabled={!isEditingProfile}
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
          />
        </div>

        <div>
          <label for="addr-state" class="block text-xs font-semibold text-slate-400 mb-1"
            >State / Province / Region</label
          >
          <input
            id="addr-state"
            type="text"
            bind:value={stateProvince}
            disabled={!isEditingProfile}
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed"
          />
        </div>

        <div>
          <label for="addr-zip" class="block text-xs font-semibold text-slate-400 mb-1"
            >ZIP / Postal Code</label
          >
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
      <h3
        class="text-base font-bold text-slate-100 flex items-center gap-2 border-b border-slate-800 pb-3"
      >
        🌐 Regional & Dental Standards
      </h3>

      <div class="space-y-3">
        <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
          <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider"
            >Practice Country</span
          >
          <span class="text-sm font-medium text-slate-200 mt-0.5 flex items-center gap-2">
            <span>{countryMeta?.flag || "📍"}</span>
            <span>{countryMeta?.name || countryCode} ({countryCode})</span>
          </span>
        </div>

        <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
          <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider"
            >Tooth Notation System</span
          >
          <span class="text-sm font-medium text-slate-200 mt-0.5 capitalize">
            {toothSystem === "fdi"
              ? "FDI World Dental Federation Notation (#11 - #48)"
              : toothSystem === "palmer"
                ? "Palmer Notation Method"
                : "Universal Numbering System (#1 - #32)"}
          </span>
        </div>

        <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
          <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider"
            >Default Practice Currency</span
          >
          <span class="text-sm font-medium text-slate-200 mt-0.5 font-mono">
            {currency || "USD"}
          </span>
        </div>

        <div class="rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
          <span class="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider"
            >System Date Format</span
          >
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
