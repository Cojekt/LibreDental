<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService, NotificationService } from "@bindings/services/index.js";
  import { m } from "../../paraglide/messages.js";
  import { auth } from "../../stores/auth.svelte.js";

  let { canEdit = false } = $props<{ canEdit: boolean }>();

  // Generic list-providers / get-config / set-config panel state, parameterized by which
  // Wails service backs it (BillingService for claims clearinghouses, NotificationService
  // for email/SMS/voice vendors), both backed by SecretsService on the Go side.
  // Both services' config methods require a session token, so each is adapted below.
  type ProviderConfig = { [key: string]: string | undefined } | null;
  type ProviderConfigService = {
    ListProviders(): Promise<string[] | null>;
    GetProviderConfig(name: string): Promise<ProviderConfig>;
    SetProviderConfig(name: string, config: ProviderConfig): Promise<void>;
  };

  function createProviderPanel(service: ProviderConfigService) {
    let providers = $state<string[]>([]);
    let providersLoaded = $state(false);
    let providersLoadError = $state(false);
    let selectedProvider = $state("");
    let providerApiKey = $state("");
    let isSavingConfig = $state(false);
    let providerConfigError = $state(false);
    let providerFullConfig = $state<{ [key: string]: string | undefined }>({});
    let isLoadingConfig = $state(false);
    let saveStatus = $state<{ ok: boolean; msg: string } | null>(null);

    async function loadProviders() {
      providersLoadError = false;
      try {
        const list = await service.ListProviders();
        providers = list || [];
      } catch (e) {
        console.error("Failed to load providers:", e);
        providersLoadError = true;
      } finally {
        providersLoaded = true;
      }
    }

    async function loadProviderConfig() {
      saveStatus = null;
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
      if (!canEdit || !selectedProvider || providerConfigError) return;
      isSavingConfig = true;
      saveStatus = null;
      const reqProvider = selectedProvider;
      try {
        await service.SetProviderConfig(reqProvider, {
          ...providerFullConfig,
          api_key: providerApiKey,
        });
        if (reqProvider === selectedProvider) {
          saveStatus = { ok: true, msg: m.integrations_save_success() };
        }
      } catch (e) {
        console.error("Failed to save provider config:", e);
        if (reqProvider === selectedProvider) {
          saveStatus = { ok: false, msg: m.integrations_save_error() };
        }
      } finally {
        isSavingConfig = false;
      }
    }

    return {
      get providers() {
        return providers;
      },
      get noProviders() {
        return providersLoaded && !providersLoadError && providers.length === 0;
      },
      get providersLoadError() {
        return providersLoadError;
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
      get saveStatus() {
        return saveStatus;
      },
      loadProviders,
      loadProviderConfig,
      saveProviderConfig,
    };
  }

  const claimsPanel = createProviderPanel({
    ListProviders: () => BillingService.ListProviders(),
    GetProviderConfig: (name) => BillingService.GetProviderConfig(auth.token, name),
    SetProviderConfig: (name, config) => BillingService.SetProviderConfig(auth.token, name, config),
  });
  const notificationsPanel = createProviderPanel({
    ListProviders: () => NotificationService.ListProviders(),
    GetProviderConfig: (name) => NotificationService.GetProviderConfig(auth.token, name),
    SetProviderConfig: (name, config) =>
      NotificationService.SetProviderConfig(auth.token, name, config),
  });

  onMount(() => {
    claimsPanel.loadProviders();
    notificationsPanel.loadProviders();
  });
</script>

<div class="space-y-8 animate-fadeIn">
  <div>
    <h3 class="text-lg font-bold text-slate-100 mb-1">{m.integrations_title()}</h3>
    <p class="text-sm text-slate-400 mb-6">{m.integrations_subtitle()}</p>

    <div class="space-y-6">
      <!-- Claims Integrations (US) Section -->
      <div>
        <span class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2"
          >{m.integrations_section_claims_us()}
        </span>

        <div class="space-y-3 rounded-xl border border-slate-800 bg-slate-950/80 p-4">
          {#if claimsPanel.providersLoadError}
            <p class="text-xs text-red-400">{m.integrations_providers_load_error()}</p>
          {:else if claimsPanel.noProviders}
            <p class="text-xs text-slate-500">{m.integrations_claims_no_providers()}</p>
          {/if}

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="claims-provider-select" class="block text-xs text-slate-400 mb-1"
                >{m.integrations_label_provider()}</label
              >
              <select
                id="claims-provider-select"
                bind:value={claimsPanel.selectedProvider}
                onchange={claimsPanel.loadProviderConfig}
                disabled={claimsPanel.noProviders}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
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
                disabled={!canEdit || !claimsPanel.selectedProvider || claimsPanel.isLoadingConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              />
            </div>
          </div>

          <div class="flex items-center justify-end gap-3">
            {#if claimsPanel.saveStatus}
              <span
                class="text-xs font-semibold {claimsPanel.saveStatus.ok
                  ? 'text-emerald-400'
                  : 'text-rose-400'}"
                role={claimsPanel.saveStatus.ok ? "status" : "alert"}
                >{claimsPanel.saveStatus.msg}</span
              >
            {/if}
            <button
              type="button"
              class="btn btn-secondary btn-sm bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
              disabled={!canEdit ||
                !claimsPanel.selectedProvider ||
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
          {#if notificationsPanel.providersLoadError}
            <p class="text-xs text-red-400">{m.integrations_providers_load_error()}</p>
          {:else if notificationsPanel.noProviders}
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
                disabled={notificationsPanel.noProviders}
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
                disabled={!canEdit ||
                  !notificationsPanel.selectedProvider ||
                  notificationsPanel.isLoadingConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              />
            </div>
          </div>

          <div class="flex items-center justify-end gap-3">
            {#if notificationsPanel.saveStatus}
              <span
                class="text-xs font-semibold {notificationsPanel.saveStatus.ok
                  ? 'text-emerald-400'
                  : 'text-rose-400'}"
                role={notificationsPanel.saveStatus.ok ? "status" : "alert"}
                >{notificationsPanel.saveStatus.msg}</span
              >
            {/if}
            <button
              type="button"
              class="btn btn-secondary btn-sm bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
              disabled={!canEdit ||
                !notificationsPanel.selectedProvider ||
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
