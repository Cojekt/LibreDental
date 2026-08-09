<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService, ChartService } from "@bindings/services/index.js";
  import type {
    Patient,
    Provider,
    Claim,
    ClaimLineItem,
    CountryConfig,
    ToothCondition,
  } from "@bindings/domain/index.js";
  import { ClaimStatus } from "@bindings/domain/index.js";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

  let {
    patients = [],
    providers = [],
    countryMeta = null,
  } = $props<{
    patients: Patient[];
    providers: Provider[];
    countryMeta?: CountryConfig | null;
  }>();

  let claims = $state<Claim[]>([]);
  let loadingClaims = $state(false);
  let showClaimModal = $state(false);
  let isEditingClaim = $state(false);
  let editingClaimId = $state("");

  // Claim form fields
  let claimPatientId = $state("");
  let claimProviderId = $state("");
  let claimDateOfService = $state(new Date().toISOString().split("T")[0]);
  let claimInsuranceCarrier = $state("");
  let claimPolicyNumber = $state("");
  let claimGroupNumber = $state("");
  let claimStatus = $state<ClaimStatus>(ClaimStatus.ClaimStatusDraft);
  let claimNotes = $state("");
  let claimLineItems = $state<ClaimLineItem[]>([]);

  // Bundle shortname lookup for claim entry
  let bundleLookupInput = $state("");
  let bundleLookupError = $state("");
  let bundleLookupLoading = $state(false);

  // Claim filter
  let claimFilterPatient = $state("");

  // Chart Conditions Import Modal State for Claim Entry
  let showChartImportModal = $state(false);
  let chartImportConditions = $state<ToothCondition[]>([]);
  let selectedImportConditionIds = $state<string[]>([]);
  let loadingChartImport = $state(false);

  const CLAIM_STATUSES = ["draft", "submitted", "accepted", "rejected", "paid"];

  function patientName(id: string) {
    const p = patients.find((p: Patient) => p.id === id);
    return p ? `${p.first_name} ${p.last_name}` : id;
  }

  function providerName(id: string) {
    const p = providers.find((p: Provider) => p.id === id);
    return p ? p.name : id;
  }

  function fmt(n: number) {
    const curr = countryMeta?.default_currency || "USD";
    try {
      return new Intl.NumberFormat("en-US", { style: "currency", currency: curr }).format(n);
    } catch {
      return `${n.toFixed(2)}`;
    }
  }

  function claimTotal(c: Claim) {
    return (c.line_items ?? []).reduce((s, i) => s + i.fee, 0);
  }

  export async function loadClaims() {
    loadingClaims = true;
    try {
      const res = await BillingService.ListClaims(claimFilterPatient);
      claims = (res?.filter(Boolean) as Claim[]) || [];
    } catch (e) {
      console.error("Failed to load claims:", e);
    } finally {
      loadingClaims = false;
    }
  }

  export function openNewClaim() {
    isEditingClaim = false;
    editingClaimId = "";
    claimPatientId = patients[0]?.id ?? "";
    claimProviderId = providers[0]?.id ?? "";
    claimDateOfService = new Date().toISOString().split("T")[0];
    claimInsuranceCarrier = "";
    claimPolicyNumber = "";
    claimGroupNumber = "";
    claimStatus = ClaimStatus.ClaimStatusDraft;
    claimNotes = "";
    claimLineItems = [];
    bundleLookupInput = "";
    bundleLookupError = "";
    showClaimModal = true;
  }

  async function openEditClaim(c: Claim) {
    isEditingClaim = true;
    editingClaimId = c.id;
    claimPatientId = c.patient_id;
    claimProviderId = c.provider_id;
    claimDateOfService = c.date_of_service;
    claimInsuranceCarrier = c.insurance_carrier ?? "";
    claimPolicyNumber = c.policy_number ?? "";
    claimGroupNumber = c.group_number ?? "";
    claimStatus = c.status;
    claimNotes = c.notes ?? "";
    claimLineItems = (c.line_items ?? []).map((li) => ({ ...li }));
    bundleLookupInput = "";
    bundleLookupError = "";
    showClaimModal = true;
  }

  function addLineItem() {
    claimLineItems = [
      ...claimLineItems,
      { id: `li_${Date.now()}`, ada_code: "", description: "", fee: 0 },
    ];
  }

  function removeLineItem(idx: number) {
    claimLineItems = claimLineItems.filter((_, i) => i !== idx);
  }

  async function applyBundleLookup() {
    const sn = bundleLookupInput.trim().toLowerCase();
    if (!sn) return;
    bundleLookupLoading = true;
    bundleLookupError = "";
    try {
      const b = await BillingService.GetBundleByShortname(sn);
      if (b) {
        const newItems: ClaimLineItem[] = (b.items ?? []).map((item, i) => ({
          id: `li_${Date.now()}_${i}`,
          ada_code: item.ada_code,
          description: item.description,
          fee: item.default_fee,
        }));
        claimLineItems = [...claimLineItems, ...newItems];
        bundleLookupInput = "";
      } else {
        bundleLookupError = `No bundle found for shortname "${sn}"`;
      }
    } catch {
      bundleLookupError = `No bundle found for shortname "${sn}"`;
    } finally {
      bundleLookupLoading = false;
    }
  }

  async function saveClaim(e: Event) {
    e.preventDefault();
    if (!claimPatientId || !claimDateOfService) return;

    const payload: Claim = {
      id: isEditingClaim ? editingClaimId : `claim_${Date.now()}`,
      patient_id: claimPatientId,
      provider_id: claimProviderId,
      date_of_service: claimDateOfService,
      insurance_carrier: claimInsuranceCarrier,
      policy_number: claimPolicyNumber,
      group_number: claimGroupNumber,
      status: claimStatus,
      notes: claimNotes,
      line_items: claimLineItems,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    try {
      if (isEditingClaim) {
        await BillingService.UpdateClaim(payload as any);
      } else {
        await BillingService.CreateClaim(payload as any);
      }
      showClaimModal = false;
      await loadClaims();
    } catch (e) {
      console.error("Failed to save claim:", e);
    }
  }

  async function deleteClaim(id: string) {
    if (!confirm("Delete this claim? This cannot be undone.")) return;
    try {
      await BillingService.DeleteClaim(id);
      await loadClaims();
    } catch (e) {
      console.error("Failed to delete claim:", e);
    }
  }

  async function openChartImportModal() {
    if (!claimPatientId) {
      alert("Please select a patient first.");
      return;
    }
    loadingChartImport = true;
    selectedImportConditionIds = [];
    try {
      const chart = await ChartService.GetPatientChart(claimPatientId);
      chartImportConditions = (chart?.conditions || []).filter(
        (c) => c.status === "treatment_planned" || c.status === "completed"
      );
      if (chartImportConditions.length === 0) {
        alert("No treatment planned or completed tooth conditions found for this patient.");
        return;
      }
      showChartImportModal = true;
    } catch (e) {
      console.error("Failed to load chart conditions:", e);
      alert("Failed to load patient chart conditions.");
    } finally {
      loadingChartImport = false;
    }
  }

  function applyChartImport() {
    const toImport = chartImportConditions.filter((c) => selectedImportConditionIds.includes(c.id));
    const newItems: ClaimLineItem[] = toImport.map((cond, i) => ({
      id: `li_${Date.now()}_${i}`,
      tooth_condition_id: cond.id,
      tooth_number: cond.tooth_number,
      surfaces: cond.surfaces,
      ada_code: cond.ada_code || "PROC",
      description: cond.description || `Tooth #${cond.tooth_number} procedure`,
      fee: cond.fee || 0,
    }));

    claimLineItems = [...claimLineItems, ...newItems];
    showChartImportModal = false;
  }

  onMount(async () => {
    await loadClaims();
  });
</script>

<div class="space-y-4">
  <div class="billing-toolbar flex items-center justify-between gap-4">
    <select bind:value={claimFilterPatient} onchange={loadClaims} class="billing-filter-select">
      <option value="">{m.billing_filter_all_patients()}</option>
      {#each patients as p}
        <option value={p.id}>{p.first_name} {p.last_name}</option>
      {/each}
    </select>

    <button type="button" class="btn btn-primary" onclick={openNewClaim}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {m.billing_btn_new_claim()}
    </button>
  </div>

  {#if loadingClaims}
    <div class="billing-loading">{m.common_loading()}</div>
  {:else if claims.length === 0}
    <EmptyState
      title={m.billing_no_claims()}
      icon="M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414A1 1 0 0 1 19 9.414V19a2 2 0 0 1-2 2z"
    />
  {:else}
    <div class="claims-table-wrap">
      <table class="claims-table">
        <thead>
          <tr>
            <th>{m.billing_th_date()}</th>
            <th>{m.billing_th_patient()}</th>
            <th>{m.billing_th_provider()}</th>
            <th>{m.billing_th_carrier()}</th>
            <th>{m.billing_th_procedures()}</th>
            <th>{m.billing_th_total()}</th>
            <th>{m.billing_th_status()}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each claims as c (c.id)}
            <tr class="claim-row">
              <td class="claim-date">{c.date_of_service}</td>
              <td>{patientName(c.patient_id)}</td>
              <td class="text-slate-400">{providerName(c.provider_id)}</td>
              <td class="text-slate-400">{c.insurance_carrier || "—"}</td>
              <td>
                <div class="line-items-preview">
                  {#each (c.line_items ?? []).slice(0, 2) as li}
                    <span class="ada-badge">{li.ada_code}</span>
                  {/each}
                  {#if (c.line_items ?? []).length > 2}
                    <span class="text-slate-500 text-xs">+{(c.line_items ?? []).length - 2}</span>
                  {/if}
                </div>
              </td>
              <td class="claim-fee">{fmt(claimTotal(c))}</td>
              <td>
                <StatusBadge variant={c.status} />
              </td>
              <td class="claim-actions">
                <button class="action-btn" onclick={() => openEditClaim(c)} title="Edit">
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
                  onclick={() => deleteClaim(c.id)}
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
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- CLAIM MODAL -->
{#if showClaimModal}
  <div class="modal-backdrop" role="dialog" aria-modal="true">
    <div class="modal-box modal-wide">
      <div class="modal-header">
        <h2 class="modal-title">{isEditingClaim ? "Edit Claim" : "New Insurance Claim"}</h2>
        <button class="modal-close" onclick={() => (showClaimModal = false)}>✕</button>
      </div>

      <form onsubmit={saveClaim} class="modal-body">
        <div class="form-grid-3">
          <div class="form-field">
            <label class="form-label" for="cl-patient">Patient *</label>
            <select id="cl-patient" bind:value={claimPatientId} required>
              {#each patients as p}
                <option value={p.id}>{p.first_name} {p.last_name}</option>
              {/each}
            </select>
          </div>
          <div class="form-field">
            <label class="form-label" for="cl-provider">Provider</label>
            <select id="cl-provider" bind:value={claimProviderId}>
              <option value="">— None —</option>
              {#each providers as pr}
                <option value={pr.id}>{pr.name}</option>
              {/each}
            </select>
          </div>
          <div class="form-field">
            <label class="form-label" for="cl-dos">Date of Service *</label>
            <input id="cl-dos" type="date" bind:value={claimDateOfService} required />
          </div>
        </div>

        <div class="form-grid-3">
          <div class="form-field">
            <label class="form-label" for="cl-carrier">Insurance Carrier</label>
            <input
              id="cl-carrier"
              type="text"
              bind:value={claimInsuranceCarrier}
              placeholder="e.g. Delta Dental"
            />
          </div>
          <div class="form-field">
            <label class="form-label" for="cl-policy">Policy #</label>
            <input
              id="cl-policy"
              type="text"
              bind:value={claimPolicyNumber}
              placeholder="Policy number"
            />
          </div>
          <div class="form-field">
            <label class="form-label" for="cl-group">Group #</label>
            <input
              id="cl-group"
              type="text"
              bind:value={claimGroupNumber}
              placeholder="Group number"
            />
          </div>
        </div>

        <div class="form-grid-2">
          <div class="form-field">
            <label class="form-label" for="cl-status">Status</label>
            <select id="cl-status" bind:value={claimStatus}>
              {#each CLAIM_STATUSES as s}
                <option value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>
              {/each}
            </select>
          </div>
          <div class="form-field">
            <label class="form-label" for="cl-notes">Notes</label>
            <input id="cl-notes" type="text" bind:value={claimNotes} placeholder="Optional notes" />
          </div>
        </div>

        <div class="line-items-section">
          <div class="line-items-header">
            <span class="form-label mb-0">Procedure Line Items</span>
            <div class="line-items-header-actions">
              <button
                type="button"
                class="btn btn-secondary btn-sm flex items-center gap-1 bg-emerald-950/60 border border-emerald-500/40 text-emerald-300 hover:bg-emerald-900/60"
                onclick={openChartImportModal}
              >
                🦷 Import Chart Procedures
              </button>
              <div class="bundle-lookup">
                <input
                  type="text"
                  class="bundle-lookup-input"
                  bind:value={bundleLookupInput}
                  placeholder="Bundle shortname (e.g. crwn)"
                  onkeydown={(e) => e.key === "Enter" && (e.preventDefault(), applyBundleLookup())}
                />
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  onclick={applyBundleLookup}
                  disabled={bundleLookupLoading}
                >
                  {bundleLookupLoading ? "…" : "Apply"}
                </button>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" onclick={addLineItem}>
                + Add Row
              </button>
            </div>
          </div>

          {#if bundleLookupError}
            <p class="lookup-error">{bundleLookupError}</p>
          {/if}

          {#if claimLineItems.length > 0}
            <div class="line-items-grid-header">
              <span>ADA Code</span><span>Description</span>
              <span>Tooth</span><span>Fee</span><span>Ins. Allowed</span><span>Pt. Portion</span
              ><span></span>
            </div>
            {#each claimLineItems as li, i}
              <div class="line-item-row">
                <input type="text" bind:value={li.ada_code} placeholder="D0120" class="li-ada" />
                <input
                  type="text"
                  bind:value={li.description}
                  placeholder="Description"
                  class="li-desc"
                />
                <input
                  type="number"
                  bind:value={li.tooth_number}
                  placeholder="—"
                  min="1"
                  max="32"
                  class="li-tooth"
                />
                <input
                  type="number"
                  bind:value={li.fee}
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  class="li-fee"
                />
                <input
                  type="number"
                  bind:value={li.insurance_allowed}
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  class="li-fee"
                />
                <input
                  type="number"
                  bind:value={li.patient_portion}
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  class="li-fee"
                />
                <button
                  type="button"
                  class="action-btn action-btn-danger"
                  onclick={() => removeLineItem(i)}>✕</button
                >
              </div>
            {/each}
            <div class="line-items-total">
              Total: <strong>{fmt(claimLineItems.reduce((s, li) => s + (li.fee || 0), 0))}</strong>
            </div>
          {:else}
            <div class="line-items-empty">
              No line items. Add rows manually or apply a bundle shortname above.
            </div>
          {/if}
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" onclick={() => (showClaimModal = false)}
            >Cancel</button
          >
          <button type="submit" class="btn btn-primary">
            {isEditingClaim ? "Save Changes" : "Create Claim"}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Import Chart Conditions Modal -->
{#if showChartImportModal}
  <div class="modal-backdrop" role="dialog" aria-modal="true">
    <div class="modal-box modal-wide max-w-2xl">
      <div class="modal-header">
        <h2 class="modal-title">{m.billing_chart_import_title()}</h2>
        <button class="modal-close" onclick={() => (showChartImportModal = false)}>✕</button>
      </div>

      <div class="modal-body flex flex-col gap-4">
        <p class="text-xs text-slate-300 m-0">
          {m.billing_chart_import_desc()}
        </p>

        {#if chartImportConditions.length === 0}
          <div
            class="p-6 text-center text-xs text-slate-400 bg-slate-950 border border-slate-800 rounded-xl"
          >
            {m.billing_chart_import_empty()}
          </div>
        {:else}
          <div
            class="max-h-60 overflow-y-auto border border-slate-800 rounded-xl divide-y divide-slate-800 bg-slate-950"
          >
            {#each chartImportConditions as cond}
              <label
                class="flex items-center justify-between p-3 hover:bg-slate-900 cursor-pointer text-xs"
              >
                <div class="flex items-center gap-3">
                  <input
                    type="checkbox"
                    value={cond.id}
                    checked={selectedImportConditionIds.includes(cond.id)}
                    onchange={(e) => {
                      const checked = (e.target as HTMLInputElement).checked;
                      if (checked) {
                        selectedImportConditionIds = [...selectedImportConditionIds, cond.id];
                      } else {
                        selectedImportConditionIds = selectedImportConditionIds.filter(
                          (id) => id !== cond.id
                        );
                      }
                    }}
                    class="rounded border-slate-700 text-sky-500 focus:ring-sky-500"
                  />
                  <div>
                    <div class="font-bold text-slate-200">
                      Tooth #{cond.tooth_number}
                      {cond.surfaces?.length ? `(${cond.surfaces.join(", ")})` : ""}
                      <span class="ml-2 font-mono text-sky-400">[{cond.ada_code || "PROC"}]</span>
                    </div>
                    <div class="text-slate-400">{cond.description}</div>
                  </div>
                </div>
                <div class="font-mono font-semibold text-slate-100">{fmt(cond.fee || 0)}</div>
              </label>
            {/each}
          </div>
        {/if}

        <div class="modal-footer">
          <button
            type="button"
            class="btn btn-secondary"
            onclick={() => (showChartImportModal = false)}
          >
            {m.common_cancel()}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            disabled={selectedImportConditionIds.length === 0}
            onclick={applyChartImport}
          >
            {m.billing_chart_import_submit({ count: selectedImportConditionIds.length })}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
