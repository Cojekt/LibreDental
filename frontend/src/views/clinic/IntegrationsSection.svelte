<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService, NotificationService } from "@bindings/services/index.js";
  import { m } from "../../paraglide/messages.js";

  // Generic list-providers / get-config / set-config panel state, parameterized by which
  // Wails service backs it (BillingService for claims clearinghouses, NotificationService
  // for email/SMS/voice vendors) — both expose the identical ListProviders/Get/SetProviderConfig
  // shape backed by SecretsService on the Go side.
  type ProviderConfig = { [key: string]: string | undefined } | null;
  type ProviderConfigService = {
    ListProviders(): Promise<string[] | null>;
    GetProviderConfig(name: string): Promise<ProviderConfig>;
    SetProviderConfig(name: string, config: ProviderConfig): Promise<void>;
  };

  function createProviderPanel(service: ProviderConfigService) {
    let providers = $state<string[]>([]);
    let selectedProvider = $state("");
    let providerApiKey = $state("");
    let isSavingConfig = $state(false);
    let providerConfigError = $state(false);
    let providerFullConfig = $state<{ [key: string]: string | undefined }>({});
    let isLoadingConfig = $state(false);

    async function loadProviders() {
      try {
        const list = await service.ListProviders();
        providers = list || [];
      } catch (e) {
        console.error("Failed to load providers:", e);
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
        const config = await service.GetProviderConfig(reqProvider);
        if (reqProvider !== selectedProvider) return;
        providerFullConfig = config || {};
        providerApiKey = (config && config["api_key"]) || "";
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
        await service.SetProviderConfig(selectedProvider, {
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

    return {
      get providers() {
        return providers;
      },
      get selectedProvider() {
        return selectedProvider;
      },
      set selectedProvider(v: string) {
        selectedProvider = v;
      },
      get providerApiKey() {
        return providerApiKey;
      },
      set providerApiKey(v: string) {
        providerApiKey = v;
      },
      get isSavingConfig() {
        return isSavingConfig;
      },
      get isLoadingConfig() {
        return isLoadingConfig;
      },
      get providerConfigError() {
        return providerConfigError;
      },
      loadProviders,
      loadProviderConfig,
      saveProviderConfig,
    };
  }

  const claimsPanel = createProviderPanel(BillingService);
  const notificationsPanel = createProviderPanel(NotificationService);

  onMount(() => {
    claimsPanel.loadProviders();
    notificationsPanel.loadProviders();
  });
</script>

<div class="space-y-8 animate-fadeIn">
  <div>
    <h3 class="text-lg font-bold text-white mb-1">{m.integrations_title()}</h3>
    <p class="text-sm text-slate-400 mb-6">{m.integrations_subtitle()}</p>

    <div class="space-y-6">
      <!-- Claims Integrations (US) Section -->
      <div>
        <span class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2"
          >{m.integrations_section_claims_us()}
        </span>

        <div class="space-y-3 rounded-xl border border-slate-800 bg-slate-950/80 p-4">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="claims-provider-select" class="block text-xs text-slate-400 mb-1"
                >{m.integrations_label_provider()}</label
              >
              <select
                id="claims-provider-select"
                bind:value={claimsPanel.selectedProvider}
                onchange={claimsPanel.loadProviderConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
              >
                <option value="">{m.integrations_placeholder_provider()}</option>
                {#each claimsPanel.providers as p}
                  <option value={p}>{p}</option>
                {/each}
              </select>
            </div>

            <div>
              <label for="claims-provider-api-key" class="block text-xs text-slate-400 mb-1"
                >{m.integrations_label_api_key()}</label
              >
              <input
                type="password"
                id="claims-provider-api-key"
                bind:value={claimsPanel.providerApiKey}
                placeholder={m.integrations_placeholder_api_key()}
                disabled={!claimsPanel.selectedProvider || claimsPanel.isLoadingConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              />
            </div>
          </div>

          <div class="flex justify-end">
            <button
              type="button"
              class="btn btn-secondary btn-sm bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
              disabled={!claimsPanel.selectedProvider ||
                claimsPanel.isSavingConfig ||
                claimsPanel.isLoadingConfig ||
                claimsPanel.providerConfigError}
              onclick={claimsPanel.saveProviderConfig}
            >
              {claimsPanel.isSavingConfig ? m.integrations_btn_saving() : m.integrations_btn_save()}
            </button>
          </div>
        </div>
      </div>

      <!-- Patient Notifications (Email/SMS/Voice) Section -->
      <div>
        <span class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2"
          >{m.integrations_section_notifications()}
        </span>

        <div class="space-y-3 rounded-xl border border-slate-800 bg-slate-950/80 p-4">
          {#if notificationsPanel.providers.length === 0}
            <p class="text-xs text-slate-500">{m.integrations_notifications_no_providers()}</p>
          {/if}

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="notification-provider-select" class="block text-xs text-slate-400 mb-1"
                >{m.integrations_label_provider()}</label
              >
              <select
                id="notification-provider-select"
                bind:value={notificationsPanel.selectedProvider}
                onchange={notificationsPanel.loadProviderConfig}
                disabled={notificationsPanel.providers.length === 0}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              >
                <option value="">{m.integrations_placeholder_provider()}</option>
                {#each notificationsPanel.providers as p}
                  <option value={p}>{p}</option>
                {/each}
              </select>
            </div>

            <div>
              <label for="notification-provider-api-key" class="block text-xs text-slate-400 mb-1"
                >{m.integrations_label_api_key()}</label
              >
              <input
                type="password"
                id="notification-provider-api-key"
                bind:value={notificationsPanel.providerApiKey}
                placeholder={m.integrations_placeholder_api_key()}
                disabled={!notificationsPanel.selectedProvider ||
                  notificationsPanel.isLoadingConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              />
            </div>
          </div>

          <div class="flex justify-end">
            <button
              type="button"
              class="btn btn-secondary btn-sm bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
              disabled={!notificationsPanel.selectedProvider ||
                notificationsPanel.isSavingConfig ||
                notificationsPanel.isLoadingConfig ||
                notificationsPanel.providerConfigError}
              onclick={notificationsPanel.saveProviderConfig}
            >
              {notificationsPanel.isSavingConfig
                ? m.integrations_btn_saving()
                : m.integrations_btn_save()}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
