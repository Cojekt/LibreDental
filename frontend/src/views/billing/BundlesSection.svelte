<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService } from "@bindings/services/index.js";
  import type {
    TreatmentBundle,
    BundleItemTemplate,
    CountryConfig,
  } from "@bindings/domain/index.js";
  import Modal from "../../components/ui/Modal.svelte";
  import ConfirmModal from "../../components/ui/ConfirmModal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

  let { countryMeta = null } = $props<{
    countryMeta?: CountryConfig | null;
  }>();

  let bundles = $state<TreatmentBundle[]>([]);
  let loadingBundles = $state(false);
  let showBundleModal = $state(false);
  let isEditingBundle = $state(false);
  let editingBundleId = $state("");
  let editingBundleCreatedAt = $state("");

  // Bundle form
  let bundleShortname = $state("");
  let bundleName = $state("");
  let bundleDescription = $state("");
  let bundleItems = $state<BundleItemTemplate[]>([]);
  let shortnameError = $state("");

  function fmt(n: number) {
    const curr = countryMeta?.default_currency || "USD";
    try {
      return new Intl.NumberFormat("en-US", { style: "currency", currency: curr }).format(n / 100);
    } catch {
      return `${(n / 100).toFixed(2)}`;
    }
  }

  let requestGenBundles = 0;
  export async function loadBundles() {
    const gen = ++requestGenBundles;
    loadingBundles = true;
    try {
      const res = await BillingService.ListBundles();
      if (gen === requestGenBundles) {
        bundles = (res?.filter(Boolean) as TreatmentBundle[]) || [];
      }
    } catch (e) {
      if (gen === requestGenBundles) {
        console.error("Failed to load bundles:", e);
      }
    } finally {
      if (gen === requestGenBundles) {
        loadingBundles = false;
      }
    }
  }

  export function openNewBundle() {
    isEditingBundle = false;
    editingBundleId = "";
    editingBundleCreatedAt = "";
    bundleShortname = "";
    bundleName = "";
    bundleDescription = "";
    bundleItems = [];
    shortnameError = "";
    showBundleModal = true;
  }

  function openEditBundle(b: TreatmentBundle) {
    isEditingBundle = true;
    editingBundleId = b.id;
    editingBundleCreatedAt = b.created_at || "";
    bundleShortname = b.shortname;
    bundleName = b.name;
    bundleDescription = b.description ?? "";
    bundleItems = (b.items ?? []).map((i) => ({
      ...i,
      default_fee: (i.default_fee || 0) / 100,
    }));
    shortnameError = "";
    showBundleModal = true;
  }

  function addBundleItem() {
    bundleItems = [...bundleItems, { ada_code: "", description: "", default_fee: 0 }];
  }

  function removeBundleItem(idx: number) {
    bundleItems = bundleItems.filter((_, i: number) => i !== idx);
  }

  function bundleTotalFee() {
    return bundleItems.reduce((s: number, i: BundleItemTemplate) => s + (i.default_fee || 0), 0);
  }

  async function saveBundle(e: Event) {
    e.preventDefault();
    shortnameError = "";
    const sn = bundleShortname.trim().toLowerCase();
    if (!sn || !bundleName.trim()) return;

    const convertedItems = bundleItems.map((item) => ({
      ...item,
      default_fee: Math.round((item.default_fee || 0) * 100),
    }));

    const payload: TreatmentBundle = {
      id: isEditingBundle ? editingBundleId : `bundle_${Date.now()}`,
      shortname: sn,
      name: bundleName.trim(),
      description: bundleDescription,
      items: convertedItems,
      total_fee: Math.round(bundleTotalFee() * 100),
      created_at:
        isEditingBundle && editingBundleCreatedAt
          ? editingBundleCreatedAt
          : new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    try {
      if (isEditingBundle) {
        await BillingService.UpdateBundle(payload);
      } else {
        await BillingService.CreateBundle(payload);
      }
      showBundleModal = false;
      await loadBundles();
    } catch (e: any) {
      const msg = String(e);
      if (msg.includes("UNIQUE") || msg.includes("unique")) {
        shortnameError = `Shortname "${sn}" is already taken.`;
      } else {
        console.error("Failed to save bundle:", e);
      }
    }
  }

  let showConfirmDelete = $state(false);
  let bundleToDelete = $state<string | null>(null);

  function promptDelete(id: string) {
    bundleToDelete = id;
    showConfirmDelete = true;
  }

  async function executeDelete() {
    if (!bundleToDelete) return;
    try {
      await BillingService.DeleteBundle(bundleToDelete);
      await loadBundles();
    } catch (e) {
      console.error("Failed to delete bundle:", e);
    } finally {
      bundleToDelete = null;
    }
  }

  onMount(async () => {
    await loadBundles();
  });
</script>

<div class="space-y-4">
  <div class="flex justify-end">
    <button
      type="button"
      class="btn btn-primary text-xs flex items-center gap-1.5"
      onclick={openNewBundle}
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {m.billing_bundle_btn_create()}
    </button>
  </div>

  {#if loadingBundles}
    <div class="p-8 text-center text-sm text-slate-400">{m.common_loading()}</div>
  {:else if bundles.length === 0}
    <EmptyState
      title={m.billing_no_bundles()}
      subtitle="Create procedure bundle templates (e.g. Crown + Build-up) to speed up claim entry."
      icon="📦"
    />
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each bundles as b (b.id)}
        <div
          class="rounded-xl border border-slate-800 bg-slate-900/80 hover:border-slate-700 flex flex-col justify-between overflow-hidden transition-colors"
        >
          <div class="p-4 space-y-2 border-b border-slate-800">
            <div class="flex items-center gap-2">
              <span
                class="px-2 py-0.5 text-xs font-mono font-bold rounded-md bg-sky-500/10 text-sky-400 border border-sky-500/20"
              >
                {b.shortname}
              </span>
              <h4 class="text-sm font-bold text-slate-100 m-0 truncate">{b.name}</h4>
            </div>
            {#if b.description}
              <p class="text-xs text-slate-400 m-0 line-clamp-2">{b.description}</p>
            {/if}
          </div>

          <div class="p-4 space-y-2 flex-1">
            {#each b.items as item}
              <div class="flex items-center justify-between text-xs gap-2">
                <div class="flex items-center gap-2 min-w-0">
                  <span
                    class="px-1.5 py-0.5 text-[11px] font-mono font-bold rounded bg-slate-800 text-sky-300 border border-slate-700 shrink-0"
                  >
                    {item.ada_code}
                  </span>
                  <span class="text-slate-300 truncate">{item.description}</span>
                </div>
                <span class="font-mono font-semibold text-slate-100 shrink-0"
                  >{fmt(item.default_fee)}</span
                >
              </div>
            {/each}
          </div>

          <div
            class="flex items-center justify-between px-4 py-3 bg-slate-950/60 border-t border-slate-800 text-xs"
          >
            <span class="text-slate-400"
              >Total: <strong class="text-slate-100 font-mono text-sm">{fmt(b.total_fee)}</strong
              ></span
            >
            <div class="flex items-center gap-1">
              <button
                class="p-1.5 text-slate-400 hover:text-sky-300 rounded-lg hover:bg-slate-800 transition-colors"
                onclick={() => openEditBundle(b)}
                title={m.patients_btn_edit()}
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  class="h-4 w-4"
                >
                  <path
                    d="M11 5H6a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2v-5m-1.414-9.414a2 2 0 1 1 2.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                  />
                </svg>
              </button>

              <button
                class="p-1.5 text-slate-400 hover:text-rose-400 rounded-lg hover:bg-slate-800 transition-colors"
                onclick={() => promptDelete(b.id)}
                title={m.patient_archive()}
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  class="h-4 w-4"
                >
                  <path
                    d="M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- BUNDLE MODAL -->
<Modal
  bind:showModal={showBundleModal}
  title={isEditingBundle ? "Edit Bundle" : m.billing_bundle_btn_create()}
  subtitle="Configure multi-code procedure templates for single-click claim entry"
  icon="📦"
  maxWidth="max-w-3xl"
>
  <form onsubmit={saveBundle} class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <FormField
        label="Shortname"
        forId="b-shortname"
        helpText="Used for fast search lookup (e.g. 'crwn', 'rct-a')"
        required
        error={shortnameError}
      >
        <Input
          id="b-shortname"
          type="text"
          bind:value={bundleShortname}
          placeholder="e.g. crwn"
          required
        />
      </FormField>
      <FormField label="Full Name" forId="b-name" required>
        <Input
          id="b-name"
          type="text"
          bind:value={bundleName}
          placeholder={m.billing_bundle_name_placeholder()}
          required
        />
      </FormField>
    </div>

    <FormField label={m.charting_th_desc()} forId="b-desc">
      <Input
        id="b-desc"
        type="text"
        bind:value={bundleDescription}
        placeholder={m.billing_bundle_desc_placeholder()}
      />
    </FormField>

    <div class="rounded-xl border border-slate-800 bg-slate-950 p-4 space-y-3">
      <div class="flex items-center justify-between">
        <h4 class="text-xs font-bold uppercase tracking-wider text-slate-400 m-0">
          {m.billing_th_procedures()}
        </h4>
        <button type="button" class="btn btn-secondary btn-sm" onclick={addBundleItem}>
          + Add Item
        </button>
      </div>

      {#if bundleItems.length > 0}
        <div class="space-y-2">
          <div
            class="grid grid-cols-12 gap-2 text-[10px] font-bold uppercase tracking-wider text-slate-500 px-1"
          >
            <span class="col-span-3">{m.charting_th_code()}</span>
            <span class="col-span-6">{m.charting_th_desc()}</span>
            <span class="col-span-2 text-right">{m.billing_bundle_default_fee()}</span>
            <span class="col-span-1 text-center"></span>
          </div>
          {#each bundleItems as item, i}
            <div class="grid grid-cols-12 gap-2 items-center">
              <div class="col-span-3">
                <Input
                  bind:value={item.ada_code}
                  placeholder="D0120"
                  class="font-mono text-xs py-1.5 px-2"
                />
              </div>
              <div class="col-span-6">
                <Input
                  bind:value={item.description}
                  placeholder={m.charting_th_desc()}
                  class="text-xs py-1.5 px-2"
                />
              </div>
              <div class="col-span-2">
                <Input
                  type="number"
                  bind:value={item.default_fee}
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  class="text-right text-xs py-1.5 px-2"
                />
              </div>
              <div class="col-span-1 flex justify-center">
                <button
                  type="button"
                  class="p-1 text-rose-400 hover:bg-rose-500/10 rounded transition-colors"
                  onclick={() => removeBundleItem(i)}
                  title="Remove item">✕</button
                >
              </div>
            </div>
          {/each}
          <div class="text-right text-xs text-slate-400 pt-2 border-t border-slate-800">
            Total Bundle Fee: <strong class="text-white text-sm font-mono"
              >{fmt(Math.round(bundleTotalFee() * 100))}</strong
            >
          </div>
        </div>
      {:else}
        <div
          class="p-6 text-center text-xs text-slate-500 bg-slate-900/50 rounded-xl border border-dashed border-slate-800"
        >
          No CDT procedure items yet. Click '+ Add Item' above.
        </div>
      {/if}
    </div>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
        onclick={() => (showBundleModal = false)}
      >
        {m.common_cancel()}
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        {isEditingBundle ? m.patient_save_changes() : m.billing_bundle_btn_save()}
      </button>
    </div>
  </form>
</Modal>

<ConfirmModal
  bind:showModal={showConfirmDelete}
  title={m.billing_bundle_delete_title()}
  message={m.billing_bundle_confirm_delete()}
  confirmText={m.billing_bundle_delete_confirm()}
  onConfirm={executeDelete}
/>
