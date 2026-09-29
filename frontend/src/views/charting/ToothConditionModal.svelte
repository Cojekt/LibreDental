<script lang="ts">
  import type { CountryConfig, ProcedureCode } from "@bindings/domain/models.js";
  import { ToothSystem, ToothSurface, ToothStatus } from "@bindings/domain/models.js";
  import { m } from "../../paraglide/messages.js";
  import Modal from "../../components/ui/Modal.svelte";
  import { getLocaleVersion } from "$lib/locale.svelte.js";

  let {
    showConditionModal = $bindable(false),
    selectedToothNumber,
    currentToothSystem,
    countryMeta = null,
    formSurfaces = $bindable([]),
    formADACode = $bindable(""),
    formDescription = $bindable(""),
    formStatus = $bindable(ToothStatus.ToothStatusTreatmentPlanned),
    formFee = $bindable(0),
    isEditingCondition,
    editingConditionId,
    procedurePresets,
    procedureCodes,
    getToothLabel,
    toggleSurface,
    applyPreset,
    handleSaveCondition,
    handleDeleteCondition,
    formatCurrency,
  } = $props<{
    showConditionModal: boolean;
    selectedToothNumber: number | null;
    currentToothSystem: ToothSystem;
    countryMeta?: CountryConfig | null;
    formSurfaces: ToothSurface[];
    formADACode: string;
    formDescription: string;
    formStatus: ToothStatus;
    formFee: number;
    isEditingCondition: boolean;
    editingConditionId: string;
    procedurePresets: { code: string; desc: string; fee: number; status: string }[];
    procedureCodes: ProcedureCode[];
    getToothLabel: (num: number, system: ToothSystem) => string;
    toggleSurface: (s: ToothSurface) => void;
    applyPreset: (preset: { code: string; desc: string; fee: number; status: string }) => void;
    handleSaveCondition: (e: Event) => void;
    handleDeleteCondition: (id: string) => void;
    formatCurrency: (amount: number) => string;
  }>();

  const surfaceOptions = $derived.by(() => {
    getLocaleVersion();
    return [
      { id: ToothSurface.SurfaceMesial, label: m.charting_surface_mesial() },
      { id: ToothSurface.SurfaceDistal, label: m.charting_surface_distal() },
      { id: ToothSurface.SurfaceOcclusal, label: m.charting_surface_occlusal() },
      { id: ToothSurface.SurfaceIncisal, label: m.charting_surface_incisal() },
      { id: ToothSurface.SurfaceFacial, label: m.charting_surface_facial() },
      { id: ToothSurface.SurfaceLingual, label: m.charting_surface_lingual() },
    ];
  });
</script>

<Modal
  bind:showModal={showConditionModal}
  title={selectedToothNumber
    ? m.charting_modal_title({
        label: getToothLabel(selectedToothNumber, currentToothSystem),
      })
    : ""}
  subtitle={`${countryMeta?.name || m.charting_practice_country()} ${m.charting_modal_inspector_title()}`}
  maxWidth="max-w-lg"
