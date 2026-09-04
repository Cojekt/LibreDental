<script lang="ts">
  import type { CountryConfig, Provider } from "@bindings/domain/models.js";
  import Modal from "./ui/Modal.svelte";
  import FormField from "./ui/FormField.svelte";
  import Input from "./ui/Input.svelte";
  import EmailInput from "./ui/EmailInput.svelte";
  import PhoneInput from "./ui/PhoneInput.svelte";
  import IdInput from "./ui/IdInput.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";

  let {
    showPatientModal = $bindable(),
    isEditing,
    firstName = $bindable(),
    lastName = $bindable(),
    sex = $bindable(),
    dob = $bindable(),
    email = $bindable(),
    phone = $bindable(),
    phoneSecondary = $bindable(),
    nationalId = $bindable(),
    addressLine1 = $bindable(),
    addressLine2 = $bindable(),
    city = $bindable(),
    stateProvince = $bindable(),
    postalCode = $bindable(),
    emergencyName = $bindable(),
    emergencyRel = $bindable(),
    emergencyPhone = $bindable(),
    guarantorName = $bindable(),
    guarantorRel = $bindable(),
    guarantorPhone = $bindable(),
    insuranceCarrier = $bindable(),
    insurancePolicy = $bindable(),
    insuranceGroup = $bindable(),
    insuranceIsSubscriber = $bindable(),
    insuranceSubscriberId = $bindable(),
    preferredContactMethod = $bindable(),
    preferredLanguage = $bindable(),
    reminderOptIn = $bindable(),
    preferredProviderId = $bindable(),
    referralSource = $bindable(),
    medicalAlerts = $bindable(),
    countryMeta,
    configuredProviders = [],
    onsave,
  } = $props<{
    showPatientModal: boolean;
    isEditing: boolean;
    firstName: string;
    lastName: string;
    sex: string;
    dob: string;
    email: string;
    phone: string;
    phoneSecondary: string;
    nationalId: string;
    addressLine1: string;
    addressLine2: string;
    city: string;
    stateProvince: string;
    postalCode: string;
    emergencyName: string;
    emergencyRel: string;
    emergencyPhone: string;
    guarantorName: string;
    guarantorRel: string;
    guarantorPhone: string;
    insuranceCarrier: string;
    insurancePolicy: string;
    insuranceGroup: string;
    insuranceIsSubscriber: boolean;
    insuranceSubscriberId: string;
    preferredContactMethod: string;
    preferredLanguage: string;
    reminderOptIn: boolean;
    preferredProviderId: string;
    referralSource: string;
    medicalAlerts: string;
    countryMeta?: CountryConfig | null;
    configuredProviders?: Provider[];
    onsave: (e: Event) => void;
  }>();

  const idLabel = $derived(countryMeta?.national_id_name || "National ID / SSN");
  const idPlaceholder = $derived(countryMeta?.national_id_placeholder || "000-00-0000");
  const stateLabel = $derived(countryMeta?.state_province_label || "State / Province");
  const postalLabel = $derived(countryMeta?.postal_code_label || "Postal Code");

  const modalTitle = $derived.by(() => {
    getLocaleVersion();
    return isEditing ? m.patient_modal_edit_title() : m.patient_modal_add_title();
  });
  const modalSubtitle = $derived.by(() => {
    getLocaleVersion();
    return m.patient_modal_subtitle();
  });
</script>

<Modal
  bind:showModal={showPatientModal}
  title={modalTitle}
  subtitle={modalSubtitle}
  maxWidth="max-w-3xl"
