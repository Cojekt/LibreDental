<script lang="ts">
  import type { CountryConfig } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import Modal from "./ui/Modal.svelte";
  import FormField from "./ui/FormField.svelte";
  import Input from "./ui/Input.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../lib/locale.svelte.js";

  let {
    showPatientModal = $bindable(),
    isEditing,
    firstName = $bindable(),
    lastName = $bindable(),
    email = $bindable(),
    phone = $bindable(),
    dob = $bindable(),
    nationalId = $bindable(),
    stateProvince = $bindable(),
    postalCode = $bindable(),
    medicalAlerts = $bindable(),
    countryMeta,
    onsave,
  } = $props<{
    showPatientModal: boolean;
    isEditing: boolean;
    firstName: string;
    lastName: string;
    email: string;
    phone: string;
    dob: string;
    nationalId: string;
    stateProvince: string;
    postalCode: string;
    medicalAlerts: string;
    countryMeta?: CountryConfig | null;
    onsave: (e: Event) => void;
  }>();

  const idLabel = $derived(countryMeta?.national_id_name || "");
  const idPlaceholder = $derived(countryMeta?.national_id_placeholder || "");
  const stateLabel = $derived(countryMeta?.state_province_label || "");
  const postalLabel = $derived(countryMeta?.postal_code_label || "");

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
  icon="👤"
  maxWidth="max-w-2xl"
>
  <form onsubmit={onsave}>
    <div class="mb-6 grid grid-cols-1 sm:grid-cols-2 gap-4">
      <FormField label={m.patient_first_name()} forId="fname" required>
        <Input id="fname" type="text" required bind:value={firstName} placeholder="Jane" />
      </FormField>

      <FormField label={m.patient_last_name()} forId="lname" required>
        <Input id="lname" type="text" required bind:value={lastName} placeholder="Smith" />
      </FormField>

      <FormField label={idLabel} forId="national-id">
        <Input id="national-id" type="text" bind:value={nationalId} placeholder={idPlaceholder} />
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

      <FormField label={m.patient_email()} forId="email">
        <Input id="email" type="email" bind:value={email} placeholder="jane.smith@example.com" />
      </FormField>

      <FormField label={m.patient_phone()} forId="phone" required>
        <Input id="phone" type="tel" required bind:value={phone} placeholder="(555) 019-2834" />
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

      <div class="sm:col-span-2">
        <FormField label={m.patient_medical_alerts()} forId="alerts">
          <Input
            id="alerts"
            type="text"
            bind:value={medicalAlerts}
            placeholder="e.g. Penicillin, Latex"
          />
        </FormField>
      </div>
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
