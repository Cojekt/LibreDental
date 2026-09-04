<script lang="ts">
  import type { CountryConfig } from "@bindings/domain/models.js";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion, getLanguageName, useLanguageState } from "$lib/locale.svelte.js";
  import { locales } from "../paraglide/runtime.js";
  import ModalGridSelect from "./ui/ModalGridSelect.svelte";

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

  const langState = useLanguageState();

  function getFlagEmoji(countryCode: string) {
    if (!countryCode) return "";
    const codePoints = countryCode
      .toUpperCase()
      .split("")
      .map((char) => 127397 + char.charCodeAt(0));
    return String.fromCodePoint(...codePoints);
  }

  function handleSubmit(e: Event) {
    e.preventDefault();
    if (!selectedCountry) return;
    isSubmitting = true;
    oncomplete(selectedCountry);
  }
</script>

{#if showOnboarding}
  {@const _ = getLocaleVersion()}
  <div
    class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/90 backdrop-blur-md p-4 overflow-y-auto"
  >
    <div
      class="w-full max-w-4xl rounded-2xl border border-slate-700/80 bg-slate-900 p-8 shadow-2xl my-auto"
    >
      <div class="flex items-center gap-4 mb-8">
        <img
          src="/sourceicon.svg"
          alt="LibreDental Logo"
          class="h-10 w-10 rounded-xl shadow-md shadow-purple-500/20 object-contain"
        />
        <h2 class="m-0 text-2xl font-bold text-white tracking-tight">
          {(getLocaleVersion(), m.onboarding_title())}
        </h2>
      </div>

      <p class="text-base text-slate-300 leading-relaxed mb-8">
        {m.onboarding_body()}
      </p>

      <form onsubmit={handleSubmit} class="flex flex-col gap-8">
        <!-- Language Selector -->
        <div class="flex flex-col gap-2">
          <label
            for="onboard-language"
            class="text-sm font-semibold text-slate-400 uppercase tracking-wide"
          >
            {m.settings_section_language ? m.settings_section_language() : "Language"}
          </label>
          <select
            id="onboard-language"
            value={langState.selectedLanguage}
            onchange={langState.handleSelectLanguage}
            class="w-full max-w-xs rounded-lg border border-slate-700 bg-slate-950 px-4 py-3 text-base text-white focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-all cursor-pointer"
          >
            {#each locales as locale}
              <option value={locale}>{getLanguageName(locale)}</option>
            {/each}
          </select>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
          <!-- Country Picker -->
          <div class="flex flex-col gap-2">
            <label
              for="country-select-btn"
              class="text-sm font-semibold text-slate-400 uppercase tracking-wide"
            >
              {m.onboarding_country_label()}
            </label>

            <ModalGridSelect
              id="country-select-btn"
              bind:value={selectedCountry}
              options={supportedCountries.map((c: CountryConfig) => ({
                value: c.code,
                label: c.name,
                code: c.code,
              }))}
              placeholder={m.onboarding_country_placeholder()}
              modalTitle={m.onboarding_country_modal_title()}
              buttonClass="flex flex-col items-center justify-center w-full p-4 h-32 border-2 border-dashed border-slate-700 hover:border-blue-500 bg-slate-900/50 hover:bg-slate-800/80 rounded-lg text-slate-100 transition-colors"
              hideChevron={true}
            >
              {#snippet buttonContent(selected)}
                <div class="flex flex-col items-center gap-2">
                  <span class="text-4xl leading-none">{getFlagEmoji(selected.code)}</span>
                  <span class="font-bold text-lg">{selected.label}</span>
                  <span class="text-xs text-blue-400 font-semibold uppercase tracking-wider"
                    >{m.onboarding_change_region()}</span
                  >
                </div>
              {/snippet}

              {#snippet optionContent(option)}
                <div class="text-4xl leading-none">{getFlagEmoji(option.code)}</div>
                <div class="font-medium text-sm text-center">{option.label}</div>
              {/snippet}
            </ModalGridSelect>
          </div>

          <!-- Region Config Details -->
          <div>
            {#if selectedCountry}
              {@const meta = supportedCountries.find(
                (c: CountryConfig) => c.code === selectedCountry
              )}
              {#if meta}
                <div
                  class="rounded-xl border border-slate-800 bg-slate-950/60 p-6 h-full text-sm text-slate-400 space-y-4"
                >
                  <div class="font-medium text-slate-200 text-base mb-2">
                    {m.onboarding_regional_config_title()}
                  </div>
                  <div class="flex flex-col border-b border-slate-800/80 pb-3">
                    <span class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-1"
                      >{m.onboarding_field_national_id()}</span
                    >
                    <span class="text-blue-400 font-medium text-base">{meta.national_id_name}</span>
                  </div>
                  <div class="flex flex-col border-b border-slate-800/80 pb-3">
                    <span class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-1"
                      >{m.onboarding_field_tooth_system()}</span
                    >
                    <span class="text-blue-400 font-medium text-base">
                      {meta.default_tooth_system === "universal"
                        ? m.onboarding_tooth_universal()
                        : m.onboarding_tooth_fdi()}
                    </span>
                  </div>
                  <div class="flex flex-col border-b border-slate-800/80 pb-3">
                    <span class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-1"
                      >{m.onboarding_field_address()}</span
                    >
                    <span class="text-slate-300 text-base"
                      >{meta.state_province_label} & {meta.postal_code_label}</span
                    >
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm font-semibold text-slate-400 uppercase tracking-wide mb-1"
                      >{m.onboarding_field_currency()}</span
                    >
                    <span class="text-slate-300 text-base">{meta.default_currency}</span>
                  </div>
                </div>
              {/if}
            {/if}
          </div>
        </div>

        <div class="flex justify-end mt-4">
          <button
            type="submit"
            disabled={isSubmitting}
            class="w-full md:w-auto px-10 rounded-lg bg-blue-600 hover:bg-blue-500 py-3 text-base font-semibold text-white shadow-lg shadow-blue-600/20 transition-all disabled:opacity-50 cursor-pointer"
          >
            {isSubmitting ? m.onboarding_submitting() : m.onboarding_submit()}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
