<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService, NotificationService } from "@bindings/services/index.js";
  import { m } from "../../paraglide/messages.js";
  import { auth } from "../../stores/auth.svelte.js";
  import { handleError } from "$lib/error.js";

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

  // Credential fields the backend never returns: it sends REDACTED to show one is stored. On
  // save, a secret left out keeps its stored value, so untouched secrets are never sent. Must
  // match secretConfigKeys in secrets_service.go.
  const SECRET_KEYS = ["api_key", "password", "secret_access_key"];
  const REDACTED = "********";

  // apiKey: whether the panel has the single API key field (claims) or only provider-specific
  // fields edited through fieldValue/setFieldValue (notifications).
  function createProviderPanel(service: ProviderConfigService, options: { apiKey: boolean }) {
    let providers = $state<string[]>([]);
    let providersLoaded = $state(false);
    let providersLoadError = $state(false);
    let selectedProvider = $state("");
    let providerApiKey = $state("");
    let isSavingConfig = $state(false);
    let providerConfigError = $state(false);
    let providerFullConfig = $state<{ [key: string]: string | undefined }>({});
    // Secret fields with a stored value. They're shown empty rather than as REDACTED, which
    // would otherwise be saved as part of the credential if someone typed after it.
    let savedSecrets = $state<{ [key: string]: boolean }>({});
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
      savedSecrets = {};
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
        const loaded = { ...(config || {}) };
        const saved: { [key: string]: boolean } = {};
        for (const key of SECRET_KEYS) {
          if (loaded[key] === REDACTED) {
            saved[key] = true;
            loaded[key] = "";
          }
        }
        providerFullConfig = loaded;
        savedSecrets = saved;
        providerApiKey = loaded["api_key"] || "";
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
        const config: { [key: string]: string | undefined } = { ...providerFullConfig };
        if (options.apiKey) config.api_key = providerApiKey;
        // An empty secret field means "keep the saved one", so leave it out.
        for (const key of SECRET_KEYS) {
          if (!config[key]) delete config[key];
        }
        await service.SetProviderConfig(reqProvider, config);
        if (reqProvider === selectedProvider) {
          // Typed secrets are now stored; don't keep them in the form as plain text.
          const cleared = { ...providerFullConfig };
          const saved = { ...savedSecrets };
          for (const key of SECRET_KEYS) {
            if (config[key]) {
              saved[key] = true;
              cleared[key] = "";
            }
          }
          providerFullConfig = cleared;
          savedSecrets = saved;
          providerApiKey = "";
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
      // Claim providers treat anything but an explicit "false" as test mode, so an unset
      // value shows as on.
      get testMode() {
        return providerFullConfig["test_mode"] !== "false";
      },
      set testMode(v: boolean) {
        providerFullConfig = { ...providerFullConfig, test_mode: v ? "true" : "false" };
      },
      get saveStatus() {
        return saveStatus;
      },
      fieldValue(key: string): string {
        return providerFullConfig[key] ?? "";
      },
      secretSaved(key: string): boolean {
        return !!savedSecrets[key];
      },
      setFieldValue(key: string, value: string) {
        providerFullConfig = { ...providerFullConfig, [key]: value };
      },
      loadProviders,
      loadProviderConfig,
      saveProviderConfig,
    };
  }

  const claimsPanel = createProviderPanel(
    {
      ListProviders: () => BillingService.ListProviders(),
      GetProviderConfig: (name) => BillingService.GetProviderConfig(auth.token, name),
      SetProviderConfig: (name, config) =>
        BillingService.SetProviderConfig(auth.token, name, config),
    },
    { apiKey: true }
  );
  const notificationsPanel = createProviderPanel(
    {
      ListProviders: () => NotificationService.ListProviders(),
      GetProviderConfig: (name) => NotificationService.GetProviderConfig(auth.token, name),
      SetProviderConfig: (name, config) =>
        NotificationService.SetProviderConfig(auth.token, name, config),
    },
    { apiKey: false }
  );

  // Settings each notification provider reads, keyed by provider name. Keys must match the Go
  // providers; secret fields come back from the backend redacted.
  type NotificationField = {
    key: string;
    label: () => string;
    placeholder?: () => string;
    type?: "text" | "password" | "number";
    options?: { value: string; label: () => string }[];
  };
  const notificationFields: Record<string, NotificationField[]> = {
    smtp_email: [
      {
        key: "host",
        label: m.integrations_smtp_host,
        placeholder: m.integrations_smtp_host_placeholder,
      },
      {
        key: "tls_mode",
        label: m.integrations_smtp_tls_mode,
        options: [
          { value: "starttls", label: m.integrations_smtp_tls_starttls },
          { value: "implicit", label: m.integrations_smtp_tls_implicit },
        ],
      },
      {
        key: "port",
        label: m.integrations_smtp_port,
        placeholder: m.integrations_smtp_port_placeholder,
        type: "number",
      },
      { key: "username", label: m.integrations_smtp_username },
      { key: "password", label: m.integrations_smtp_password, type: "password" },
      {
        key: "from_address",
        label: m.integrations_smtp_from_address,
        placeholder: m.integrations_smtp_from_address_placeholder,
      },
      {
        key: "from_name",
        label: m.integrations_smtp_from_name,
        placeholder: m.integrations_smtp_from_name_placeholder,
      },
    ],
    aws_sms: [
      { key: "access_key_id", label: m.integrations_aws_access_key_id },
      { key: "secret_access_key", label: m.integrations_aws_secret_access_key, type: "password" },
      {
        key: "region",
        label: m.integrations_aws_region,
        placeholder: m.integrations_aws_region_placeholder,
      },
      {
        key: "origination_identity",
        label: m.integrations_aws_origination_identity,
        placeholder: m.integrations_aws_origination_identity_placeholder,
      },
      { key: "configuration_set", label: m.integrations_aws_configuration_set },
    ],
  };
  const selectedNotificationFields = $derived(
    notificationFields[notificationsPanel.selectedProvider] ?? []
  );

  let testRecipient = $state("");
  let isSendingTest = $state(false);
  let testStatus = $state<{ ok: boolean; msg: string } | null>(null);

  // Sends with the saved settings, not the ones being edited, so it checks what reminders use.
  async function sendTestMessage() {
    const provider = notificationsPanel.selectedProvider;
    if (!canEdit || !provider || !testRecipient.trim()) return;
    isSendingTest = true;
    testStatus = null;
    // Staff can switch providers while a send is in flight; its result belongs only to the
    // provider it was sent with.
    const publish = (status: { ok: boolean; msg: string }) => {
      if (notificationsPanel.selectedProvider === provider) testStatus = status;
    };
    try {
      await NotificationService.SendTestMessage(
        auth.token,
        provider,
        testRecipient,
        m.integrations_test_subject(),
        m.integrations_test_body()
      );
      publish({ ok: true, msg: m.integrations_test_success() });
    } catch (e) {
      console.error("Failed to send test message:", e);
      publish({ ok: false, msg: handleError(e, m.integrations_test_error()) });
    } finally {
      isSendingTest = false;
    }
  }

  // Also locked while saving: the save payload is captured when it starts, so a toggle
  // mid-save would show a mode that was never persisted.
  const claimsTestModeDisabled = $derived(
    !canEdit ||
      !claimsPanel.selectedProvider ||
      claimsPanel.isLoadingConfig ||
      claimsPanel.isSavingConfig
  );

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
                placeholder={claimsPanel.secretSaved("api_key")
                  ? m.integrations_secret_saved_placeholder()
                  : m.integrations_placeholder_api_key()}
                disabled={!canEdit ||
                  !claimsPanel.selectedProvider ||
                  claimsPanel.isLoadingConfig ||
                  claimsPanel.isSavingConfig}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              />
            </div>
          </div>

          <label
            class="flex items-start gap-2.5 text-sm text-slate-200 select-none {claimsTestModeDisabled
              ? 'opacity-50'
              : 'cursor-pointer'}"
          >
            <input
              type="checkbox"
              class="mt-0.5 disabled:cursor-not-allowed"
              bind:checked={claimsPanel.testMode}
              disabled={claimsTestModeDisabled}
            />
            <span>
              {m.integrations_claims_test_mode()}
              <span class="block text-xs text-slate-500"
                >{m.integrations_claims_test_mode_hint()}</span
              >
            </span>
          </label>

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
                onchange={() => {
                  testStatus = null;
                  notificationsPanel.loadProviderConfig();
                }}
                disabled={notificationsPanel.noProviders}
                class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
              >
                <option value="">{m.integrations_placeholder_provider()}</option>
                {#each notificationsPanel.providers as p}
                  <option value={p}>{p}</option>
                {/each}
              </select>
            </div>
          </div>

          {#if selectedNotificationFields.length > 0}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              {#each selectedNotificationFields as field (field.key)}
                <div>
                  <label
                    for="notification-field-{field.key}"
                    class="block text-xs text-slate-400 mb-1">{field.label()}</label
                  >
                  {#if field.options}
                    <select
                      id="notification-field-{field.key}"
                      value={notificationsPanel.fieldValue(field.key) || field.options[0].value}
                      onchange={(e) =>
                        notificationsPanel.setFieldValue(field.key, e.currentTarget.value)}
                      disabled={!canEdit ||
                        notificationsPanel.isLoadingConfig ||
                        notificationsPanel.isSavingConfig}
                      class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
                    >
                      {#each field.options as option}
                        <option value={option.value}>{option.label()}</option>
                      {/each}
                    </select>
                  {:else}
                    <input
                      type={field.type ?? "text"}
                      id="notification-field-{field.key}"
                      value={notificationsPanel.fieldValue(field.key)}
                      oninput={(e) =>
                        notificationsPanel.setFieldValue(field.key, e.currentTarget.value)}
                      placeholder={notificationsPanel.secretSaved(field.key)
                        ? m.integrations_secret_saved_placeholder()
                        : field.placeholder?.()}
                      autocomplete="off"
                      disabled={!canEdit ||
                        notificationsPanel.isLoadingConfig ||
                        notificationsPanel.isSavingConfig}
                      class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
                    />
                  {/if}
                </div>
              {/each}
            </div>
          {/if}

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

          {#if notificationsPanel.selectedProvider}
            <div class="space-y-2 border-t border-slate-800 pt-3">
              <label for="notification-test-recipient" class="block text-xs text-slate-400"
                >{m.integrations_test_recipient()}</label
              >
              <p class="text-xs text-slate-500">{m.integrations_test_hint()}</p>
              <div class="flex flex-col gap-2 md:flex-row md:items-center">
                <input
                  type="text"
                  id="notification-test-recipient"
                  bind:value={testRecipient}
                  placeholder={m.integrations_test_recipient_placeholder()}
                  disabled={!canEdit || isSendingTest}
                  class="w-full md:flex-1 rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
                />
                <button
                  type="button"
                  class="btn btn-secondary btn-sm bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
                  disabled={!canEdit || isSendingTest || !testRecipient.trim()}
                  onclick={sendTestMessage}
                >
                  {isSendingTest ? m.integrations_test_sending() : m.integrations_test_send()}
                </button>
              </div>
              {#if testStatus}
                <p
                  class="text-xs font-semibold {testStatus.ok
                    ? 'text-emerald-400'
                    : 'text-rose-400'}"
                  role={testStatus.ok ? "status" : "alert"}
                >
                  {testStatus.msg}
                </p>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
