<script lang="ts">
  import type { ToothCondition, CountryConfig } from "@bindings/domain/models.js";
  import { ToothSystem, ToothSurface } from "@bindings/domain/models.js";
  import { m } from "../../paraglide/messages.js";

  let {
    dentitionType = "adult",
    currentToothSystem,
    loadingChart = false,
    upperAdultRight,
    upperAdultLeft,
    upperPrimaryRight,
    upperPrimaryLeft,
    lowerAdultRight,
    lowerAdultLeft,
    lowerPrimaryRight,
    lowerPrimaryLeft,
    getConditionsForTooth,
    getToothPrimaryStatus,
    getToothLabel,
    openAddConditionForTooth,
  } = $props<{
    dentitionType: "adult" | "primary";
    currentToothSystem: ToothSystem;
    loadingChart: boolean;
    upperAdultRight: number[];
    upperAdultLeft: number[];
    upperPrimaryRight: number[];
    upperPrimaryLeft: number[];
    lowerAdultRight: number[];
    lowerAdultLeft: number[];
    lowerPrimaryRight: number[];
    lowerPrimaryLeft: number[];
    getConditionsForTooth: (num: number) => ToothCondition[];
    getToothPrimaryStatus: (num: number) => string | null;
    getToothLabel: (num: number, system: ToothSystem) => string;
    openAddConditionForTooth: (num: number) => void;
  }>();

  function getSurfaceFill(conds: ToothCondition[], s1: ToothSurface, s2?: ToothSurface) {
    const match = conds.find(
      (c: ToothCondition) => c.surfaces?.includes(s1) || (s2 && c.surfaces?.includes(s2))
    );
    if (!match) return "fill-slate-800 hover:fill-slate-700/80";
    if (match.status === "completed") return "fill-emerald-500 hover:fill-emerald-400";
    if (match.status === "treatment_planned") return "fill-amber-500 hover:fill-amber-400";
    return "fill-blue-500 hover:fill-blue-400";
  }
</script>

