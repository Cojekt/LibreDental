<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService } from "@bindings/services/index.js";
  import { m } from "../../paraglide/messages.js";

  let providers = $state<string[]>([]);
  let selectedProvider = $state("");
  let providerApiKey = $state("");
  let isSavingConfig = $state(false);
  let providerConfigError = $state(false);
  let providerFullConfig = $state<Record<string, string>>({});
  let isLoadingConfig = $state(false);

  async function loadProviders() {
    try {
      const list = await BillingService.ListProviders();
      providers = list || [];
    } catch (e) {
      console.error("Failed to load claim providers:", e);
    }
  }

  async function loadProviderConfig() {
    providerConfigError = false;
    providerFullConfig = {};
    providerApiKey = "";
    if (!selectedProvider) {
      isLoadingConfig = false;
      return;
    }
    isLoadingConfig = true;
    const reqProvider = selectedProvider;
    try {
      const config = await BillingService.GetProviderConfig(reqProvider);
      if (reqProvider !== selectedProvider) return;
      providerFullConfig = (config as Record<string, string>) || {};
      if (config && config["api_key"]) {
        providerApiKey = config["api_key"];
      } else {
        providerApiKey = "";
      }
    } catch (e) {
      if (reqProvider !== selectedProvider) return;
      console.error("Failed to load provider config:", e);
      providerConfigError = true;
    } finally {
      if (reqProvider === selectedProvider) {
        isLoadingConfig = false;
      }
    }
  }

  async function saveProviderConfig() {
    if (!selectedProvider || providerConfigError) return;
    isSavingConfig = true;
    try {
      await BillingService.SetProviderConfig(selectedProvider, {
        ...providerFullConfig,
        api_key: providerApiKey,
      });
      alert(m.integrations_save_success());
    } catch (e) {
      console.error("Failed to save provider config:", e);
      alert(m.integrations_save_error());
    } finally {
      isSavingConfig = false;
    }
  }

  onMount(() => {
    loadProviders();
  });
</script>

<div class="space-y-8 animate-fadeIn">
  <div>
    <h3 class="text-lg font-bold text-white mb-1">{m.integrations_title()}</h3>
    <p class="text-sm text-slate-400 mb-6">{m.integrations_subtitle()}</p>

    <div class="space-y-6">
      <!-- Claims Integrations (US) Section -->
      <div>
        <span class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2">
          {m.integrations_section_claims_us()}
        </span>

        <div class="space-y-3 rounded-xl border border-slate-800 bg-slate-950/80 p-4">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="provider-select" class="block text-xs text-slate-400 mb-1"
                >{m.integrations_label_provider()}</label
              >
              <select
                id="provider-select"
                bind:value={selectedProvider}
                onchange={loadProviderConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
              >
                <option value="">{m.integrations_placeholder_provider()}</option>
                {#each providers as p}
                  <option value={p}>{p}</option>
                {/each}
              </select>
            </div>

            <div>
              <label for="provider-api-key" class="block text-xs text-slate-400 mb-1"
                >{m.integrations_label_api_key()}</label
              >
              <input
                type="password"
                id="provider-api-key"
                bind:value={providerApiKey}
                placeholder={m.integrations_placeholder_api_key()}
                disabled={!selectedProvider || isLoadingConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              />
            </div>
          </div>

          <div class="flex justify-end">
            <button
              type="button"
              class="btn btn-secondary btn-sm bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
              disabled={!selectedProvider ||
                isSavingConfig ||
                isLoadingConfig ||
                providerConfigError}
              onclick={saveProviderConfig}
            >
              {isSavingConfig ? m.integrations_btn_saving() : m.integrations_btn_save()}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
