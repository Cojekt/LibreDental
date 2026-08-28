<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService, ChartService } from "@bindings/services/index.js";
  import { auth } from "../../stores/auth.svelte.js";
  import type {
    Patient,
    Provider,
    Claim,
    ClaimLineItem,
    CountryConfig,
    ToothCondition,
  } from "@bindings/domain/index.js";
  import { ClaimStatus } from "@bindings/domain/index.js";
  import { getTodayDateString } from "$lib/date.js";
  import Modal from "../../components/ui/Modal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
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
  let editingClaimCreatedAt = $state("");

  // Claim form fields
  let claimPatientId = $state("");
  let claimProviderId = $state("");
  let claimDateOfService = $state(getTodayDateString());
  let claimInsuranceCarrier = $state("");
  let claimPolicyNumber = $state("");
  let claimGroupNumber = $state("");
  let claimStatus = $state<any>(ClaimStatus.ClaimStatusDraft);
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

  let submittingClaims = $state<Record<string, boolean>>({});
  let integrationProviders = $state<string[]>([]);
  let hasConfiguredProvider = $state(false);

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
      return new Intl.NumberFormat("en-US", { style: "currency", currency: curr }).format(n / 100);
    } catch {
      return `${(n / 100).toFixed(2)}`;
    }
  }

  function claimTotal(c: Claim) {
    return (c.line_items ?? []).reduce((s, i) => s + i.fee, 0);
  }

  let requestGenClaims = 0;
  export async function loadClaims() {
    const gen = ++requestGenClaims;
    loadingClaims = true;
    try {
      const res = await BillingService.ListClaims(auth.token, claimFilterPatient);
      if (gen === requestGenClaims) {
        claims = (res?.filter(Boolean) as Claim[]) || [];
      }
    } catch (e) {
      if (gen === requestGenClaims) {
        console.error("Failed to load claims:", e);
      }
    } finally {
      if (gen === requestGenClaims) {
        loadingClaims = false;
      }
    }
  }

  export function openNewClaim() {
    isEditingClaim = false;
    editingClaimId = "";
    editingClaimCreatedAt = "";
    claimPatientId = patients[0]?.id ?? "";
    claimProviderId = providers[0]?.id ?? "";
    claimDateOfService = getTodayDateString();
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

  async function openEditClaim(id: string) {
    const c = claims.find((x) => x.id === id);
    if (!c) return;

    isEditingClaim = true;
    editingClaimId = c.id;
    editingClaimCreatedAt = c.created_at || "";
    claimPatientId = c.patient_id;
    claimProviderId = c.provider_id;
    claimDateOfService = c.date_of_service;
    claimInsuranceCarrier = c.insurance_carrier ?? "";
    claimPolicyNumber = c.policy_number ?? "";
    claimGroupNumber = c.group_number ?? "";
    claimStatus = c.status;
    claimNotes = c.notes ?? "";
    claimLineItems = (c.line_items ?? []).map((li) => ({
      ...li,
      fee: (li.fee || 0) / 100,
      insurance_allowed: li.insurance_allowed != null ? li.insurance_allowed / 100 : undefined,
      patient_portion: li.patient_portion != null ? li.patient_portion / 100 : undefined,
    }));
    bundleLookupInput = "";
    bundleLookupError = "";
    showClaimModal = true;
  }

  function addLineItem() {
    claimLineItems = [
      ...claimLineItems,
      { id: `li_${Date.now()}`, ada_code: "", description: "", fee: 0 } as any as ClaimLineItem,
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
        const newItems: ClaimLineItem[] = (b.items ?? []).map(
          (item, i) =>
            ({
              id: `li_${Date.now()}_${i}`,
              ada_code: item.ada_code,
              description: item.description,
              fee: (item.default_fee || 0) / 100,
            }) as any as ClaimLineItem
        );
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
      appointment_id: undefined,
      date_of_service: claimDateOfService,
      insurance_carrier: claimInsuranceCarrier,
      policy_number: claimPolicyNumber,
      group_number: claimGroupNumber,
      status: claimStatus,
      notes: claimNotes,
      line_items: claimLineItems.map((li) => ({
        ...li,
        fee: Math.round((li.fee || 0) * 100),
        insurance_allowed:
          li.insurance_allowed != null ? Math.round(li.insurance_allowed * 100) : undefined,
        patient_portion:
          li.patient_portion != null ? Math.round(li.patient_portion * 100) : undefined,
      })),
      created_at:
        isEditingClaim && editingClaimCreatedAt ? editingClaimCreatedAt : new Date().toISOString(),
      updated_at: new Date().toISOString(),
    } as any as Claim;

    try {
      if (isEditingClaim) {
        await BillingService.UpdateClaim(auth.token, payload);
      } else {
        await BillingService.CreateClaim(auth.token, payload);
      }
      showClaimModal = false;
      await loadClaims();
    } catch (e) {
      console.error("Failed to save claim:", e);
    }
  }

  async function deleteClaim(id: string) {
    if (!confirm(m.billing_claims_confirm_delete())) return;
    try {
      await BillingService.DeleteClaim(auth.token, id);
      await loadClaims();
    } catch (e) {
      console.error("Failed to delete claim:", e);
    }
  }

  async function submitClaim(id: string) {
    if (submittingClaims[id]) return;
    if (!confirm("Are you sure you want to submit this claim to the clearinghouse?")) return;

    try {
      submittingClaims[id] = true;
      const providersList = await BillingService.ListProviders();
      if (!providersList || providersList.length === 0) {
        alert("No claim providers registered. Check system configuration.");
        return;
      }

      let providerToUse = providersList[0];
      if (providersList.length > 1) {
        const choice = prompt(
          `Available providers: ${providersList.join(", ")}\nEnter provider to use:`,
          providersList[0]
        );
        if (!choice) return;
        if (!providersList.includes(choice)) {
          alert("Invalid provider selected.");
          return;
        }
        providerToUse = choice;
      }

      await BillingService.SubmitClaimToProvider(auth.token, id, providerToUse);
      await loadClaims();
    } catch (e) {
      console.error("Failed to submit claim:", e);
      alert("Failed to submit claim. Check console for details.");
    } finally {
      submittingClaims[id] = false;
    }
  }

  async function openChartImportModal() {
    if (!claimPatientId) {
      alert(m.billing_claim_err_patient());
      return;
    }
    loadingChartImport = true;
    selectedImportConditionIds = [];
    try {
      const chart = await ChartService.GetPatientChart(auth.token, claimPatientId);
      chartImportConditions = (chart?.conditions || []).filter(
        (c) => c.status === "treatment_planned" || c.status === "completed"
      );
      if (chartImportConditions.length === 0) {
        alert(m.billing_chart_import_empty());
        return;
      }
      showChartImportModal = true;
    } catch (e) {
      console.error("Failed to load chart conditions:", e);
      alert(m.billing_claim_err_load_chart());
    } finally {
      loadingChartImport = false;
    }
  }

  function applyChartImport() {
    const toImport = chartImportConditions.filter((c) => selectedImportConditionIds.includes(c.id));
    const newItems: ClaimLineItem[] = toImport.map(
      (cond, i) =>
        ({
          id: `li_${Date.now()}_${i}`,
          tooth_condition_id: cond.id,
          tooth_number: cond.tooth_number,
          surfaces: cond.surfaces,
          ada_code: cond.ada_code || "PROC",
          description: cond.description || `Tooth #${cond.tooth_number} procedure`,
          fee: (cond.fee || 0) / 100,
        }) as any as ClaimLineItem
    );

    claimLineItems = [...claimLineItems, ...newItems];
    showChartImportModal = false;
  }

  onMount(async () => {
    try {
      integrationProviders = (await BillingService.ListProviders()) || [];
      for (const provider of integrationProviders) {
        try {
          const config = (await BillingService.GetProviderConfig(provider)) as Record<string, any>;
          if (config && config["api_key"]) {
            hasConfiguredProvider = true;
            break;
          }
        } catch (e) {
          // Ignore individual provider config load errors
        }
      }
    } catch (e) {
      console.error("Failed to load providers:", e);
    }
    await loadClaims();
  });
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-4">
    <select
      bind:value={claimFilterPatient}
      onchange={loadClaims}
      class="w-full max-w-xs rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
    >
      <option value="">{m.billing_filter_all_patients()}</option>
      {#each patients as p}
        <option value={p.id}>{p.first_name} {p.last_name}</option>
      {/each}
    </select>

    <button type="button" class="btn btn-primary text-xs" onclick={openNewClaim}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {m.billing_btn_new_claim()}
    </button>
  </div>

  {#if loadingClaims}
    <div class="p-8 text-center text-sm text-slate-400">{m.common_loading()}</div>
  {:else if claims.length === 0}
    <EmptyState title={m.billing_no_claims()} icon="📄" />
  {:else}
    <div class="rounded-xl border border-slate-800 bg-slate-900/40 overflow-x-auto">
      <table class="w-full text-left text-sm text-slate-300">
        <thead
          class="bg-slate-900/80 border-b border-slate-800 text-xs font-semibold uppercase tracking-wider text-slate-400"
        >
          <tr>
            <th class="px-4 py-3">{m.billing_th_date()}</th>
            <th class="px-4 py-3">{m.billing_th_patient()}</th>
            <th class="px-4 py-3">{m.billing_th_provider()}</th>
            <th class="px-4 py-3">{m.billing_th_carrier()}</th>
            <th class="px-4 py-3">{m.billing_th_procedures()}</th>
            <th class="px-4 py-3">{m.billing_th_total()}</th>
            <th class="px-4 py-3">{m.billing_th_status()}</th>
            <th class="px-4 py-3 text-right">{m.patients_th_actions()}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          {#each claims as c (c.id)}
            <tr class="hover:bg-slate-900/50 transition-colors">
              <td class="px-4 py-3 text-slate-400 font-mono text-xs">{c.date_of_service}</td>
              <td class="px-4 py-3 font-semibold text-slate-100">{patientName(c.patient_id)}</td>
              <td class="px-4 py-3 text-slate-400">{providerName(c.provider_id)}</td>
              <td class="px-4 py-3 text-slate-400">{c.insurance_carrier || "—"}</td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5 flex-wrap">
                  {#each (c.line_items ?? []).slice(0, 2) as li}
                    <span
                      class="px-2 py-0.5 text-xs font-mono font-bold rounded bg-slate-800 text-sky-300 border border-slate-700"
                    >
                      {li.ada_code}
                    </span>
                  {/each}
                  {#if (c.line_items ?? []).length > 2}
                    <span class="text-slate-500 text-xs font-medium"
                      >+{(c.line_items ?? []).length - 2}</span
                    >
                  {/if}
                </div>
              </td>
              <td class="px-4 py-3 font-bold text-slate-100 font-mono">{fmt(claimTotal(c))}</td>
              <td class="px-4 py-3">
                <StatusBadge variant={c.status} />
              </td>
              <td class="px-4 py-3 text-right">
                <div class="flex items-center justify-end gap-1">
                  {#if c.status === "draft"}
                    <button
                      type="button"
                      class="p-1.5 text-slate-400 hover:text-emerald-400 rounded-lg hover:bg-slate-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      onclick={() => submitClaim(c.id)}
                      title={!hasConfiguredProvider
                        ? m.billing_claim_submit_disabled_tooltip()
                        : m.billing_btn_submit_claim()}
                      disabled={submittingClaims[c.id] || !hasConfiguredProvider}
                    >
                      <svg
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        class="h-4 w-4 pointer-events-none"
                      >
                        <line x1="22" y1="2" x2="11" y2="13"></line>
                        <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
                      </svg>
                    </button>
                  {/if}
                  <button
                    type="button"
                    class="p-1.5 text-slate-400 hover:text-sky-300 rounded-lg hover:bg-slate-800 transition-colors"
                    onclick={() => openEditClaim(c.id)}
                    title={m.patients_btn_edit()}
                  >
                    <svg
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      class="h-4 w-4 pointer-events-none"
                    >
                      <path
                        d="M11 5H6a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2v-5m-1.414-9.414a2 2 0 1 1 2.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                      />
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="p-1.5 text-slate-400 hover:text-rose-400 rounded-lg hover:bg-slate-800 transition-colors"
                    onclick={() => deleteClaim(c.id)}
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
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- CLAIM MODAL -->
<Modal
  bind:showModal={showClaimModal}
  title={isEditingClaim ? "Edit Claim" : m.billing_btn_new_claim()}
  subtitle="Configure claim details, insurance carrier policy numbers, and CDT line items"
  icon="📄"
  maxWidth="max-w-4xl"
>
  <form onsubmit={saveClaim} class="space-y-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <FormField label={m.appt_label_patient()} forId="cl-patient" required>
        <select
          id="cl-patient"
          bind:value={claimPatientId}
          required
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          {#each patients as p}
            <option value={p.id}>{p.first_name} {p.last_name}</option>
          {/each}
        </select>
      </FormField>

      <FormField label={m.appt_label_provider()} forId="cl-provider">
        <select
          id="cl-provider"
          bind:value={claimProviderId}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          <option value="">— None —</option>
          {#each providers as pr}
            <option value={pr.id}>{pr.name}</option>
          {/each}
        </select>
      </FormField>

      <FormField label={m.appt_label_date()} forId="cl-dos" required>
        <Input id="cl-dos" type="date" bind:value={claimDateOfService} required />
      </FormField>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <FormField label={m.patient_insurance_carrier()} forId="cl-carrier">
        <Input
          id="cl-carrier"
          type="text"
          bind:value={claimInsuranceCarrier}
          placeholder="e.g. Delta Dental"
        />
      </FormField>
      <FormField label={m.patient_insurance_policy()} forId="cl-policy">
        <Input
          id="cl-policy"
          type="text"
          bind:value={claimPolicyNumber}
          placeholder={m.billing_claim_policy_placeholder()}
        />
      </FormField>
      <FormField label={m.patient_insurance_group()} forId="cl-group">
        <Input
          id="cl-group"
          type="text"
          bind:value={claimGroupNumber}
          placeholder={m.billing_claim_group_placeholder()}
        />
      </FormField>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <FormField label={m.appt_label_status()} forId="cl-status">
        <select
          id="cl-status"
          bind:value={claimStatus}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          {#each CLAIM_STATUSES as s}
            <option value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>
          {/each}
        </select>
      </FormField>
      <FormField label={m.appt_label_notes()} forId="cl-notes">
        <Input
          id="cl-notes"
          type="text"
          bind:value={claimNotes}
          placeholder={m.billing_pay_notes_placeholder()}
        />
      </FormField>
    </div>

    <div class="rounded-xl border border-slate-800 bg-slate-950 p-4 space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h4 class="text-xs font-bold uppercase tracking-wider text-slate-400">
          {m.billing_th_procedures()}
        </h4>
        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            class="btn btn-secondary btn-sm flex items-center gap-1 bg-emerald-950/60 border border-emerald-500/40 text-emerald-300 hover:bg-emerald-900/60"
            onclick={openChartImportModal}
          >
            🦷 {m.billing_btn_import_chart()}
          </button>
          <div class="flex items-center gap-1.5">
            <input
              type="text"
              class="w-36 rounded-lg border border-slate-700 bg-slate-900 px-2.5 py-1 text-xs font-mono text-slate-100 placeholder-slate-500 focus:border-sky-500 focus:outline-none"
              bind:value={bundleLookupInput}
              placeholder={m.billing_claim_bundle_placeholder()}
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
        <p class="text-xs text-rose-400 m-0">⚠️ {bundleLookupError}</p>
      {/if}

      {#if claimLineItems.length > 0}
        <div class="space-y-2">
          <div
            class="grid grid-cols-12 gap-2 text-[10px] font-bold uppercase tracking-wider text-slate-500 px-1"
          >
            <span class="col-span-2">{m.charting_th_code()}</span>
            <span class="col-span-3">{m.charting_th_desc()}</span>
            <span class="col-span-1 text-center">{m.billing_claim_tooth_label()}</span>
            <span class="col-span-2 text-right">{m.charting_th_fee()}</span>
            <span class="col-span-2 text-right">Ins. Allowed</span>
            <span class="col-span-1 text-center"></span>
          </div>
          {#each claimLineItems as li, i}
            <div class="grid grid-cols-12 gap-2 items-center">
              <div class="col-span-2">
                <Input
                  bind:value={li.ada_code}
                  placeholder={m.billing_claim_code_placeholder()}
                  class="font-mono text-xs py-1.5 px-2"
                />
              </div>
              <div class="col-span-3">
                <Input
                  bind:value={li.description}
                  placeholder={m.charting_th_desc()}
                  class="text-xs py-1.5 px-2"
                />
              </div>
              <div class="col-span-1">
                <Input
                  type="number"
                  bind:value={li.tooth_number}
                  min="1"
                  max="32"
                  placeholder="—"
                  class="text-center text-xs py-1.5 px-1"
                />
              </div>
              <div class="col-span-2">
                <Input
                  type="number"
                  bind:value={li.fee}
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  class="text-right text-xs py-1.5 px-2"
                />
              </div>
              <div class="col-span-2">
                <Input
                  type="number"
                  bind:value={li.insurance_allowed}
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
                  onclick={() => removeLineItem(i)}
                  title="Remove row">✕</button
                >
              </div>
            </div>
          {/each}
          <div class="text-right text-xs text-slate-400 pt-2 border-t border-slate-800">
            Total: <strong class="text-white text-sm font-mono"
              >{fmt(
                Math.round(claimLineItems.reduce((s, li) => s + (li.fee || 0), 0) * 100)
              )}</strong
            >
          </div>
        </div>
      {:else}
        <div
          class="p-6 text-center text-xs text-slate-500 bg-slate-900/50 rounded-xl border border-dashed border-slate-800"
        >
          No line items added yet. Click '+ Add Row' or apply a bundle shortname above.
        </div>
      {/if}
    </div>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
        onclick={() => (showClaimModal = false)}
      >
        {m.common_cancel()}
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        {isEditingClaim ? m.patient_save_changes() : m.billing_btn_new_claim()}
      </button>
    </div>
  </form>
</Modal>

<!-- Import Chart Conditions Modal -->
<Modal
  bind:showModal={showChartImportModal}
  title={m.billing_chart_import_title()}
  subtitle={m.billing_chart_import_desc()}
  icon="🦷"
  maxWidth="max-w-2xl"
>
  <div class="space-y-4">
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
            class="flex items-center justify-between p-3 hover:bg-slate-900 cursor-pointer text-xs transition-colors"
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

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
        onclick={() => (showChartImportModal = false)}
      >
        {m.common_cancel()}
      </button>
      <button
        type="button"
        class="btn btn-primary text-xs px-5 py-2 cursor-pointer"
        disabled={selectedImportConditionIds.length === 0}
        onclick={applyChartImport}
      >
        {m.billing_chart_import_submit({ count: selectedImportConditionIds.length })}
      </button>
    </div>
  </div>
</Modal>
