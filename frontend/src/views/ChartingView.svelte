<script lang="ts">
  import type {
    Patient,
    DentalChart,
    ToothCondition,
    ToothStatus,
    CountryConfig,
  } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import {
    ToothSystem,
    ToothSurface,
  } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";
  import { ChartService } from "../../bindings/github.com/LibreDental/libredental/pkg/services/index.js";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion } from "../lib/locale.svelte.js";

  let { patients = [], countryMeta = null } = $props<{
    patients: Patient[];
    countryMeta?: CountryConfig | null;
  }>();

  // Patient selection state
  let selectedPatientId = $state<string>("");
  let selectedPatient = $derived(
    patients.find((p: Patient) => p.id === selectedPatientId) || null,
  );

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
  let formStatus = $state<ToothStatus>("treatment_planned" as ToothStatus);
  let formFee = $state<number>(0);

  // Active tooth system strictly derived from country configuration
  let currentToothSystem = $derived<ToothSystem>(
    countryMeta?.default_tooth_system || ToothSystem.ToothSystemUniversal,
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
        "A",
        "B",
        "C",
        "D",
        "E",
        "F",
        "G",
        "H",
        "I",
        "J",
        "K",
        "L",
        "M",
        "N",
        "O",
        "P",
        "Q",
        "R",
        "S",
        "T",
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
  const upperPrimaryRight = [101, 102, 103, 104, 105]; // A..E
  const upperPrimaryLeft = [106, 107, 108, 109, 110]; // F..J
  const lowerPrimaryLeft = [111, 112, 113, 114, 115]; // K..O
  const lowerPrimaryRight = [116, 117, 118, 119, 120]; // P..T

  // Preset dental procedures
  const procedurePresets = [
    {
      code: "D2391",
      desc: "1-Surface Composite Resin (Posterior)",
      fee: 140,
      status: "treatment_planned",
    },
    {
      code: "D2392",
      desc: "2-Surface Composite Resin (Posterior)",
      fee: 185,
      status: "treatment_planned",
    },
    {
      code: "D2393",
      desc: "3-Surface Composite Resin (Posterior)",
      fee: 230,
      status: "treatment_planned",
    },
    {
      code: "D2750",
      desc: "Crown - Porcelain Fused to High Noble Metal",
      fee: 950,
      status: "treatment_planned",
    },
    {
      code: "D3330",
      desc: "Endodontic Therapy - Molar Root Canal",
      fee: 850,
      status: "treatment_planned",
    },
    {
      code: "D7140",
      desc: "Extraction, Erupted Tooth or Exposed Root",
      fee: 160,
      status: "missing",
    },
    {
      code: "D1351",
      desc: "Dental Sealant - Per Tooth",
      fee: 55,
      status: "completed",
    },
    {
      code: "EXISTS",
      desc: "Existing Restoration / Healthy Tooth",
      fee: 0,
      status: "existing",
    },
  ];

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

  // Auto-select first patient if available and none selected
  $effect(() => {
    if (!selectedPatientId && patients && patients.length > 0) {
      selectedPatientId = patients[0].id;
    }
  });

  function getConditionsForTooth(toothNum: number): ToothCondition[] {
    if (!currentChart || !currentChart.conditions) return [];
    return currentChart.conditions.filter(
      (c: ToothCondition) => c.tooth_number === toothNum,
    );
  }

  function getToothPrimaryStatus(toothNum: number): string | null {
    const conds = getConditionsForTooth(toothNum);
    if (conds.length === 0) return null;
    if (conds.some((c: ToothCondition) => c.status === "missing"))
      return "missing";
    if (conds.some((c: ToothCondition) => c.status === "treatment_planned"))
      return "treatment_planned";
    if (conds.some((c: ToothCondition) => c.status === "completed"))
      return "completed";
    if (conds.some((c: ToothCondition) => c.status === "existing"))
      return "existing";
    return null;
  }

  function openAddConditionForTooth(toothNum: number) {
    selectedToothNumber = toothNum;
    isEditingCondition = false;
    editingConditionId = "";
    formSurfaces = [];
    formADACode = "D2391";
    formDescription = "1-Surface Composite Resin";
    formStatus = "treatment_planned" as ToothStatus;
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
    } catch (e) {
      return `${amount.toFixed(2)}`;
    }
  }
</script>