<div class="flex flex-col gap-6">
  <!-- Status Legend -->
  <div
    class="flex flex-wrap items-center justify-between gap-4 bg-slate-900/80 border border-slate-800/80 rounded-xl px-5 py-3 text-xs"
  >
    <div class="flex items-center gap-5">
      <span class="font-semibold text-slate-300">{m.charting_legend_title()}</span>
      <div class="flex items-center gap-1.5">
        <span class="w-3 h-3 rounded-full bg-blue-500 shadow-sm shadow-blue-500/50"></span>
        <span class="text-slate-300">{m.charting_legend_existing()}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="w-3 h-3 rounded-full bg-amber-500 shadow-sm shadow-amber-500/50"></span>
        <span class="text-slate-300">{m.charting_legend_planned()}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="w-3 h-3 rounded-full bg-emerald-500 shadow-sm shadow-emerald-500/50"></span>
        <span class="text-slate-300">{m.charting_legend_completed()}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="w-3 h-3 rounded-full bg-rose-500/80 border border-rose-400"></span>
        <span class="text-slate-300">{m.charting_legend_missing()}</span>
      </div>
    </div>
    <div class="text-slate-400 text-[11px]">{m.charting_guidance_text()}</div>
  </div>

  <!-- Odontogram Chart Layout -->
  <div
    class="flex flex-col gap-6 bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-xl relative"
  >
    {#if loadingChart}
      <div
        class="absolute inset-0 bg-slate-950/60 backdrop-blur-sm flex items-center justify-center rounded-2xl z-20"
      >
        <div class="flex items-center gap-2 text-sky-400 text-sm font-semibold">
          <svg
            class="animate-spin h-5 w-5 text-sky-400"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          {m.common_loading()}
        </div>
      </div>
    {/if}

    <!-- Maxillary Arch (Upper Teeth) -->
    <div>
      <div class="flex items-center justify-between mb-3 border-b border-slate-800 pb-2">
        <span class="text-xs font-bold uppercase tracking-wider text-sky-400"
          >{m.charting_upper_arch()}</span
        >
        <span class="text-[11px] text-slate-500 font-mono">{m.charting_arch_direction()}</span>
      </div>

      <div class="grid grid-cols-8 sm:grid-cols-16 gap-2 sm:gap-2.5">
        {#if dentitionType === "adult"}
          {#each upperAdultRight as toothNum}
            {@render toothCard(toothNum)}
          {/each}
          {#each upperAdultLeft as toothNum}
            {@render toothCard(toothNum)}
          {/each}
        {:else}
          <div class="col-span-3"></div>
          {#each upperPrimaryRight as toothNum}
            {@render toothCard(toothNum)}
          {/each}
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
      <div class="flex items-center justify-between mb-3 border-b border-slate-800 pb-2">
        <span class="text-xs font-bold uppercase tracking-wider text-sky-400"
          >{m.charting_lower_arch()}</span
        >
        <span class="text-[11px] text-slate-500 font-mono">{m.charting_arch_direction()}</span>
      </div>

      <div class="grid grid-cols-8 sm:grid-cols-16 gap-2 sm:gap-2.5">
        {#if dentitionType === "adult"}
          {#each lowerAdultRight.slice().reverse() as toothNum}
            {@render toothCard(toothNum)}
          {/each}
          {#each lowerAdultLeft.slice().reverse() as toothNum}
            {@render toothCard(toothNum)}
          {/each}
        {:else}
          <div class="col-span-3"></div>
          {#each lowerPrimaryRight.slice().reverse() as toothNum}
            {@render toothCard(toothNum)}
          {/each}
          {#each lowerPrimaryLeft.slice().reverse() as toothNum}
            {@render toothCard(toothNum)}
          {/each}
          <div class="col-span-3"></div>
        {/if}
      </div>
    </div>
  </div>
</div>

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
    <span
      class={`text-xs font-bold mb-1 font-mono transition-colors ${
        status ? "text-slate-100" : "text-slate-400 group-hover:text-sky-300"
      }`}
    >
      {label}
    </span>

    <div class="relative w-8 h-8 my-1 flex items-center justify-center">
      {#if status === "missing"}
        <div
          class="absolute inset-0 flex items-center justify-center text-rose-500 font-bold text-lg"
        >
          ✕
        </div>
      {:else}
        <svg viewBox="0 0 100 100" class="w-8 h-8 drop-shadow-sm">
          <g stroke="#1e293b" stroke-width="4" stroke-linejoin="round">
            <!-- Facial -->
            <path
              d="M 14.64 14.64 A 50 50 0 0 1 85.35 14.64 L 70.71 29.29 A 29 29 0 0 0 29.29 29.29 Z"
              class={`transition-colors ${getSurfaceFill(conds, ToothSurface.SurfaceFacial)}`}
            />
            <!-- Mesial -->
            <path
              d="M 14.64 85.35 A 50 50 0 0 1 14.64 14.64 L 29.29 29.29 A 29 29 0 0 0 29.29 70.71 Z"
              class={`transition-colors ${getSurfaceFill(conds, ToothSurface.SurfaceMesial)}`}
            />
            <!-- Distal -->
            <path
              d="M 85.35 14.64 A 50 50 0 0 1 85.35 85.35 L 70.71 70.71 A 29 29 0 0 0 70.71 29.29 Z"
              class={`transition-colors ${getSurfaceFill(conds, ToothSurface.SurfaceDistal)}`}
            />
            <!-- Lingual -->
            <path
              d="M 85.35 85.35 A 50 50 0 0 1 14.64 85.35 L 29.29 70.71 A 29 29 0 0 0 70.71 70.71 Z"
              class={`transition-colors ${getSurfaceFill(conds, ToothSurface.SurfaceLingual)}`}
            />
            <!-- Occlusal / Incisal -->
            <circle
              cx="50"
              cy="50"
              r="29"
              class={`transition-colors ${getSurfaceFill(conds, ToothSurface.SurfaceOcclusal, ToothSurface.SurfaceIncisal)}`}
            />
          </g>
        </svg>
      {/if}
    </div>

    {#if conds.length > 0}
      <span
        class="mt-1 px-1.5 py-0.2 text-[9px] font-bold rounded-full bg-slate-800 text-sky-300 border border-slate-700"
      >
        {conds.length}
        {conds.length === 1 ? "entry" : "entries"}
      </span>
    {:else}
      <span class="mt-1 text-[9px] text-slate-600 group-hover:text-slate-400 transition-colors">
        Chart
      </span>
    {/if}
  </button>
{/snippet}
