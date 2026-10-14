<script lang="ts">
  import { onMount } from "svelte";
  import { Dialogs } from "@wailsio/runtime";
  import { BridgeService } from "@bindings/services/index.js";
  import { BridgeCapability, type BridgeInfo } from "@bindings/domain/models.js";
  import { m } from "../../paraglide/messages.js";
  import { auth } from "../../stores/auth.svelte.js";
  import { handleError } from "$lib/error.js";

  let { canEdit = false } = $props<{ canEdit: boolean }>();

  // Names must match DefaultProgramBridges in bridge_command_line.go; unknown names show as-is.
  const bridgeLabels: Record<string, () => string> = {
    custom: m.integrations_bridge_name_custom,
    weasis: m.integrations_bridge_name_weasis,
    radiant: m.integrations_bridge_name_radiant,
    microdicom: m.integrations_bridge_name_microdicom,
  };
  const capabilityLabels: Record<string, () => string> = {
    [BridgeCapability.BridgeCapabilityPatient]: m.integrations_bridge_capability_patient,
    [BridgeCapability.BridgeCapabilityDocuments]: m.integrations_bridge_capability_documents,
  };

  function bridgeLabel(name: string): string {
    return bridgeLabels[name]?.() ?? name;
  }

  let available = $state(true);
  let bridges = $state<BridgeInfo[]>([]);
  let tokens = $state<string[]>([]);
  let loaded = $state(false);
  let loadError = $state(false);
  let selectedName = $state("");
  let enabled = $state(false);
  let path = $state("");
  let args = $state("");
  let isSaving = $state(false);
  let saveStatus = $state<{ ok: boolean; msg: string } | null>(null);

  const selected = $derived(bridges.find((b) => b.config.name === selectedName) ?? null);
  const fieldsDisabled = $derived(!canEdit || !selected || isSaving);

  async function loadBridges() {
    loadError = false;
    try {
      available = await BridgeService.IsAvailable();
      if (!available) return;
      const [list, tokenList] = await Promise.all([
        BridgeService.ListBridges(auth.token),
        BridgeService.ListArgumentTokens(),
      ]);
      bridges = list || [];
      tokens = tokenList || [];
    } catch (e) {
      console.error("Failed to load program bridges:", e);
      loadError = true;
    } finally {
      loaded = true;
    }
  }

  function selectBridge() {
    saveStatus = null;
    enabled = selected?.config.enabled ?? false;
    path = selected?.config.path ?? "";
    args = selected?.config.args ?? "";
  }

  async function browseForProgram() {
    try {
      const picked = await Dialogs.OpenFile({
        Title: m.integrations_bridge_browse_title(),
        CanChooseFiles: true,
        CanChooseDirectories: false,
      });
      if (picked) path = picked;
    } catch (e) {
      console.error("Failed to pick program:", e);
    }
  }

  async function saveBridge() {
    if (!canEdit || !selected) return;
    isSaving = true;
    saveStatus = null;
    const reqName = selected.config.name;
    const config = { ...selected.config, enabled, path: path.trim(), args };
    try {
      await BridgeService.SetBridgeConfig(auth.token, config);
      bridges = bridges.map((b) => (b.config.name === reqName ? { ...b, config } : b));
      if (reqName === selectedName) {
        path = config.path;
        saveStatus = { ok: true, msg: m.integrations_bridge_save_success() };
      }
    } catch (e) {
      console.error("Failed to save program bridge:", e);
      if (reqName === selectedName) {
        saveStatus = { ok: false, msg: handleError(e, m.integrations_save_error()) };
      }
    } finally {
      isSaving = false;
    }
  }

  onMount(loadBridges);
</script>