>
  <form onsubmit={onsave} class="space-y-6">
    <!-- Section 1: Demographics -->
    <div>
      <h3
        class="text-xs font-bold uppercase tracking-wider text-sky-400 mb-3 flex items-center gap-1.5"
      >
        Patient Identity & Demographics
      </h3>
      <div
        class="grid grid-cols-1 sm:grid-cols-3 gap-4 bg-slate-900/60 p-4 rounded-xl border border-slate-800"
      >
        <FormField label={m.patient_first_name()} forId="fname" required>
          <Input
            id="fname"
            type="text"
            required
            bind:value={firstName}
            placeholder={m.patient_placeholder_fname()}
          />
        </FormField>

        <FormField label={m.patient_last_name()} forId="lname" required>
          <Input
            id="lname"
            type="text"
            required
            bind:value={lastName}
            placeholder={m.patient_placeholder_lname()}
          />
        </FormField>

        <FormField label={m.patient_sex()} forId="sex">
          <select
            id="sex"
            bind:value={sex}
            class="w-full rounded-xl border border-slate-700 bg-slate-950/80 px-4 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
          >
            <option value="male">{(getLocaleVersion(), m.sex_male())}</option>
            <option value="female">{m.sex_female()}</option>
            <option value="other">{m.sex_other()}</option>
            <option value="undisclosed">{m.sex_undisclosed()}</option>
          </select>
        </FormField>

        <FormField label={m.patient_dob()} forId="dob" required>
          <Input
            id="dob"
            type="date"
            required
            bind:value={dob}
            dateFormat={countryMeta?.date_format}
          />
        </FormField>

        <FormField label={idLabel} forId="national-id">
          <IdInput
            id="national-id"
            bind:value={nationalId}
            placeholder={idPlaceholder}
            {countryMeta}
          />
        </FormField>

        <FormField label={m.patient_pref_provider()} forId="pref-provider">
          <select
            id="pref-provider"
            bind:value={preferredProviderId}
            class="w-full rounded-xl border border-slate-700 bg-slate-950/80 px-4 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
          >
            <option value="">-- Unassigned --</option>
            {#each configuredProviders as prov}
              <option value={prov.id}>{prov.name} ({prov.specialty || prov.role})</option>
            {/each}
          </select>
        </FormField>
      </div>
    </div>

    <!-- Section 2: Contact & Address -->
    <div>
      <h3
        class="text-xs font-bold uppercase tracking-wider text-sky-400 mb-3 flex items-center gap-1.5"
      >
        Contact Details & Location
      </h3>
      <div
        class="grid grid-cols-1 sm:grid-cols-3 gap-4 bg-slate-900/60 p-4 rounded-xl border border-slate-800"
      >
        <FormField label={m.patient_phone()} forId="phone" required>
          <PhoneInput id="phone" required bind:value={phone} placeholder="555-019-2834" />
        </FormField>

        <FormField label={m.patient_phone_secondary()} forId="phone-sec">
          <PhoneInput id="phone-sec" bind:value={phoneSecondary} placeholder="555-019-9988" />
        </FormField>

        <FormField label={m.patient_email()} forId="email">
          <EmailInput id="email" bind:value={email} placeholder="jane.smith@example.com" />
        </FormField>

        <div class="sm:col-span-2">
          <FormField label={m.patient_address_line1()} forId="addr1">
            <Input
              id="addr1"
              type="text"
              bind:value={addressLine1}
              placeholder="742 Evergreen Terrace"
            />
          </FormField>
        </div>

        <FormField label={m.patient_address_line2()} forId="addr2">
          <Input
            id="addr2"
            type="text"
            bind:value={addressLine2}
            placeholder={m.patient_placeholder_addr2()}
          />
        </FormField>

        <FormField label={m.patient_city()} forId="city">
          <Input
            id="city"
            type="text"
            bind:value={city}
            placeholder={m.patient_placeholder_city()}
          />
        </FormField>

        <FormField label={stateLabel} forId="state-province">
          <Input
            id="state-province"
            type="text"
            bind:value={stateProvince}
            placeholder="e.g. CA, ON, London"
          />
        </FormField>

        <FormField label={postalLabel} forId="postal-code">
          <Input
            id="postal-code"
            type="text"
            bind:value={postalCode}
            placeholder="e.g. 90210, M5V 2T6"
          />
        </FormField>

        <FormField label={m.patient_pref_contact_method()} forId="pref-method">
          <select
            id="pref-method"
            bind:value={preferredContactMethod}
            class="w-full rounded-xl border border-slate-700 bg-slate-950/80 px-4 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
          >
            <option value="phone">{(getLocaleVersion(), m.patient_pref_method_phone())}</option>
            <option value="sms">{m.patient_pref_method_sms()}</option>
            <option value="email">{m.patient_pref_method_email()}</option>
          </select>
        </FormField>

        <FormField label={m.patient_pref_referral()} forId="referral">
          <Input
            id="referral"
            type="text"
            bind:value={referralSource}
            placeholder="e.g. Doctor Referral, Online"
          />
        </FormField>

        <div class="flex items-center pt-6">
          <label
            class="flex items-center gap-2.5 cursor-pointer text-sm text-slate-200 select-none"
          >
            <input type="checkbox" bind:checked={reminderOptIn} />
            <span>{m.patient_pref_reminder_opt_in()}</span>
          </label>
        </div>
      </div>
    </div>

    <!-- Section 3: Emergency Contact & Guarantor -->
    <div>
      <h3
        class="text-xs font-bold uppercase tracking-wider text-sky-400 mb-3 flex items-center gap-1.5"
      >
        Emergency Contact & Billing Guarantor
      </h3>
      <div
        class="grid grid-cols-1 sm:grid-cols-3 gap-4 bg-slate-900/60 p-4 rounded-xl border border-slate-800"
      >
        <FormField label={m.patient_emergency_name()} forId="emerg-name">
          <Input
            id="emerg-name"
            type="text"
            bind:value={emergencyName}
            placeholder={m.patient_placeholder_emer_name()}
          />
        </FormField>
        <FormField label={m.patient_emergency_rel()} forId="emerg-rel">
          <Input
            id="emerg-rel"
            type="text"
            bind:value={emergencyRel}
            placeholder="e.g. Spouse, Parent"
          />
        </FormField>
        <FormField label={m.patient_emergency_phone()} forId="emerg-phone">
          <PhoneInput id="emerg-phone" bind:value={emergencyPhone} placeholder="555-999-8877" />
        </FormField>

        <FormField label={m.patient_guarantor_name()} forId="guar-name">
          <Input
            id="guar-name"
            type="text"
            bind:value={guarantorName}
            placeholder={m.patient_placeholder_guar_name()}
          />
        </FormField>
        <FormField label={m.patient_guarantor_rel()} forId="guar-rel">
          <Input
            id="guar-rel"
            type="text"
            bind:value={guarantorRel}
            placeholder={m.patient_placeholder_guar_rel()}
          />
        </FormField>
        <FormField label={m.patient_guarantor_phone()} forId="guar-phone">
          <PhoneInput id="guar-phone" bind:value={guarantorPhone} placeholder="555-333-2211" />
        </FormField>
      </div>
    </div>

    <!-- Section 4: Primary Insurance & Preferences -->
    <div>
      <h3
        class="text-xs font-bold uppercase tracking-wider text-sky-400 mb-3 flex items-center gap-1.5"
      >
        Dental Insurance & Preferences
      </h3>
      <div
        class="grid grid-cols-1 sm:grid-cols-3 gap-4 bg-slate-900/60 p-4 rounded-xl border border-slate-800"
      >
        <FormField label={m.patient_insurance_carrier()} forId="ins-carrier">
          <Input
            id="ins-carrier"
            type="text"
            bind:value={insuranceCarrier}
            placeholder="e.g. Delta Dental PPO"
          />
        </FormField>
        <FormField label={m.patient_insurance_policy()} forId="ins-policy">
          <Input
            id="ins-policy"
            type="text"
            bind:value={insurancePolicy}
            placeholder={m.patient_placeholder_ins_policy()}
          />
        </FormField>
        <FormField label={m.patient_insurance_group()} forId="ins-group">
          <Input
            id="ins-group"
            type="text"
            bind:value={insuranceGroup}
            placeholder={m.patient_placeholder_ins_group()}
          />
        </FormField>

        <div class="flex items-center pt-6">
          <label
            class="flex items-center gap-2.5 cursor-pointer text-sm text-slate-200 select-none"
          >
            <input type="checkbox" bind:checked={insuranceIsSubscriber} />
            <span>Subscriber (Head of Family)</span>
          </label>
        </div>

        {#if !insuranceIsSubscriber}
          <FormField label="Subscriber ID" forId="ins-subscriber">
            <Input
              id="ins-subscriber"
              type="text"
              bind:value={insuranceSubscriberId}
              placeholder="e.g. pat_123456"
            />
          </FormField>
        {/if}
      </div>
    </div>

    <!-- Section 5: Medical Alerts -->
    <div>
      <FormField label={m.patient_medical_alerts()} forId="alerts">
        <Input
          id="alerts"
          type="text"
          bind:value={medicalAlerts}
          placeholder="e.g. Penicillin Allergy, Latex Allergy, High Blood Pressure"
        />
      </FormField>
    </div>

    <div class="flex justify-end gap-3 border-t border-slate-800 pt-4">
      <button
        type="button"
        class="btn btn-secondary cursor-pointer"
        onclick={() => (showPatientModal = false)}
      >
        {m.common_cancel()}
      </button>
      <button type="submit" class="btn btn-primary cursor-pointer">
        {isEditing ? m.patient_save_changes() : m.patient_save_record()}
      </button>
    </div>
  </form>
</Modal>
