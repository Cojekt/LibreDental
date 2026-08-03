<script lang="ts">
  import type { CountryConfig } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";

  let {
    showOnboarding = $bindable(),
    supportedCountries = [],
    oncomplete,
  } = $props<{
    showOnboarding: boolean;
    supportedCountries: CountryConfig[];
    oncomplete: (countryCode: string) => void;
  }>();

  let selectedCountry = $state("US");
  let isSubmitting = $state(false);

  function handleSubmit(e: Event) {
    e.preventDefault();
    if (!selectedCountry) return;
    isSubmitting = true;
    oncomplete(selectedCountry);
  }
</script>

{#if showOnboarding}
  <div class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/90 backdrop-blur-md p-4">
    <div class="w-full max-w-[560px] rounded-2xl border border-slate-700/80 bg-slate-900 p-8 shadow-2xl">
      <div class="flex items-center gap-3 mb-4">
        <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-600/20 text-blue-400 font-bold text-lg border border-blue-500/30">
          📍
        </div>
        <div>
          <h2 class="m-0 text-xl font-semibold text-white tracking-tight">Welcome to LibreDental™</h2>
          <p class="m-0 text-xs text-slate-400">One-time initial practice setup</p>
        </div>
      </div>

      <p class="text-sm text-slate-300 leading-relaxed mb-6">
        Please select the country where your dental practice is located. This will automatically configure region-specific compliance rules, national identity fields, tooth numbering standards, and address formats.
      </p>

      <form onsubmit={handleSubmit} class="flex flex-col gap-5">
        <div class="flex flex-col gap-2">
          <label for="country-select" class="text-xs font-semibold text-slate-300 uppercase tracking-wider">
            Practice Country / Region Jurisdiction *
          </label>
          <select
            id="country-select"
            bind:value={selectedCountry}
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-sm text-white focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-all cursor-pointer"
          >
            {#each supportedCountries as country}
              <option value={country.code}>
                {country.name} ({country.default_currency} • Tooth System: {country.default_tooth_system.toUpperCase()})
              </option>
            {/each}
          </select>
        </div>

        {#if selectedCountry}
          {@const meta = supportedCountries.find((c: CountryConfig) => c.code === selectedCountry)}
          {#if meta}
            <div class="rounded-xl border border-slate-800 bg-slate-950/60 p-4 text-xs text-slate-400 space-y-2">
              <div class="font-medium text-slate-200 mb-1">Automatic Regional Configuration:</div>
              <div class="flex justify-between border-b border-slate-800/80 pb-1">
                <span>National Identifier:</span>
                <span class="text-blue-400 font-medium">{meta.national_id_name}</span>
              </div>
              <div class="flex justify-between border-b border-slate-800/80 pb-1">
                <span>Tooth Numbering Notation:</span>
                <span class="text-blue-400 font-medium">
                  {meta.default_tooth_system === "universal" ? "Universal System (1-32 / US)" : "FDI Standard (11-48 / ISO 3950)"}
                </span>
              </div>
              <div class="flex justify-between border-b border-slate-800/80 pb-1">
                <span>Address Format:</span>
                <span class="text-slate-300">{meta.state_province_label} & {meta.postal_code_label}</span>
              </div>
              <div class="flex justify-between">
                <span>Practice Currency:</span>
                <span class="text-slate-300">{meta.default_currency}</span>
              </div>
            </div>
          {/if}
        {/if}

        <button
          type="submit"
          disabled={isSubmitting}
          class="w-full rounded-lg bg-blue-600 hover:bg-blue-500 py-3 text-sm font-semibold text-white shadow-lg shadow-blue-600/20 transition-all disabled:opacity-50 cursor-pointer mt-2"
        >
          {isSubmitting ? "Configuring Practice..." : "Save & Complete Setup"}
        </button>
      </form>
    </div>
  </div>
{/if}
