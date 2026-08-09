<script lang="ts">
  import type { Provider } from "@bindings/domain/models.js";
  import Modal from "../../components/ui/Modal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";

  let {
    providers = [],
    openAddProviderModal,
    openEditProviderModal,
    handleDeleteProvider,
    handleSaveProvider,
    showProviderModal = $bindable(false),
    isEditingProvider = false,
    provName = $bindable(""),
    provRole = $bindable("dentist"),
    provSpecialty = $bindable(""),
    provLicense = $bindable(""),
    provEmail = $bindable(""),
    provPhone = $bindable(""),
    provColor = $bindable("#3b82f6"),
    provIsActive = $bindable(true),
  } = $props<{
    providers: Provider[];
    openAddProviderModal: () => void;
    openEditProviderModal: (p: Provider) => void;
    handleDeleteProvider: (id: string) => void;
    handleSaveProvider: (e: Event) => void;
    showProviderModal: boolean;
    isEditingProvider: boolean;
    provName: string;
    provRole: string;
    provSpecialty: string;
    provLicense: string;
    provEmail: string;
    provPhone: string;
    provColor: string;
    provIsActive: boolean;
  }>();
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <div>
      <h3 class="text-lg font-bold text-slate-100">👨‍⚕️ Practice Staff & Providers</h3>
      <p class="text-xs text-slate-400 mt-0.5">
        Manage dentists, hygienists, and support staff assigned to patient appointments.
      </p>
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
    <EmptyState
      title="No staff members or providers added yet"
      subtitle="Click 'Add Provider / Staff' above to configure practice dentists and hygienists."
      icon="👨‍⚕️"
    />
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each providers as p}
        <div
          class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 relative group hover:border-slate-700 transition-colors"
        >
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
                <p class="text-xs text-sky-400 capitalize font-medium">
                  {p.role}
                  {p.specialty ? `• ${p.specialty}` : ""}
                </p>
              </div>
            </div>
            <span
              class={`px-2 py-0.5 text-[10px] font-bold rounded-full uppercase ${p.is_active ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" : "bg-slate-800 text-slate-500"}`}
            >
              {p.is_active ? "Active" : "Inactive"}
            </span>
          </div>

          {#if p.license_number || p.email || p.phone}
            <div class="text-xs text-slate-400 space-y-1 pt-2 border-t border-slate-800">
              {#if p.license_number}
                <div>
                  🪪 License: <span class="text-slate-300 font-mono">{p.license_number}</span>
                </div>
              {/if}
              {#if p.email}
                <div>✉️ {p.email}</div>
              {/if}
              {#if p.phone}
                <div>📞 {p.phone}</div>
              {/if}
            </div>
          {/if}

          <div
            class="flex items-center justify-end gap-2 pt-2 border-t border-slate-800/60 text-xs"
          >
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

<!-- PROVIDER MODAL -->
<Modal
  bind:showModal={showProviderModal}
  title={isEditingProvider ? "Edit Staff / Provider" : "Add New Staff / Provider"}
  subtitle="Configure provider details, specialties, and schedule badges"
  icon="👨‍⚕️"
  maxWidth="max-w-md"
>
  <form onsubmit={handleSaveProvider} class="space-y-4">
    <FormField label="Full Name" forId="prov-name" required>
      <Input
        id="prov-name"
        type="text"
        bind:value={provName}
        required
        placeholder="e.g. Dr. Sarah Smith"
      />
    </FormField>

    <div class="grid grid-cols-2 gap-3">
      <FormField label="Role" forId="prov-role">
        <select
          id="prov-role"
          bind:value={provRole}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          <option value="dentist">Dentist</option>
          <option value="hygienist">Dental Hygienist</option>
          <option value="assistant">Dental Assistant</option>
          <option value="staff">Administrative Staff</option>
        </select>
      </FormField>

      <FormField label="Specialty" forId="prov-specialty">
        <Input
          id="prov-specialty"
          type="text"
          bind:value={provSpecialty}
          placeholder="e.g. Orthodontics"
        />
      </FormField>
    </div>

    <FormField label="License Number" forId="prov-license">
      <Input id="prov-license" type="text" bind:value={provLicense} placeholder="e.g. DENT-88912" />
    </FormField>

    <div class="grid grid-cols-2 gap-3">
      <FormField label="Email" forId="prov-email">
        <Input
          id="prov-email"
          type="email"
          bind:value={provEmail}
          placeholder="doctor@example.com"
        />
      </FormField>

      <FormField label="Phone" forId="prov-phone">
        <Input id="prov-phone" type="tel" bind:value={provPhone} placeholder="(555) 019-2834" />
      </FormField>
    </div>

    <div class="flex items-center justify-between pt-2">
      <div>
        <label for="prov-color" class="block text-xs font-semibold text-slate-300 mb-1"
          >Schedule Color Badge</label
        >
        <input
          id="prov-color"
          type="color"
          bind:value={provColor}
          class="h-9 w-16 cursor-pointer rounded border border-slate-700 bg-slate-950 p-1"
        />
      </div>

      <div class="flex items-center gap-2 pt-4">
        <input type="checkbox" id="prov-active" bind:checked={provIsActive} />
        <label for="prov-active" class="text-xs font-semibold text-slate-300 cursor-pointer"
          >Active Provider</label
        >
      </div>
    </div>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        onclick={() => (showProviderModal = false)}
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
      >
        Cancel
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        Save Staff Member
      </button>
    </div>
  </form>
</Modal>