<div class="space-y-3 rounded-xl border border-slate-800 bg-slate-950/80 p-4">
  <p class="text-xs text-slate-500">{m.integrations_bridges_hint()}</p>

  {#if !available}
    <p class="text-xs text-amber-400">{m.integrations_bridges_unavailable()}</p>
  {:else if loadError}
    <p class="text-xs text-red-400">{m.integrations_providers_load_error()}</p>
  {:else if loaded && bridges.length === 0}
    <p class="text-xs text-slate-500">{m.integrations_bridges_none()}</p>
  {/if}

  {#if available}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="bridge-select" class="block text-xs text-slate-400 mb-1"
          >{m.integrations_bridge_label_program()}</label
        >
        <select
          id="bridge-select"
          bind:value={selectedName}
          onchange={selectBridge}
          disabled={bridges.length === 0 || isSaving}
          class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
        >
          <option value="">{m.integrations_bridge_placeholder_program()}</option>
          {#each bridges as b (b.config.name)}
            <option value={b.config.name}>{bridgeLabel(b.config.name)}</option>
          {/each}
        </select>
      </div>

      {#if selected}
        <div>
          <span class="block text-xs text-slate-400 mb-1"
            >{m.integrations_bridge_label_capabilities()}</span
          >
          <div class="flex flex-wrap gap-1.5 pt-1.5">
            {#each selected.capabilities as cap}
              <span
                class="rounded-md border border-slate-700 bg-slate-900 px-2 py-0.5 text-xs text-slate-300"
                >{capabilityLabels[cap]?.() ?? cap}</span
              >
            {/each}
          </div>
        </div>
      {/if}
    </div>

    {#if selected}
      <label
        class="flex items-start gap-2.5 text-sm text-slate-200 select-none {fieldsDisabled
          ? 'opacity-50'
          : 'cursor-pointer'}"
      >
        <input
          type="checkbox"
          class="mt-0.5 disabled:cursor-not-allowed"
          bind:checked={enabled}
          disabled={fieldsDisabled}
        />
        <span>
          {m.integrations_bridge_enabled()}
          <span class="block text-xs text-slate-500">{m.integrations_bridge_enabled_hint()}</span>
        </span>
      </label>

      <div>
        <label for="bridge-path" class="block text-xs text-slate-400 mb-1"
          >{m.integrations_bridge_label_path()}</label
        >
        <div class="flex items-center gap-2">
          <input
            type="text"
            id="bridge-path"
            bind:value={path}
            placeholder={m.integrations_bridge_placeholder_path()}
            autocomplete="off"
            spellcheck="false"
            disabled={fieldsDisabled}
            class="min-w-0 flex-1 rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 font-mono text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
          />
          <button
            type="button"
            class="btn btn-secondary btn-sm shrink-0 bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
            disabled={fieldsDisabled}
            onclick={browseForProgram}>{m.integrations_bridge_browse()}</button
          >
        </div>
      </div>

      <div>
        <label for="bridge-args" class="block text-xs text-slate-400 mb-1"
          >{m.integrations_bridge_label_args()}</label
        >
        <input
          type="text"
          id="bridge-args"
          bind:value={args}
          autocomplete="off"
          spellcheck="false"
          disabled={fieldsDisabled}
          class="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 font-mono text-sm text-slate-100 focus:border-sky-500 focus:outline-none disabled:opacity-50"
        />
        <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1">
          {#if tokens.length > 0}
            <span class="text-xs text-slate-500"
              >{m.integrations_bridge_tokens_hint({ tokens: tokens.join(" ") })}</span
            >
          {/if}
          {#if selected.default_args && args !== selected.default_args}
            <button
              type="button"
              class="text-xs text-slate-400 underline cursor-pointer disabled:opacity-50"
              disabled={fieldsDisabled}
              onclick={() => (args = selected.default_args)}
              >{m.integrations_bridge_restore_default()}</button
            >
          {/if}
        </div>
      </div>
    {/if}

    <div class="flex items-center justify-end gap-3">
      {#if saveStatus}
        <span
          class="text-xs font-semibold {saveStatus.ok ? 'text-emerald-400' : 'text-rose-400'}"
          role={saveStatus.ok ? "status" : "alert"}>{saveStatus.msg}</span
        >
      {/if}
      <button
        type="button"
        class="btn btn-secondary btn-sm bg-slate-800 text-white border-slate-700 hover:bg-slate-700 px-4 py-1 rounded-md text-xs cursor-pointer"
        disabled={fieldsDisabled}
        onclick={saveBridge}
      >
        {isSaving ? m.integrations_btn_saving() : m.integrations_bridge_btn_save()}
      </button>
    </div>
  {/if}
</div>
