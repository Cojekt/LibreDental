<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService } from "@bindings/services/index.js";
  import type {
    TreatmentBundle,
    BundleItemTemplate,
    CountryConfig,
  } from "@bindings/domain/index.js";
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

  // Bundle form
  let bundleShortname = $state("");
  let bundleName = $state("");
  let bundleDescription = $state("");
  let bundleItems = $state<BundleItemTemplate[]>([]);
  let shortnameError = $state("");

  function fmt(n: number) {
    const curr = countryMeta?.default_currency || "USD";
    try {
      return new Intl.NumberFormat("en-US", { style: "currency", currency: curr }).format(n);
    } catch {
      return `${n.toFixed(2)}`;
    }
  }

  export async function loadBundles() {
    loadingBundles = true;
    try {
      const res = await BillingService.ListBundles();
      bundles = (res?.filter(Boolean) as TreatmentBundle[]) || [];
    } catch (e) {
      console.error("Failed to load bundles:", e);
    } finally {
      loadingBundles = false;
    }
  }

  export function openNewBundle() {
    isEditingBundle = false;
    editingBundleId = "";
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
    bundleShortname = b.shortname;
    bundleName = b.name;
    bundleDescription = b.description ?? "";
    bundleItems = (b.items ?? []).map((i) => ({ ...i }));
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

    const payload: TreatmentBundle = {
      id: isEditingBundle ? editingBundleId : `bundle_${Date.now()}`,
      shortname: sn,
      name: bundleName.trim(),
      description: bundleDescription,
      items: bundleItems,
      total_fee: bundleTotalFee(),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    try {
      if (isEditingBundle) {
        await BillingService.UpdateBundle(payload as any);
      } else {
        await BillingService.CreateBundle(payload as any);
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

  async function deleteBundle(id: string) {
    if (!confirm("Delete this procedure bundle?")) return;
    try {
      await BillingService.DeleteBundle(id);
      await loadBundles();
    } catch (e) {
      console.error("Failed to delete bundle:", e);
    }
  }

  onMount(async () => {
    await loadBundles();
  });
</script>

<div class="space-y-4">
  <div class="flex justify-end mb-2">
    <button type="button" class="btn btn-primary" onclick={openNewBundle}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {m.billing_btn_new_bundle()}
    </button>
  </div>

  {#if loadingBundles}
    <div class="billing-loading">Loading bundles…</div>
  {:else if bundles.length === 0}
    <EmptyState
      title="No procedure bundles yet"
      subtitle="Create one to speed up claim entry."
      icon="M19 11H5m14 0a2 2 0 0 1 2 2v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6a2 2 0 0 1 2-2m14 0V9a2 2 0 0 1-2-2M5 11V9a2 2 0 0 1 2-2m0 0V5a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2M7 7h10"
    />
  {:else}
    <div class="bundle-grid">
      {#each bundles as b (b.id)}
        <div class="bundle-card">
          <div class="bundle-card-header">
            <div class="bundle-card-title-row">
              <span class="shortname-badge">{b.shortname}</span>
              <span class="bundle-name">{b.name}</span>
            </div>
            {#if b.description}
              <p class="bundle-description">{b.description}</p>
            {/if}
          </div>

          <div class="bundle-items-list">
            {#each b.items as item}
              <div class="bundle-item-row">
                <span class="ada-badge">{item.ada_code}</span>
                <span class="bundle-item-desc">{item.description}</span>
                <span class="bundle-item-fee">{fmt(item.default_fee)}</span>
              </div>
            {/each}
          </div>

          <div class="bundle-card-footer">
            <span class="bundle-total">Total: <strong>{fmt(b.total_fee)}</strong></span>
            <div class="bundle-card-actions">
              <button class="action-btn" onclick={() => openEditBundle(b)} title="Edit">
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
                class="action-btn action-btn-danger"
                onclick={() => deleteBundle(b.id)}
                title="Delete"
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
{#if showBundleModal}
  <div class="modal-backdrop" role="dialog" aria-modal="true">
    <div class="modal-box modal-wide">
      <div class="modal-header">
        <h2 class="modal-title">{isEditingBundle ? "Edit Bundle" : "New Procedure Bundle"}</h2>
        <button class="modal-close" onclick={() => (showBundleModal = false)}>✕</button>
      </div>

      <form onsubmit={saveBundle} class="modal-body">
        <div class="form-grid-2">
          <div class="form-field">
            <label class="form-label" for="b-shortname">
              Shortname *
              <span class="form-label-hint">e.g. "crwn", "rct-a" — used for fast lookup</span>
            </label>
            <input
              id="b-shortname"
              type="text"
              bind:value={bundleShortname}
              placeholder="crwn"
              required
              class={shortnameError ? "border-red-500 focus:border-red-500 focus:ring-red-500" : ""}
            />
            {#if shortnameError}
              <p class="field-error">{shortnameError}</p>
            {/if}
          </div>
          <div class="form-field">
            <label class="form-label" for="b-name">Full Name *</label>
            <input
              id="b-name"
              type="text"
              bind:value={bundleName}
              placeholder="Crown + Build-up"
              required
            />
          </div>
        </div>

        <div class="form-field">
          <label class="form-label" for="b-desc">Description</label>
          <input
            id="b-desc"
            type="text"
            bind:value={bundleDescription}
            placeholder="Optional description"
          />
        </div>

        <div class="line-items-section">
          <div class="line-items-header">
            <span class="form-label mb-0">Procedure Items</span>
            <button type="button" class="btn btn-secondary btn-sm" onclick={addBundleItem}
              >+ Add Item</button
            >
          </div>

          {#if bundleItems.length > 0}
            <div class="bundle-items-grid-header">
              <span>ADA Code</span><span>Description</span><span>Default Fee</span><span></span>
            </div>
            {#each bundleItems as item, i}
              <div class="bundle-item-edit-row">
                <input type="text" bind:value={item.ada_code} placeholder="D0120" />
                <input
                  type="text"
                  bind:value={item.description}
                  placeholder="Procedure description"
                />
                <input
                  type="number"
                  bind:value={item.default_fee}
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                />
                <button
                  type="button"
                  class="action-btn action-btn-danger"
                  onclick={() => removeBundleItem(i)}>✕</button
                >
              </div>
            {/each}
            <div class="line-items-total">
              Total: <strong>{fmt(bundleTotalFee())}</strong>
            </div>
          {:else}
            <div class="line-items-empty">No items yet. Add CDT-coded procedures above.</div>
          {/if}
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" onclick={() => (showBundleModal = false)}
            >Cancel</button
          >
          <button type="submit" class="btn btn-primary">
            {isEditingBundle ? "Save Changes" : "Create Bundle"}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
