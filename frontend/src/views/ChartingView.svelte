<script lang="ts">
  import type {
    Patient,
    DentalChart,
    ToothCondition,
    CountryConfig,
    ProcedureCode,
  } from "@bindings/domain/models.js";
  import { ToothSystem, ToothSurface, ToothStatus } from "@bindings/domain/models.js";
  import { ChartService, BillingService } from "@bindings/services/index.js";
  import ViewHeader from "../components/ui/ViewHeader.svelte";
  import OdontogramChart from "./charting/OdontogramChart.svelte";
  import ToothConditionModal from "./charting/ToothConditionModal.svelte";
  import ChartSummaryTable from "./charting/ChartSummaryTable.svelte";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "$lib/locale.svelte.js";

  let { patients = [], countryMeta = null } = $props<{
    patients: Patient[];
    countryMeta?: CountryConfig | null;
  }>();

  // Patient selection state
  let selectedPatientId = $state<string>("");
  let selectedPatient = $derived(patients.find((p: Patient) => p.id === selectedPatientId) || null);

  function getPatientLabel(p: Patient): string {
    const dobStr = p.date_of_birth || (p as any).dob;
    if (!dobStr) return `${p.first_name} ${p.last_name}`;
    try {
      const d = new Date(dobStr);
      if (!isNaN(d.getTime())) {
        const formattedDob = d.toLocaleDateString();
        return `${p.first_name} ${p.last_name} (${formattedDob})`;
      }
    } catch {
      // fallback
    }
    return `${p.first_name} ${p.last_name}`;
  }

  function calculateAge(dobStr?: string): number | null {
    if (!dobStr) return null;
    try {
      const dob = new Date(dobStr);
      if (isNaN(dob.getTime())) return null;
      const today = new Date();
      let age = today.getFullYear() - dob.getFullYear();
      const monthDiff = today.getMonth() - dob.getMonth();
      if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < dob.getDate())) {
        age--;
      }
      return age >= 0 ? age : null;
    } catch {
      return null;
    }
  }

  // Dental Chart state
  let currentChart = $state<DentalChart | null>(null);
  let loadingChart = $state<boolean>(false);

  // View state: "adult" or "primary"
  let dentitionType = $state<"adult" | "primary">("adult");

  // Selected tooth & inspector state
  let selectedToothNumber = $state<number | null>(null);
  let showConditionModal = $state<boolean>(false);
  let isEditingCondition = $state<boolean>(false);
  let editingConditionId = $state<string>("");

  // Condition Form Fields
  let formSurfaces = $state<ToothSurface[]>([]);
  let formADACode = $state<string>("");
  let formDescription = $state<string>("");
  let formStatus = $state<ToothStatus>(ToothStatus.ToothStatusTreatmentPlanned);
  let formFee = $state<number>(0);

  // Active tooth system strictly derived from country configuration
  let currentToothSystem = $derived<ToothSystem>(
    countryMeta?.default_tooth_system || ToothSystem.ToothSystemUniversal
  );

  // Tooth numbering helper functions
  function getToothLabel(num: number, system: ToothSystem): string {
    if (num >= 1 && num <= 32) {
      if (system === ToothSystem.ToothSystemFDI) {
        if (num >= 1 && num <= 8) return `1${9 - num}`;
        if (num >= 9 && num <= 16) return `2${num - 8}`;
        if (num >= 17 && num <= 24) return `3${num - 16}`;
        if (num >= 25 && num <= 32) return `4${33 - num}`;
      } else if (system === ToothSystem.ToothSystemPalmer) {
        if (num >= 1 && num <= 8) return `UR${9 - num}`;
        if (num >= 9 && num <= 16) return `UL${num - 8}`;
        if (num >= 17 && num <= 24) return `LL${num - 16}`;
        if (num >= 25 && num <= 32) return `LR${33 - num}`;
      }
      return String(num);
    }

    if (num >= 101 && num <= 120) {
      const idx = num - 101;
      if (system === ToothSystem.ToothSystemFDI) {
        if (idx >= 0 && idx <= 4) return `5${5 - idx}`;
        if (idx >= 5 && idx <= 9) return `6${idx - 4}`;
        if (idx >= 10 && idx <= 14) return `7${idx - 9}`;
        if (idx >= 15 && idx <= 19) return `8${20 - idx}`;
      } else if (system === ToothSystem.ToothSystemPalmer) {
        const letters = ["A", "B", "C", "D", "E"];
        if (idx >= 0 && idx <= 4) return `UR${letters[4 - idx]}`;
        if (idx >= 5 && idx <= 9) return `UL${letters[idx - 5]}`;
        if (idx >= 10 && idx <= 14) return `LL${letters[idx - 10]}`;
        if (idx >= 15 && idx <= 19) return `LR${letters[19 - idx]}`;
      }
      const universalPrimary = [
        "A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
        "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T",
      ];
      return universalPrimary[idx] || String(num);
    }

    return String(num);
  }

  // Adult teeth lists (1..32)
  const upperAdultRight = [1, 2, 3, 4, 5, 6, 7, 8];
  const upperAdultLeft = [9, 10, 11, 12, 13, 14, 15, 16];
  const lowerAdultLeft = [17, 18, 19, 20, 21, 22, 23, 24];
  const lowerAdultRight = [25, 26, 27, 28, 29, 30, 31, 32];

  // Primary teeth lists (101..120)
  const upperPrimaryRight = [101, 102, 103, 104, 105];
  const upperPrimaryLeft = [106, 107, 108, 109, 110];
  const lowerPrimaryLeft = [111, 112, 113, 114, 115];
  const lowerPrimaryRight = [116, 117, 118, 119, 120];

  const procedurePresets = [
    { code: "D2391", desc: "1-Surface Composite Resin (Posterior)", fee: 140, status: "treatment_planned" },
    { code: "D2392", desc: "2-Surface Composite Resin (Posterior)", fee: 185, status: "treatment_planned" },
    { code: "D2393", desc: "3-Surface Composite Resin (Posterior)", fee: 230, status: "treatment_planned" },
    { code: "D2750", desc: "Crown - Porcelain Fused to High Noble Metal", fee: 950, status: "treatment_planned" },
    { code: "D3330", desc: "Endodontic Therapy - Molar Root Canal", fee: 850, status: "treatment_planned" },
    { code: "D7140", desc: "Extraction, Erupted Tooth or Exposed Root", fee: 160, status: "missing" },
    { code: "D1351", desc: "Dental Sealant - Per Tooth", fee: 55, status: "completed" },
    { code: "EXISTS", desc: "Existing Restoration / Healthy Tooth", fee: 0, status: "existing" },
  ];

  let procedureCodes = $state<ProcedureCode[]>([]);
  let isCreatingClaim = $state(false);
  let claimNoticeMsg = $state("");

  async function loadProcedureCodes() {
    const cc = countryMeta?.code || "US";
    try {
      const res = await BillingService.ListProcedureCodes(cc, "");
      procedureCodes = (res?.filter(Boolean) as ProcedureCode[]) || [];
    } catch (e) {
      console.error("Failed to load procedure codes for country:", cc, e);
    }
  }

  $effect(() => {
    loadProcedureCodes();
  });

  async function handleGenerateClaimFromChart() {
    if (
      !selectedPatientId ||
      !currentChart ||
      !currentChart.conditions ||
      currentChart.conditions.length === 0
    )
      return;

    const billable = currentChart.conditions.filter(
      (c) => c.status === "treatment_planned" || c.status === "completed"
    );
    if (billable.length === 0) {
      alert("No treatment-planned or completed conditions found for billing.");
      return;
    }

    isCreatingClaim = true;
    claimNoticeMsg = "";
    try {
      const ids = billable.map((c) => c.id);
      const claim = await BillingService.CreateClaimFromChartConditions(selectedPatientId, "", ids);
      if (claim) {
        claimNoticeMsg = `Claim created successfully! (${claim.line_items?.length || 0} line items billed)`;
        await loadChart(selectedPatientId);
      }
    } catch (e) {
      console.error("Failed to create claim from chart:", e);
      alert("Failed to generate claim from chart.");
    } finally {
      isCreatingClaim = false;
    }
  }

  async function loadChart(patientId: string) {
    if (!patientId) {
      currentChart = null;
      return;
    }
    loadingChart = true;
    try {
      const chart = await ChartService.GetPatientChart(patientId);
      currentChart = chart || {
        patient_id: patientId,
        conditions: [],
        updated_at: "",
      };
    } catch (e) {
      console.error("Failed to load patient chart:", e);
      currentChart = { patient_id: patientId, conditions: [], updated_at: "" };
    } finally {
      loadingChart = false;
    }
  }

  $effect(() => {
    if (selectedPatientId) {
      loadChart(selectedPatientId);
    }
  });

  function getConditionsForTooth(toothNum: number): ToothCondition[] {
    if (!currentChart || !currentChart.conditions) return [];
    return currentChart.conditions.filter((c: ToothCondition) => c.tooth_number === toothNum);
  }

  function getToothPrimaryStatus(toothNum: number): string | null {
    const conds = getConditionsForTooth(toothNum);
    if (conds.length === 0) return null;
    if (conds.some((c: ToothCondition) => c.status === "missing")) return "missing";
    if (conds.some((c: ToothCondition) => c.status === "treatment_planned"))
      return "treatment_planned";
    if (conds.some((c: ToothCondition) => c.status === "completed")) return "completed";
    if (conds.some((c: ToothCondition) => c.status === "existing")) return "existing";
    return null;
  }

  function openAddConditionForTooth(toothNum: number) {
    selectedToothNumber = toothNum;
    isEditingCondition = false;
    editingConditionId = "";
    formSurfaces = [];
    formADACode = "D2391";
    formDescription = "1-Surface Composite Resin";
    formStatus = ToothStatus.ToothStatusTreatmentPlanned;
    formFee = 140;
    showConditionModal = true;
  }

  function openEditCondition(cond: ToothCondition) {
    selectedToothNumber = cond.tooth_number;
    isEditingCondition = true;
    editingConditionId = cond.id;
    formSurfaces = cond.surfaces ? [...cond.surfaces] : [];
    formADACode = cond.ada_code || "";
    formDescription = cond.description;
    formStatus = cond.status;
    formFee = cond.fee || 0;
    showConditionModal = true;
  }

  function toggleSurface(s: ToothSurface) {
    if (formSurfaces.includes(s)) {
      formSurfaces = formSurfaces.filter((item: ToothSurface) => item !== s);
    } else {
      formSurfaces = [...formSurfaces, s];
    }
  }

  function applyPreset(preset: (typeof procedurePresets)[0]) {
    formADACode = preset.code === "EXISTS" ? "" : preset.code;
    formDescription = preset.desc;
    formFee = preset.fee;
    formStatus = preset.status as ToothStatus;
  }

  async function handleSaveCondition(e: Event) {
    e.preventDefault();
    if (!selectedPatientId || !selectedToothNumber) return;

    const conditionData: ToothCondition = {
      id: isEditingCondition ? editingConditionId : `cond_${Date.now()}`,
      patient_id: selectedPatientId,
      tooth_number: selectedToothNumber,
      surfaces: formSurfaces,
      ada_code: formADACode,
      description: formDescription || "Tooth finding",
      status: formStatus,
      fee: Number(formFee) || 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    try {
      await ChartService.SaveToothCondition(conditionData);
      showConditionModal = false;
      await loadChart(selectedPatientId);
    } catch (err) {
      console.error("Failed to save tooth condition:", err);
    }
  }

  async function handleDeleteCondition(id: string) {
    if (confirm(m.confirm_delete_condition())) {
      try {
        await ChartService.DeleteToothCondition(id);
        if (editingConditionId === id) {
          showConditionModal = false;
        }
        await loadChart(selectedPatientId);
      } catch (err) {
        console.error("Failed to delete condition:", err);
      }
    }
  }

  function formatCurrency(amount: number): string {
    const currency = countryMeta?.default_currency || "";
    if (!currency) return amount.toFixed(2);
    try {
      return new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: currency,
      }).format(amount);
    } catch {
      return `${amount.toFixed(2)}`;
    }
  }