>
  {#if selectedToothNumber}
    <div class="flex flex-col gap-5">
      <!-- Surface Selection Grid -->
      <div class="flex flex-col gap-2">
        <span class="text-xs font-semibold text-slate-300">{m.charting_modal_surfaces_label()}</span
        >
        <div class="grid grid-cols-3 sm:grid-cols-6 gap-2">
          {#each surfaceOptions as s}
            <button
              type="button"
              onclick={() => toggleSurface(s.id)}
              class={`py-2 px-2 text-xs font-bold rounded-xl border transition-all ${
                formSurfaces.includes(s.id)
                  ? "bg-sky-500 text-white border-sky-400 shadow-md shadow-sky-500/20"
                  : "bg-slate-950 text-slate-300 border-slate-800 hover:border-slate-700"
              }`}
            >
              {s.label}
            </button>
          {/each}
        </div>
      </div>

      <!-- Quick Procedure Presets -->
      <div class="flex flex-col gap-2">
        <span class="text-xs font-semibold text-slate-300">{m.charting_modal_presets_label()}</span>
        <div class="flex flex-wrap gap-1.5">
          {#each procedurePresets as preset}
            <button
              type="button"
              onclick={() => applyPreset(preset)}
              class="px-2.5 py-1 text-[11px] font-medium rounded-lg bg-slate-800 text-slate-300 border border-slate-700 hover:bg-slate-700 hover:text-white transition-colors"
            >
              {preset.code ? `[${preset.code}] ` : ""}{preset.desc}
            </button>
          {/each}
        </div>
      </div>

      <!-- Form Controls -->
      <form onsubmit={handleSaveCondition} class="flex flex-col gap-4">
        {#if procedureCodes.length > 0}
          <div class="flex flex-col gap-1">
            <label for="condition-catalog-code" class="text-xs font-semibold text-sky-400">
              {countryMeta?.name || m.charting_catalog_regional()}
              {m.charting_modal_catalog_label()}
            </label>
            <select
              id="condition-catalog-code"
              onchange={(e) => {
                const code = (e.target as HTMLSelectElement).value;
                const item = procedureCodes.find((p: ProcedureCode) => p.code === code);
                if (item) {
                  formADACode = item.code;
                  formDescription = item.description;
                  formFee = (item.effective_fee || item.default_fee) / 100;
                }
              }}
              class="bg-slate-950 border border-sky-500/40 text-slate-200 text-xs rounded-xl px-3 py-2 outline-none focus:border-sky-500 cursor-pointer"
            >
              <option value=""
                >{m.charting_modal_catalog_prompt({
                  name: countryMeta?.name || m.charting_catalog_country(),
                })}</option
              >
              {#each procedureCodes as p}
                <option value={p.code}>
                  [{p.category}] {p.code} - {p.description} ({formatCurrency(
                    p.effective_fee || p.default_fee
                  )})
                </option>
              {/each}
            </select>
          </div>
        {/if}

        <div class="grid grid-cols-2 gap-3">
          <div class="flex flex-col gap-1">
            <label for="condition-ada-code" class="text-xs font-medium text-slate-400"
              >{m.charting_modal_code_label()}</label
            >
            <input
              id="condition-ada-code"
              type="text"
              bind:value={formADACode}
              placeholder={m.charting_modal_code_placeholder()}
              class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label for="condition-fee" class="text-xs font-medium text-slate-400"
              >{m.charting_modal_fee_label()}
              {countryMeta?.default_currency ? `(${countryMeta.default_currency})` : ""}</label
            >
            <input
              id="condition-fee"
              type="number"
              step="0.01"
              bind:value={formFee}
              class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500"
            />
          </div>
        </div>

        <div class="flex flex-col gap-1">
          <label for="condition-description" class="text-xs font-medium text-slate-400"
            >{m.charting_modal_desc_label()}</label
          >
          <input
            id="condition-description"
            type="text"
            bind:value={formDescription}
            required
            placeholder={m.charting_modal_desc_placeholder()}
            class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label for="condition-status" class="text-xs font-medium text-slate-400"
            >{m.charting_modal_status_label()}</label
          >
          <select
            id="condition-status"
            bind:value={formStatus}
            class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500"
          >
            <option value="treatment_planned">{m.charting_modal_status_planned()}</option>
            <option value="completed">{m.charting_modal_status_completed()}</option>
            <option value="existing">{m.charting_modal_status_existing()}</option>
            <option value="missing">{m.charting_modal_status_missing()}</option>
          </select>
        </div>

        <div class="flex items-center justify-between pt-3 border-t border-slate-800">
          {#if isEditingCondition}
            <button
              type="button"
              onclick={() => handleDeleteCondition(editingConditionId)}
              class="px-3 py-2 text-xs font-semibold text-rose-400 hover:text-rose-300 hover:bg-rose-950/40 rounded-xl transition-colors"
            >
              {m.charting_modal_btn_delete()}
            </button>
          {:else}
            <div></div>
          {/if}

          <div class="flex items-center gap-3">
            <button
              type="button"
              onclick={() => (showConditionModal = false)}
              class="btn btn-secondary text-xs"
            >
              {m.common_cancel()}
            </button>
            <button type="submit" class="btn btn-primary text-xs shadow-md shadow-sky-500/20">
              {isEditingCondition ? m.patient_save_changes() : m.common_save()}
            </button>
          </div>
        </div>
      </form>
    </div>
  {/if}
</Modal>
