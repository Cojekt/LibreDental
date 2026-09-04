<script lang="ts">
  import type { Operatory } from "@bindings/domain/models.js";
  import Modal from "../../components/ui/Modal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

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

  let searchQuery = $state("");
  let statusFilter = $state("all"); // 'all', 'active', 'inactive'

  let filteredOperatories = $derived(
    operatories.filter((op: Operatory) => {
      if (statusFilter === "active" && !op.is_active) return false;
      if (statusFilter === "inactive" && op.is_active) return false;

      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        return (
          op.name.toLowerCase().includes(q) ||
          (op.room_code || "").toLowerCase().includes(q) ||
          (op.type || "").toLowerCase().includes(q)
        );
      }
      return true;
    })
  );
</script>

<div class="space-y-6">
  <div class="flex items-center gap-3 pb-2 border-b border-slate-800">
    <div class="relative w-full max-w-[480px] flex-1">
      <svg
        class="absolute left-3.5 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-slate-400 pointer-events-none z-10"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
      </svg>
      <input
        type="text"
        placeholder={m.op_search_placeholder()}
        class="box-border w-full rounded-xl border border-slate-700 bg-slate-900 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none shadow-sm transition-all"
        style="padding-left: 2.75rem; padding-right: 0.75rem;"
        bind:value={searchQuery}
      />
    </div>
    <div
      class="flex items-center gap-1 rounded-xl border border-slate-800 bg-slate-900/90 p-1 shadow-sm select-none"
    >
      <button
        type="button"
        onclick={() => (statusFilter = "all")}
        class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${statusFilter === "all" ? "bg-slate-700 text-slate-200" : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"}`}
        >{m.common_all()}</button
      >
      <button
        type="button"
        onclick={() => (statusFilter = "active")}
        class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${statusFilter === "active" ? "bg-sky-500/20 text-sky-400 border border-sky-500/30 shadow-sm" : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"}`}
        >{m.common_active()}</button
      >
      <button
        type="button"
        onclick={() => (statusFilter = "inactive")}
        class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${statusFilter === "inactive" ? "bg-amber-500/20 text-amber-400 border border-amber-500/30 shadow-sm" : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"}`}
        >{m.common_disabled()}</button
      >
    </div>
  </div>

  {#if filteredOperatories.length === 0}
    <EmptyState title={m.clinic_op_empty_title()} subtitle={m.clinic_op_empty_sub()} />
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each filteredOperatories as op}
        <div
          class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 relative group hover:border-slate-700 transition-colors"
        >
          <div class="flex items-start justify-between">
            <div>
              <h4 class="text-sm font-bold text-slate-100">{op.name}</h4>
              <p class="text-xs text-sky-400 capitalize font-medium mt-0.5">
                {op.type}
                {m.op_room_label()}
                {op.room_code ? `• ${m.op_code_prefix()} ${op.room_code}` : ""}
              </p>
            </div>
            <span
              class={`px-2 py-0.5 text-[10px] font-bold rounded-full uppercase ${op.is_active ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" : "bg-slate-800 text-slate-500"}`}
            >
              {op.is_active ? m.common_active() : m.common_inactive()}
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
              {m.patients_btn_edit()}
            </button>
            <button
              type="button"
              onclick={() => handleDeleteOperatory(op.id)}
              class="text-rose-400 hover:text-rose-300 font-semibold"
            >
              {m.common_disable()}
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
  title={isEditingOperatory ? m.clinic_op_modal_edit_title() : m.clinic_op_modal_add_title()}
  subtitle={m.clinic_op_modal_edit_title()}
  maxWidth="max-w-md"
>
  <form onsubmit={handleSaveOperatory} class="space-y-4">
    <FormField label={m.clinic_op_name_label()} forId="op-name" required>
      <Input
        id="op-name"
        type="text"
        bind:value={opName}
        required
        placeholder={m.clinic_op_name_placeholder()}
      />
    </FormField>

    <div class="grid grid-cols-2 gap-3">
      <FormField label={m.clinic_op_code_label()} forId="op-code">
        <Input
          id="op-code"
          type="text"
          bind:value={opRoomCode}
          placeholder={m.clinic_op_code_placeholder()}
        />
      </FormField>

      <FormField label={m.clinic_op_type_label()} forId="op-type">
        <select
          id="op-type"
          bind:value={opType}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          <option value="general">{m.clinic_op_type_general()}</option>
          <option value="hygiene">{m.clinic_op_type_hygiene()}</option>
          <option value="surgery">{m.clinic_op_type_surgery()}</option>
          <option value="ortho">{m.clinic_op_type_ortho()}</option>
          <option value="pediatric">{m.clinic_op_type_pediatric()}</option>
          <option value="consultation">{m.clinic_op_type_consultation()}</option>
        </select>
      </FormField>
    </div>

    <div class="flex items-center gap-2 pt-2">
      <input type="checkbox" id="op-active" bind:checked={opIsActive} />
      <label for="op-active" class="text-xs font-semibold text-slate-300 cursor-pointer"
        >{m.clinic_op_active_label()}</label
      >
    </div>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        onclick={() => (showOperatoryModal = false)}
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
      >
        {m.common_cancel()}
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        {m.clinic_op_save_btn()}
      </button>
    </div>
  </form>
</Modal>
