<script lang="ts">
  import type { CountryConfig } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";

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

  const idLabel = $derived(countryMeta?.national_id_name || "National Identity / Tax ID");
  const idPlaceholder = $derived(countryMeta?.national_id_placeholder || "Enter ID");
  const stateLabel = $derived(countryMeta?.state_province_label || "State / Province");
  const postalLabel = $derived(countryMeta?.postal_code_label || "ZIP / Postal Code");
</script>

{#if showPatientModal}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4" onclick={() => (showPatientModal = false)}>
    <div class="w-full max-w-[620px] rounded-xl border border-slate-700 bg-slate-800 p-6 shadow-2xl" onclick={(e) => e.stopPropagation()}>
      <div class="mb-5 flex items-center justify-between">
        <h2 class="m-0 text-lg text-slate-50">{isEditing ? "Edit Patient" : "Add New Patient"}</h2>
        <button class="cursor-pointer border-none bg-transparent text-2xl text-slate-400 hover:text-white" onclick={() => (showPatientModal = false)}>&times;</button>
      </div>

      <form onsubmit={onsave}>
        <div class="mb-6 grid grid-cols-2 gap-4">
          <div class="flex flex-col gap-1.5">
            <label for="fname" class="text-xs font-medium text-slate-400">First Name *</label>
            <input id="fname" type="text" required bind:value={firstName} placeholder="Jane" class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="lname" class="text-xs font-medium text-slate-400">Last Name *</label>
            <input id="lname" type="text" required bind:value={lastName} placeholder="Smith" class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="national-id" class="text-xs font-medium text-blue-400">{idLabel}</label>
            <input id="national-id" type="text" bind:value={nationalId} placeholder={idPlaceholder} class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="dob" class="text-xs font-medium text-slate-400">Date of Birth</label>
            <input id="dob" type="date" bind:value={dob} class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="email" class="text-xs font-medium text-slate-400">Email Address</label>
            <input id="email" type="email" bind:value={email} placeholder="jane.smith@example.com" class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="phone" class="text-xs font-medium text-slate-400">Phone Primary</label>
            <input id="phone" type="tel" bind:value={phone} placeholder="(555) 019-2834" class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="state-province" class="text-xs font-medium text-slate-400">{stateLabel}</label>
            <input id="state-province" type="text" bind:value={stateProvince} placeholder="e.g. CA, ON, London" class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="postal-code" class="text-xs font-medium text-slate-400">{postalLabel}</label>
            <input id="postal-code" type="text" bind:value={postalCode} placeholder="e.g. 90210, M5V 2T6" class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>

          <div class="col-span-2 flex flex-col gap-1.5">
            <label for="alerts" class="text-xs font-medium text-slate-400">Medical Alerts (comma separated)</label>
            <input id="alerts" type="text" bind:value={medicalAlerts} placeholder="e.g. Penicillin, Latex" class="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none" />
          </div>
        </div>

        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" onclick={() => (showPatientModal = false)}>Cancel</button>
          <button type="submit" class="btn btn-primary">{isEditing ? "Save Changes" : "Save Patient Record"}</button>
        </div>
      </form>
    </div>
  </div>
{/if}
