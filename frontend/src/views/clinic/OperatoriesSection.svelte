<script lang="ts">
  import type { Operatory } from "@bindings/domain/models.js";
  import Modal from "../../components/ui/Modal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";

  let {
    operatories = [],
    openAddOperatoryModal,
    openEditOperatoryModal,
    handleDeleteOperatory,
    handleSaveOperatory,
    showOperatoryModal = $bindable(false),
    isEditingOperatory = false,
    opName = $bindable(""),
    opRoomCode = $bindable(""),
    opType = $bindable("general"),
    opIsActive = $bindable(true),
  } = $props<{
    operatories: Operatory[];
    openAddOperatoryModal: () => void;
    openEditOperatoryModal: (op: Operatory) => void;
    handleDeleteOperatory: (id: string) => void;
    handleSaveOperatory: (e: Event) => void;
    showOperatoryModal: boolean;
    isEditingOperatory: boolean;
    opName: string;
    opRoomCode: string;
    opType: string;
    opIsActive: boolean;
  }>();
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <div>
      <h3 class="text-lg font-bold text-slate-100">🚪 Treatment Operatories & Chairs</h3>
      <p class="text-xs text-slate-400 mt-0.5">
        Manage treatment rooms, hygiene bays, and surgical suites.
      </p>
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
    <EmptyState
      title="No operatories or treatment rooms configured"
      subtitle="Click 'Add Operatory / Room' above to set up treatment chairs for scheduling."
      icon="🚪"
    />
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each operatories as op}
        <div
          class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 relative group hover:border-slate-700 transition-colors"
        >
          <div class="flex items-start justify-between">
            <div>
              <h4 class="text-sm font-bold text-slate-100">{op.name}</h4>
              <p class="text-xs text-sky-400 capitalize font-medium mt-0.5">
                {op.type} room {op.room_code ? `• Code: ${op.room_code}` : ""}
              </p>
            </div>
            <span
              class={`px-2 py-0.5 text-[10px] font-bold rounded-full uppercase ${op.is_active ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" : "bg-slate-800 text-slate-500"}`}
            >
              {op.is_active ? "Active" : "Inactive"}
            </span>
          </div>

          <div
            class="flex items-center justify-end gap-2 pt-2 border-t border-slate-800/60 text-xs"
          >
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

<!-- OPERATORY MODAL -->
<Modal
  bind:showModal={showOperatoryModal}
  title={isEditingOperatory ? "Edit Treatment Operatory" : "Add New Operatory"}
  subtitle="Set up treatment rooms and clinical chair details"
  icon="🦷"
  maxWidth="max-w-md"
>
  <form onsubmit={handleSaveOperatory} class="space-y-4">
    <FormField label="Operatory Name" forId="op-name" required>
      <Input
        id="op-name"
        type="text"
        bind:value={opName}
        required
        placeholder="e.g. Operatory 1 (General Dentistry)"
      />
    </FormField>

    <div class="grid grid-cols-2 gap-3">
      <FormField label="Room Code / ID" forId="op-code">
        <Input id="op-code" type="text" bind:value={opRoomCode} placeholder="e.g. ROOM-A" />
      </FormField>

      <FormField label="Room Type" forId="op-type">
        <select
          id="op-type"
          bind:value={opType}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          <option value="general">General Dentistry</option>
          <option value="hygiene">Hygiene Bay</option>
          <option value="surgery">Surgical Suite</option>
          <option value="ortho">Orthodontics Bay</option>
          <option value="pediatric">Pediatric Operatory</option>
          <option value="consultation">Consultation Room</option>
        </select>
      </FormField>
    </div>

    <div class="flex items-center gap-2 pt-2">
      <input type="checkbox" id="op-active" bind:checked={opIsActive} />
      <label for="op-active" class="text-xs font-semibold text-slate-300 cursor-pointer"
        >Active Operatory</label
      >
    </div>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        onclick={() => (showOperatoryModal = false)}
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
      >
        Cancel
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        Save Operatory
      </button>
    </div>
  </form>
</Modal>