</script>

<div class="flex flex-col gap-6 w-full" data-locale={getLocaleVersion()}>
  <!-- Header Bar -->
  <div
    class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-lg"
  >
    <div class="flex flex-col sm:flex-row sm:items-center gap-4">
      <div class="flex items-center gap-3">
        <div
          class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-500 text-white shadow-md shadow-sky-500/20 text-lg"
        >
          🦷
        </div>
        <div>
          <h2 class="text-lg font-bold text-slate-100 m-0">
            {m.charting_title()}
          </h2>
          <p class="text-xs text-slate-400 m-0">{m.charting_subtitle()}</p>
        </div>
      </div>

      <!-- Patient Select Dropdown -->
      <div class="flex items-center gap-2">
        <label for="chart-patient-select" class="text-xs font-medium text-slate-400">Patient:</label>
        <select
          id="chart-patient-select"
          bind:value={selectedPatientId}
          class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500 transition-colors"
        >
          <option value="">{m.charting_select_patient_prompt()}</option>
          {#if patients.length === 0}
            <option value="" disabled>No active patients found</option>
          {/if}
          {#each patients as p}
            <option value={p.id}>
              {getPatientLabel(p)}
            </option>
          {/each}
        </select>

        {#if selectedPatient}
          {@const age = calculateAge(selectedPatient.date_of_birth || (selectedPatient as any).dob)}
          {#if age !== null}
            <span
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-slate-950 border border-slate-800 text-xs font-medium text-slate-300"
            >
              <span class="text-slate-400">Age:</span>
              <span class="font-bold text-sky-400">{age}</span>
            </span>
          {/if}
        {/if}
      </div>
    </div>

    <!-- Country Determined Tooth System Badge -->
    <div class="flex items-center gap-3 self-start sm:self-auto">
      <div
        class="flex items-center gap-2 px-3 py-1.5 rounded-xl border border-sky-500/30 bg-sky-950/40 text-sky-300 text-xs font-semibold"
      >
        <span>📍</span>
        <span>
          {countryMeta?.name || "Global"}:
          {#if currentToothSystem === ToothSystem.ToothSystemFDI}
            <strong class="text-sky-200">FDI (ISO 3950) Notation</strong>
          {:else if currentToothSystem === ToothSystem.ToothSystemPalmer}
            <strong class="text-sky-200">Palmer Notation</strong>
          {:else}
            <strong class="text-sky-200">Universal Numbering System (1–32)</strong>
          {/if}
        </span>
      </div>

      <!-- Dentition Switcher -->
      <div class="flex items-center bg-slate-950 p-1 rounded-xl border border-slate-800">
        <button
          type="button"
          onclick={() => (dentitionType = "adult")}
          class={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
            dentitionType === "adult"
              ? "bg-sky-500 text-white shadow-sm"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          {m.charting_adult_teeth()}
        </button>
        <button
          type="button"
          onclick={() => (dentitionType = "primary")}
          class={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
            dentitionType === "primary"
              ? "bg-sky-500 text-white shadow-sm"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          {m.charting_primary_teeth()}
        </button>
      </div>
    </div>
  </div>

  {#if !selectedPatient}
    <div
      class="flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-slate-800 rounded-2xl bg-slate-900/40"
    >
      <p class="text-slate-400 text-sm m-0 font-medium">
        {m.charting_select_patient_desc()}
      </p>
    </div>
  {:else}
    <OdontogramChart
      {dentitionType}
      {currentToothSystem}
      {loadingChart}
      {upperAdultRight}
      {upperAdultLeft}
      {upperPrimaryRight}
      {upperPrimaryLeft}
      {lowerAdultRight}
      {lowerAdultLeft}
      {lowerPrimaryRight}
      {lowerPrimaryLeft}
      {getConditionsForTooth}
      {getToothPrimaryStatus}
      {getToothLabel}
      {openAddConditionForTooth}
    />

    <ChartSummaryTable
      {selectedPatient}
      {currentChart}
      {countryMeta}
      {currentToothSystem}
      {isCreatingClaim}
      bind:claimNoticeMsg
      {getToothLabel}
      {openEditCondition}
      {handleDeleteCondition}
      {handleGenerateClaimFromChart}
      {formatCurrency}
    />
  {/if}
</div>

<ToothConditionModal
  bind:showConditionModal
  {selectedToothNumber}
  {currentToothSystem}
  {countryMeta}
  bind:formSurfaces
  bind:formADACode
  bind:formDescription
  bind:formStatus
  bind:formFee
  {isEditingCondition}
  {editingConditionId}
  {procedurePresets}
  {procedureCodes}
  {getToothLabel}
  {toggleSurface}
  {applyPreset}
  {handleSaveCondition}
  {handleDeleteCondition}
  {formatCurrency}
/>