<div class="flex flex-col gap-6 w-full">
  <!-- Header Bar: Patient Selector & Country System Indicator -->
  <div
    class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-lg"
  >
    <div class="flex flex-col sm:flex-row sm:items-center gap-4">
      <div class="flex items-center gap-3">
        <div
          class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-500 text-white shadow-md shadow-sky-500/20"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-5 w-5"
          >
            <path
              d="M12 2C8 2 5 5 5 9c0 5.25 3 9.25 7 13 4-3.75 7-7.75 7-13 0-4-3-7-7-7z"
            />
            <circle cx="12" cy="9" r="2.5" />
          </svg>
        </div>
        <div>
          <h2 class="text-lg font-bold text-slate-100 m-0">
            {(getLocaleVersion(), m.charting_title())}
          </h2>
          <p class="text-xs text-slate-400 m-0">{m.charting_subtitle()}</p>
        </div>
      </div>

      <!-- Patient Select Dropdown -->
      <div class="flex items-center gap-2">
        <label
          for="chart-patient-select"
          class="text-xs font-medium text-slate-400">Patient:</label
        >
        <select
          id="chart-patient-select"
          bind:value={selectedPatientId}
          class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500 transition-colors"
        >
          {#if patients.length === 0}
            <option value="">No active patients found</option>
          {/if}
          {#each patients as p}
            <option value={p.id}>
              {p.first_name}
              {p.last_name} ({p.dob ? p.dob.substring(0, 4) : ""})
            </option>
          {/each}
        </select>
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
            <strong class="text-sky-200"
              >Universal Numbering System (1–32)</strong
            >
          {/if}
        </span>
      </div>

      <!-- Dentition Switcher -->
      <div
        class="flex items-center bg-slate-950 p-1 rounded-xl border border-slate-800"
      >
        <button
          type="button"
          onclick={() => (dentitionType = "adult")}
          class={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
            dentitionType === "adult"
              ? "bg-sky-500 text-white shadow-sm"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          Adult Teeth (32)
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
          Primary Teeth (20)
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
    <!-- Status Legend -->
    <div
      class="flex flex-wrap items-center justify-between gap-4 bg-slate-900/80 border border-slate-800/80 rounded-xl px-5 py-3 text-xs"
    >
      <div class="flex items-center gap-5">
        <span class="font-semibold text-slate-300">Legend:</span>
        <div class="flex items-center gap-1.5">
          <span
            class="w-3 h-3 rounded-full bg-blue-500 shadow-sm shadow-blue-500/50"
          ></span>
          <span class="text-slate-300">Existing Finding</span>
        </div>
        <div class="flex items-center gap-1.5">
          <span
            class="w-3 h-3 rounded-full bg-amber-500 shadow-sm shadow-amber-500/50"
          ></span>
          <span class="text-slate-300">Treatment Planned</span>
        </div>
        <div class="flex items-center gap-1.5">
          <span
            class="w-3 h-3 rounded-full bg-emerald-500 shadow-sm shadow-emerald-500/50"
          ></span>
          <span class="text-slate-300">Completed</span>
        </div>
        <div class="flex items-center gap-1.5">
          <span
            class="w-3 h-3 rounded-full bg-rose-500/80 border border-rose-400"
          ></span>
          <span class="text-slate-300">Missing Tooth</span>
        </div>
      </div>
      <div class="text-slate-400 text-[11px]">
        Click any tooth card to view or record conditions & procedures.
      </div>
    </div>

    <!-- Odontogram Chart Layout -->
    <div
      class="flex flex-col gap-6 bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-xl relative"
    >
      {#if loadingChart}
        <div
          class="absolute inset-0 bg-slate-950/60 backdrop-blur-sm flex items-center justify-center rounded-2xl z-20"
        >
          <div
            class="flex items-center gap-2 text-sky-400 text-sm font-semibold"
          >
            <svg
              class="animate-spin h-5 w-5 text-sky-400"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            Loading Dental Chart...
          </div>
        </div>
      {/if}

      <!-- Maxillary Arch (Upper Teeth) -->
      <div>
        <div
          class="flex items-center justify-between mb-3 border-b border-slate-800 pb-2"
        >
          <span class="text-xs font-bold uppercase tracking-wider text-sky-400"
            >Upper Arch (Maxillary)</span
          >
          <span class="text-[11px] text-slate-500 font-mono"
            >Right &larr; &rarr; Left</span
          >
        </div>

        <div class="grid grid-cols-8 sm:grid-cols-16 gap-2 sm:gap-2.5">
          {#if dentitionType === "adult"}
            <!-- Upper Right 1..8 -->
            {#each upperAdultRight as toothNum}
              {@render toothCard(toothNum)}
            {/each}
            <!-- Upper Left 9..16 -->
            {#each upperAdultLeft as toothNum}
              {@render toothCard(toothNum)}
            {/each}
          {:else}
            <!-- Primary Upper Right A..E -->
            <div class="col-span-3"></div>
            {#each upperPrimaryRight as toothNum}
              {@render toothCard(toothNum)}
            {/each}
            <!-- Primary Upper Left F..J -->
            {#each upperPrimaryLeft as toothNum}
              {@render toothCard(toothNum)}
            {/each}
            <div class="col-span-3"></div>
          {/if}
        </div>
      </div>

      <!-- Midline Divider -->
      <div class="flex items-center justify-center my-1">
        <div
          class="h-[1px] bg-gradient-to-r from-transparent via-slate-700 to-transparent w-full max-w-xl"
        ></div>
      </div>

      <!-- Mandibular Arch (Lower Teeth) -->
      <div>
        <div
          class="flex items-center justify-between mb-3 border-b border-slate-800 pb-2"
        >
          <span class="text-xs font-bold uppercase tracking-wider text-sky-400"
            >Lower Arch (Mandibular)</span
          >
          <span class="text-[11px] text-slate-500 font-mono"
            >Right &larr; &rarr; Left</span
          >
        </div>

        <div class="grid grid-cols-8 sm:grid-cols-16 gap-2 sm:gap-2.5">
          {#if dentitionType === "adult"}
            <!-- Lower Right 32..25 -->
            {#each lowerAdultRight.slice().reverse() as toothNum}
              {@render toothCard(toothNum)}
            {/each}
            <!-- Lower Left 24..17 -->
            {#each lowerAdultLeft.slice().reverse() as toothNum}
              {@render toothCard(toothNum)}
            {/each}
          {:else}
            <!-- Primary Lower Right P..T -->
            <div class="col-span-3"></div>
            {#each lowerPrimaryRight.slice().reverse() as toothNum}
              {@render toothCard(toothNum)}
            {/each}
            <!-- Primary Lower Left K..O -->
            {#each lowerPrimaryLeft.slice().reverse() as toothNum}
              {@render toothCard(toothNum)}
            {/each}
            <div class="col-span-3"></div>
          {/if}
        </div>
      </div>
    </div>

    <!-- Chart Conditions Summary Table -->
    <div
      class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-xl flex flex-col gap-4"
    >
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-base font-bold text-slate-100 m-0">
            Charted Conditions & Treatment Log
          </h3>
          <p class="text-xs text-slate-400 m-0">
            Summary of findings for {selectedPatient.first_name}
            {selectedPatient.last_name}
          </p>
        </div>
        {#if currentChart && currentChart.conditions && currentChart.conditions.length > 0}
          <div class="text-xs font-semibold text-slate-300">
            Total Planned Fee: <span class="text-sky-400 font-bold"
              >{formatCurrency(
                currentChart.conditions.reduce(
                  (acc, c) => acc + (c.fee || 0),
                  0,
                ),
              )}</span
            >
          </div>
        {/if}
      </div>

      {#if !currentChart || !currentChart.conditions || currentChart.conditions.length === 0}
        <div
          class="text-center py-10 border border-dashed border-slate-800 rounded-xl bg-slate-950/40"
        >
          <p class="text-sm text-slate-400 m-0">
            No tooth conditions or treatment plans recorded yet for this
            patient.
          </p>
          <p class="text-xs text-slate-500 m-1">
            Click any tooth above to add findings.
          </p>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs border-collapse">
            <thead>
              <tr
                class="border-b border-slate-800 text-slate-400 uppercase font-semibold text-[11px] bg-slate-950/60"
              >
                <th class="py-3 px-4"
                  >Tooth # ({countryMeta?.code || "Universal"})</th
                >
                <th class="py-3 px-4">Surfaces</th>
                <th class="py-3 px-4">ADA Code</th>
                <th class="py-3 px-4">Description</th>
                <th class="py-3 px-4">Status</th>
                <th class="py-3 px-4 text-right">Fee</th>
                <th class="py-3 px-4 text-center">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/60">
              {#each currentChart.conditions || [] as cond}
                <tr class="hover:bg-slate-800/40 transition-colors">
                  <td class="py-3 px-4 font-bold text-sky-400">
                    Tooth {getToothLabel(cond.tooth_number, currentToothSystem)}
                    <span class="text-[10px] text-slate-500 font-normal">
                      (Internal #{cond.tooth_number})</span
                    >
                  </td>
                  <td class="py-3 px-4 font-mono font-medium text-slate-300">
                    {cond.surfaces && cond.surfaces.length > 0
                      ? cond.surfaces.join(", ")
                      : "Whole Tooth"}
                  </td>
                  <td class="py-3 px-4 font-mono text-slate-300">
                    {cond.ada_code || "—"}
                  </td>
                  <td class="py-3 px-4 text-slate-200 font-medium">
                    {cond.description}
                  </td>
                  <td class="py-3 px-4">
                    {#if cond.status === "missing"}
                      <span
                        class="px-2.5 py-0.5 rounded-lg text-[10px] font-bold bg-rose-950/80 text-rose-300 border border-rose-800"
                      >
                        Missing
                      </span>
                    {:else if cond.status === "treatment_planned"}
                      <span
                        class="px-2.5 py-0.5 rounded-lg text-[10px] font-bold bg-amber-950/80 text-amber-300 border border-amber-800"
                      >
                        Treatment Planned
                      </span>
                    {:else if cond.status === "completed"}
                      <span
                        class="px-2.5 py-0.5 rounded-lg text-[10px] font-bold bg-emerald-950/80 text-emerald-300 border border-emerald-800"
                      >
                        Completed
                      </span>
                    {:else}
                      <span
                        class="px-2.5 py-0.5 rounded-lg text-[10px] font-bold bg-blue-950/80 text-blue-300 border border-blue-800"
                      >
                        Existing
                      </span>
                    {/if}
                  </td>
                  <td class="py-3 px-4 text-right font-semibold text-slate-200">
                    {formatCurrency(cond.fee || 0)}
                  </td>
                  <td class="py-3 px-4 text-center">
                    <div class="flex items-center justify-center gap-2">
                      <button
                        type="button"
                        onclick={() => openEditCondition(cond)}
                        class="p-1.5 rounded-lg text-slate-400 hover:text-sky-400 hover:bg-slate-800 transition-colors"
                        title="Edit condition"
                      >
                        <svg
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                          class="w-4 h-4"
                        >
                          <path
                            d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"
                          ></path>
                          <path
                            d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"
                          ></path>
                        </svg>
                      </button>
                      <button
                        type="button"
                        onclick={() => handleDeleteCondition(cond.id)}
                        class="p-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-slate-800 transition-colors"
                        title="Delete condition"
                      >
                        <svg
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                          class="w-4 h-4"
                        >
                          <polyline points="3 6 5 6 21 6"></polyline>
                          <path
                            d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                          ></path>
                        </svg>
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Tooth Card Snippet Template -->
{#snippet toothCard(toothNum: number)}
  {@const conds = getConditionsForTooth(toothNum)}
  {@const status = getToothPrimaryStatus(toothNum)}
  {@const label = getToothLabel(toothNum, currentToothSystem)}

  <button
    type="button"
    onclick={() => openAddConditionForTooth(toothNum)}
    class={`flex flex-col items-center justify-between p-2 rounded-xl border transition-all duration-150 relative group cursor-pointer ${
      status === "missing"
        ? "bg-slate-950/80 border-rose-900/60 opacity-60 hover:opacity-100"
        : status === "treatment_planned"
          ? "bg-amber-950/30 border-amber-600/60 shadow-md shadow-amber-950/50 hover:border-amber-400"
          : status === "completed"
            ? "bg-emerald-950/30 border-emerald-600/60 shadow-md shadow-emerald-950/50 hover:border-emerald-400"
            : status === "existing"
              ? "bg-blue-950/30 border-blue-600/60 shadow-md shadow-blue-950/50 hover:border-blue-400"
              : "bg-slate-950 border-slate-800 hover:border-sky-500/80 hover:bg-slate-800/40"
    }`}
  >
    <!-- Tooth Number Label -->
    <span
      class={`text-xs font-bold mb-1 font-mono transition-colors ${
        status ? "text-slate-100" : "text-slate-400 group-hover:text-sky-300"
      }`}
    >
      {label}
    </span>

    <!-- Anatomical Tooth Surface Diagram (5 Surfaces: M, D, O, F, L) -->
    <div class="relative w-8 h-8 my-1 flex items-center justify-center">
      {#if status === "missing"}
        <div
          class="absolute inset-0 flex items-center justify-center text-rose-500 font-bold text-lg"
        >
          ✕
        </div>
      {:else}
        <!-- Surface box grid representing anatomical tooth face -->
        <div
          class="w-7 h-7 border border-slate-700 grid grid-cols-3 grid-rows-3 bg-slate-900 rounded-md overflow-hidden p-0.5"
        >
          <!-- Facial/Buccal (Top) -->
          <div
            class={`col-span-3 h-1.5 transition-colors ${
              conds.some((c) =>
                c.surfaces?.includes(ToothSurface.SurfaceFacial),
              )
                ? conds.find((c) =>
                    c.surfaces?.includes(ToothSurface.SurfaceFacial),
                  )?.status === "completed"
                  ? "bg-emerald-500"
                  : conds.find((c) =>
                        c.surfaces?.includes(ToothSurface.SurfaceFacial),
                      )?.status === "treatment_planned"
                    ? "bg-amber-500"
                    : "bg-blue-500"
                : "bg-slate-800"
            }`}
          ></div>

          <!-- Mesial (Left) -->
          <div
            class={`w-1.5 h-full transition-colors ${
              conds.some((c) =>
                c.surfaces?.includes(ToothSurface.SurfaceMesial),
              )
                ? conds.find((c) =>
                    c.surfaces?.includes(ToothSurface.SurfaceMesial),
                  )?.status === "completed"
                  ? "bg-emerald-500"
                  : conds.find((c) =>
                        c.surfaces?.includes(ToothSurface.SurfaceMesial),
                      )?.status === "treatment_planned"
                    ? "bg-amber-500"
                    : "bg-blue-500"
                : "bg-slate-800"
            }`}
          ></div>

          <!-- Occlusal/Incisal (Center) -->
          <div
            class={`flex-1 h-full transition-colors ${
              conds.some(
                (c) =>
                  c.surfaces?.includes(ToothSurface.SurfaceOcclusal) ||
                  c.surfaces?.includes(ToothSurface.SurfaceIncisal),
              )
                ? conds.find(
                    (c) =>
                      c.surfaces?.includes(ToothSurface.SurfaceOcclusal) ||
                      c.surfaces?.includes(ToothSurface.SurfaceIncisal),
                  )?.status === "completed"
                  ? "bg-emerald-500"
                  : conds.find(
                        (c) =>
                          c.surfaces?.includes(ToothSurface.SurfaceOcclusal) ||
                          c.surfaces?.includes(ToothSurface.SurfaceIncisal),
                      )?.status === "treatment_planned"
                    ? "bg-amber-500"
                    : "bg-blue-500"
                : "bg-slate-800"
            }`}
          ></div>

          <!-- Distal (Right) -->
          <div
            class={`w-1.5 h-full transition-colors ${
              conds.some((c) =>
                c.surfaces?.includes(ToothSurface.SurfaceDistal),
              )
                ? conds.find((c) =>
                    c.surfaces?.includes(ToothSurface.SurfaceDistal),
                  )?.status === "completed"
                  ? "bg-emerald-500"
                  : conds.find((c) =>
                        c.surfaces?.includes(ToothSurface.SurfaceDistal),
                      )?.status === "treatment_planned"
                    ? "bg-amber-500"
                    : "bg-blue-500"
                : "bg-slate-800"
            }`}
          ></div>

          <!-- Lingual (Bottom) -->
          <div
            class={`col-span-3 h-1.5 transition-colors ${
              conds.some((c) =>
                c.surfaces?.includes(ToothSurface.SurfaceLingual),
              )
                ? conds.find((c) =>
                    c.surfaces?.includes(ToothSurface.SurfaceLingual),
                  )?.status === "completed"
                  ? "bg-emerald-500"
                  : conds.find((c) =>
                        c.surfaces?.includes(ToothSurface.SurfaceLingual),
                      )?.status === "treatment_planned"
                    ? "bg-amber-500"
                    : "bg-blue-500"
                : "bg-slate-800"
            }`}
          ></div>
        </div>
      {/if}
    </div>

    <!-- Active conditions count pill -->
    {#if conds.length > 0}
      <span
        class="mt-1 px-1.5 py-0.2 text-[9px] font-bold rounded-full bg-slate-800 text-sky-300 border border-slate-700"
      >
        {conds.length}
        {conds.length === 1 ? "entry" : "entries"}
      </span>
    {:else}
      <span
        class="mt-1 text-[9px] text-slate-600 group-hover:text-slate-400 transition-colors"
      >
        Chart
      </span>
    {/if}
  </button>
{/snippet}

<!-- Condition Modal -->
{#if showConditionModal && selectedToothNumber}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm"
  >
    <div
      class="bg-slate-900 border border-slate-800 w-full max-w-lg rounded-2xl shadow-2xl p-6 relative flex flex-col gap-5 max-h-[90vh] overflow-y-auto"
    >
      <div
        class="flex items-center justify-between border-b border-slate-800 pb-4"
      >
        <div>
          <h3 class="text-lg font-bold text-slate-100 m-0">
            Tooth {getToothLabel(selectedToothNumber, currentToothSystem)} Conditions
          </h3>
          <p class="text-xs text-slate-400 m-0">
            {countryMeta?.name || "Practice Country"} Tooth Notation & Surface Inspector
          </p>
        </div>
        <button
          type="button"
          onclick={() => (showConditionModal = false)}
          class="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800"
          aria-label="Close modal"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="w-5 h-5"
          >
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <!-- Surface Selection Grid -->
      <div class="flex flex-col gap-2">
        <span class="text-xs font-semibold text-slate-300"
          >Select Affected Surface(s):</span
        >
        <div class="grid grid-cols-3 sm:grid-cols-6 gap-2">
          {#each [{ id: "M", label: "Mesial (M)" }, { id: "D", label: "Distal (D)" }, { id: "O", label: "Occlusal (O)" }, { id: "I", label: "Incisal (I)" }, { id: "F", label: "Facial (F)" }, { id: "L", label: "Lingual (L)" }] as s}
            <button
              type="button"
              onclick={() => toggleSurface(s.id as ToothSurface)}
              class={`py-2 px-2 text-xs font-bold rounded-xl border transition-all ${
                formSurfaces.includes(s.id as ToothSurface)
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
        <span class="text-xs font-semibold text-slate-300"
          >Quick Procedure Presets:</span
        >
        <div class="flex flex-wrap gap-1.5">
          {#each procedurePresets as preset}
            <button
              type="button"
              onclick={() => applyPreset(preset)}
              class="px-2.5 py-1 text-[11px] font-medium rounded-lg bg-slate-800 text-slate-300 border border-slate-700 hover:bg-slate-700 hover:text-white transition-colors"
            >
              {preset.code ? `[${preset.code}] ` : ""}{preset.desc.split(
                " - ",
              )[0]}
            </button>
          {/each}
        </div>
      </div>

      <!-- Form Controls -->
      <form onsubmit={handleSaveCondition} class="flex flex-col gap-4">
        <div class="grid grid-cols-2 gap-3">
          <div class="flex flex-col gap-1">
            <label
              for="condition-ada-code"
              class="text-xs font-medium text-slate-400"
              >ADA Procedure Code</label
            >
            <input
              id="condition-ada-code"
              type="text"
              bind:value={formADACode}
              placeholder="e.g. D2392"
              class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label
              for="condition-fee"
              class="text-xs font-medium text-slate-400"
              >Fee {countryMeta?.default_currency
                ? `(${countryMeta.default_currency})`
                : ""}</label
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
          <label
            for="condition-description"
            class="text-xs font-medium text-slate-400"
            >Description / Finding</label
          >
          <input
            id="condition-description"
            type="text"
            bind:value={formDescription}
            required
            placeholder="e.g. 2-Surface Composite Resin"
            class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500"
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="condition-status"
            class="text-xs font-medium text-slate-400">Status</label
          >
          <select
            id="condition-status"
            bind:value={formStatus}
            class="bg-slate-950 border border-slate-700 text-slate-200 text-sm rounded-xl px-3 py-2 outline-none focus:border-sky-500"
          >
            <option value="treatment_planned">Treatment Planned</option>
            <option value="completed">Completed</option>
            <option value="existing">Existing Finding</option>
            <option value="missing">Missing Tooth / Extracted</option>
          </select>
        </div>

        <div
          class="flex items-center justify-between pt-3 border-t border-slate-800"
        >
          {#if isEditingCondition}
            <button
              type="button"
              onclick={() => handleDeleteCondition(editingConditionId)}
              class="px-3 py-2 text-xs font-semibold text-rose-400 hover:text-rose-300 hover:bg-rose-950/40 rounded-xl transition-colors"
            >
              Delete Condition
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
              Cancel
            </button>
            <button
              type="submit"
              class="btn btn-primary text-xs shadow-md shadow-sky-500/20"
            >
              {isEditingCondition ? "Update Condition" : "Save Condition"}
            </button>
          </div>
        </div>
      </form>
    </div>
  </div>
{/if}
