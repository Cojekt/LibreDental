<script lang="ts">
  import { onMount } from "svelte";
  import { BillingService } from "@bindings/services/index.js";
  import type {
    Patient,
    Provider,
    Claim,
    ClaimLineItem,
    Payment,
    PatientBalance,
    BundleItemTemplate,
    TreatmentBundle,
  } from "@bindings/domain/index.js";
  import { ClaimStatus, PaymentMethod } from "@bindings/domain/index.js";

  // ── Props ─────────────────────────────────────────────────────────────────
  let { patients = [], providers = [] } = $props<{
    patients: Patient[];
    providers: Provider[];
  }>();

  // ── Tab State ─────────────────────────────────────────────────────────────
  let billingTab = $state<"claims" | "payments" | "bundles">("claims");

  // ── Claims State ──────────────────────────────────────────────────────────
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

  // ── Payments State ────────────────────────────────────────────────────────
  let payments = $state<Payment[]>([]);
  let loadingPayments = $state(false);
  let patientBalance = $state<PatientBalance | null>(null);
  let balancePatientId = $state("");
  let showPaymentModal = $state(false);

  // Payment form
  let payPatientId = $state("");
  let payClaimId = $state("");
  let payAmount = $state("");
  let payMethod = $state<PaymentMethod>(PaymentMethod.PaymentMethodCash);
  let payDate = $state(new Date().toISOString().split("T")[0]);
  let payNotes = $state("");

  // ── Bundles State ─────────────────────────────────────────────────────────
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

  // ── Helpers ───────────────────────────────────────────────────────────────
  function patientName(id: string) {
    const p = patients.find((p: Patient) => p.id === id);
    return p ? `${p.first_name} ${p.last_name}` : id;
  }

  function providerName(id: string) {
    const p = providers.find((p: Provider) => p.id === id);
    return p ? p.name : id;
  }

  function fmt(n: number) {
    return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(n);
  }

  function claimTotal(c: Claim) {
    return (c.line_items ?? []).reduce((s, i) => s + i.fee, 0);
  }

  // ── Claims ────────────────────────────────────────────────────────────────
  async function loadClaims() {
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

  function openNewClaim() {
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

  // ── Payments ──────────────────────────────────────────────────────────────
  async function loadPayments() {
    loadingPayments = true;
    try {
      const res = await BillingService.ListPayments(balancePatientId);
      payments = (res?.filter(Boolean) as Payment[]) || [];
    } catch (e) {
      console.error("Failed to load payments:", e);
    } finally {
      loadingPayments = false;
    }
  }

  async function loadBalance() {
    if (!balancePatientId) {
      patientBalance = null;
      return;
    }
    try {
      patientBalance = (await BillingService.GetPatientBalance(
        balancePatientId
      )) as PatientBalance | null;
    } catch (e) {
      console.error("Failed to load balance:", e);
    }
  }

  $effect(() => {
    if (billingTab === "payments") {
      loadPayments();
      loadBalance();
    }
  });

  $effect(() => {
    // Re-fetch balance when patient selection changes on payments tab
    if (balancePatientId !== undefined) {
      loadPayments();
      loadBalance();
    }
  });

  function openNewPayment() {
    payPatientId = balancePatientId || (patients[0]?.id ?? "");
    payClaimId = "";
    payAmount = "";
    payMethod = PaymentMethod.PaymentMethodCash;
    payDate = new Date().toISOString().split("T")[0];
    payNotes = "";
    showPaymentModal = true;
  }

  async function savePayment(e: Event) {
    e.preventDefault();
    const amount = parseFloat(payAmount);
    if (!payPatientId || isNaN(amount) || amount <= 0 || !payDate) return;

    const payload: Payment = {
      id: `pay_${Date.now()}`,
      patient_id: payPatientId,
      claim_id: payClaimId,
      amount,
      method: payMethod,
      date: payDate,
      notes: payNotes,
      created_at: new Date().toISOString(),
    };

    try {
      await BillingService.RecordPayment(payload as any);
      showPaymentModal = false;
      balancePatientId = payPatientId;
      await loadPayments();
      await loadBalance();
    } catch (e) {
      console.error("Failed to record payment:", e);
    }
  }

  async function deletePayment(id: string) {
    if (!confirm("Delete this payment record?")) return;
    try {
      await BillingService.DeletePayment(id);
      await loadPayments();
      await loadBalance();
    } catch (e) {
      console.error("Failed to delete payment:", e);
    }
  }

  // ── Bundles ───────────────────────────────────────────────────────────────
  async function loadBundles() {
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

  $effect(() => {
    if (billingTab === "bundles") loadBundles();
  });

  function openNewBundle() {
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

  // ── Init ──────────────────────────────────────────────────────────────────
  onMount(async () => {
    await loadClaims();
    await loadBundles();
  });

  const CLAIM_STATUSES = ["draft", "submitted", "accepted", "rejected", "paid"];
  const PAYMENT_METHODS = ["cash", "check", "credit_card", "insurance", "write_off"];

  const statusColors: Record<string, string> = {
    draft: "bg-slate-700 text-slate-300",
    submitted: "bg-blue-900/60 text-blue-300",
    accepted: "bg-emerald-900/60 text-emerald-300",
    rejected: "bg-red-900/60 text-red-300",
    paid: "bg-teal-900/60 text-teal-300",
  };

  const methodColors: Record<string, string> = {
    cash: "bg-emerald-900/60 text-emerald-300",
    check: "bg-sky-900/60 text-sky-300",
    credit_card: "bg-violet-900/60 text-violet-300",
    insurance: "bg-blue-900/60 text-blue-300",
    write_off: "bg-slate-700 text-slate-400",
  };
</script>

<!-- ─── Layout ──────────────────────────────────────────────────────────────── -->
<div class="billing-view">
  <!-- Page Header -->
  <div class="billing-header">
    <div>
      <h2 class="billing-title">Billing</h2>
      <p class="billing-subtitle">Claims, payments, and procedure bundles</p>
    </div>
    <div class="billing-header-actions">
      {#if billingTab === "claims"}
        <button class="btn btn-primary" onclick={openNewClaim}>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-4 w-4"
          >
            <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          New Claim
        </button>
      {:else if billingTab === "payments"}
        <button class="btn btn-primary" onclick={openNewPayment}>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-4 w-4"
          >
            <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          Record Payment
        </button>
      {:else if billingTab === "bundles"}
        <button class="btn btn-primary" onclick={openNewBundle}>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-4 w-4"
          >
            <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          New Bundle
        </button>
      {/if}
    </div>
  </div>

  <!-- Inner Tabs -->
  <div class="billing-tabs">
    {#each [["claims", "Claims", "M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414A1 1 0 0 1 19 9.414V19a2 2 0 0 1-2 2z"], ["payments", "Payments", "M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 0 0 3-3V8a3 3 0 0 0-3-3H6a3 3 0 0 0-3 3v8a3 3 0 0 0 3 3z"], ["bundles", "Bundles", "M19 11H5m14 0a2 2 0 0 1 2 2v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6a2 2 0 0 1 2-2m14 0V9a2 2 0 0 1-2-2M5 11V9a2 2 0 0 1 2-2m0 0V5a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2M7 7h10"]] as [id, label, path]}
      <button
        class="billing-tab-btn {billingTab === id ? 'billing-tab-active' : ''}"
        onclick={() => (billingTab = id as any)}
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
          <path d={path} stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        {label}
      </button>
    {/each}
  </div>

  <!-- ── CLAIMS TAB ──────────────────────────────────────────────────────── -->
  {#if billingTab === "claims"}
    <div class="billing-toolbar">
      <select bind:value={claimFilterPatient} onchange={loadClaims} class="billing-filter-select">
        <option value="">All Patients</option>
        {#each patients as p}
          <option value={p.id}>{p.first_name} {p.last_name}</option>
        {/each}
      </select>
    </div>

    {#if loadingClaims}
      <div class="billing-loading">Loading claims…</div>
    {:else if claims.length === 0}
      <div class="billing-empty">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          class="billing-empty-icon"
        >
          <path
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414A1 1 0 0 1 19 9.414V19a2 2 0 0 1-2 2z"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        <p>No claims yet. Create one to get started.</p>
      </div>
    {:else}
      <div class="claims-table-wrap">
        <table class="claims-table">
          <thead>
            <tr>
              <th>Date</th>
              <th>Patient</th>
              <th>Provider</th>
              <th>Carrier</th>
              <th>Procedures</th>
              <th>Total</th>
              <th>Status</th>
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
                  <span
                    class="status-badge {statusColors[c.status] ?? 'bg-slate-700 text-slate-300'}"
                  >
                    {c.status}
                  </span>
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

    <!-- ── PAYMENTS TAB ────────────────────────────────────────────────────── -->
  {:else if billingTab === "payments"}
    <div class="payments-layout">
      <!-- Patient picker + balance card -->
      <div class="balance-panel">
        <div class="balance-patient-select">
          <label class="form-label" for="balance-patient">Patient</label>
          <select id="balance-patient" bind:value={balancePatientId} class="w-full">
            <option value="">— Select patient —</option>
            {#each patients as p}
              <option value={p.id}>{p.first_name} {p.last_name}</option>
            {/each}
          </select>
        </div>

        {#if patientBalance && balancePatientId}
          <div class="balance-cards">
            <div class="balance-card balance-billed">
              <div class="balance-label">Total Billed</div>
              <div class="balance-amount">{fmt(patientBalance.total_billed)}</div>
            </div>
            <div class="balance-card balance-paid">
              <div class="balance-label">Total Paid</div>
              <div class="balance-amount">{fmt(patientBalance.total_paid)}</div>
            </div>
            <div
              class="balance-card {patientBalance.outstanding > 0
                ? 'balance-outstanding'
                : 'balance-clear'}"
            >
              <div class="balance-label">Outstanding</div>
              <div class="balance-amount">{fmt(patientBalance.outstanding)}</div>
            </div>
          </div>
        {:else if balancePatientId}
          <div class="balance-cards">
            <div class="balance-card balance-clear">
              <div class="balance-label">Outstanding</div>
              <div class="balance-amount">$0.00</div>
            </div>
          </div>
        {/if}
      </div>

      <!-- Payment log -->
      <div class="payment-log">
        <div class="payment-log-header">
          <h3 class="payment-log-title">Payment Log</h3>
          {#if loadingPayments}
            <span class="text-slate-400 text-xs">Loading…</span>
          {/if}
        </div>

        {#if payments.length === 0 && !loadingPayments}
          <div class="billing-empty billing-empty-sm">
            <p>No payments recorded{balancePatientId ? " for this patient" : ""}.</p>
          </div>
        {:else}
          <div class="payment-list">
            {#each payments as pay (pay.id)}
              <div class="payment-row">
                <div class="payment-row-left">
                  <span class="payment-amount">{fmt(pay.amount)}</span>
                  <span
                    class="status-badge {methodColors[pay.method] ?? 'bg-slate-700 text-slate-300'}"
                  >
                    {pay.method.replace("_", " ")}
                  </span>
                  {#if pay.claim_id}
                    <span class="payment-claim-ref">Claim #{pay.claim_id.slice(-6)}</span>
                  {/if}
                </div>
                <div class="payment-row-right">
                  <span class="text-slate-400 text-sm">{patientName(pay.patient_id)}</span>
                  <span class="text-slate-500 text-xs">{pay.date}</span>
                  {#if pay.notes}
                    <span class="text-slate-500 text-xs italic">{pay.notes}</span>
                  {/if}
                  <button
                    class="action-btn action-btn-danger"
                    onclick={() => deletePayment(pay.id)}
                    title="Delete"
                  >
                    <svg
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      class="h-3.5 w-3.5"
                    >
                      <path
                        d="M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16"
                      />
                    </svg>
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <!-- ── BUNDLES TAB ─────────────────────────────────────────────────────── -->
  {:else if billingTab === "bundles"}
    {#if loadingBundles}
      <div class="billing-loading">Loading bundles…</div>
    {:else if bundles.length === 0}
      <div class="billing-empty">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          class="billing-empty-icon"
        >
          <path
            d="M19 11H5m14 0a2 2 0 0 1 2 2v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-6a2 2 0 0 1 2-2m14 0V9a2 2 0 0 1-2-2M5 11V9a2 2 0 0 1 2-2m0 0V5a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2M7 7h10"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        <p>No procedure bundles yet. Create one to speed up claim entry.</p>
      </div>
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
  {/if}
</div>

<!-- ══════════════════════════════════════════════════════════════════════════
     CLAIM MODAL
═══════════════════════════════════════════════════════════════════════════ -->
{#if showClaimModal}
  <div class="modal-backdrop" role="dialog" aria-modal="true">
    <div class="modal-box modal-wide">
      <div class="modal-header">
        <h2 class="modal-title">{isEditingClaim ? "Edit Claim" : "New Insurance Claim"}</h2>
        <button class="modal-close" onclick={() => (showClaimModal = false)}>✕</button>
      </div>

      <form onsubmit={saveClaim} class="modal-body">
        <!-- Row 1: Patient / Provider / Date -->
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

        <!-- Row 2: Insurance -->
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

        <!-- Status + Notes -->
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

        <!-- Line Items Section -->
        <div class="line-items-section">
          <div class="line-items-header">
            <span class="form-label mb-0">Procedure Line Items</span>
            <div class="line-items-header-actions">
              <!-- Bundle shortname lookup -->
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

<!-- ══════════════════════════════════════════════════════════════════════════
     PAYMENT MODAL
═══════════════════════════════════════════════════════════════════════════ -->
{#if showPaymentModal}
  <div class="modal-backdrop" role="dialog" aria-modal="true">
    <div class="modal-box">
      <div class="modal-header">
        <h2 class="modal-title">Record Payment</h2>
        <button class="modal-close" onclick={() => (showPaymentModal = false)}>✕</button>
      </div>

      <form onsubmit={savePayment} class="modal-body">
        <div class="form-grid-2">
          <div class="form-field">
            <label class="form-label" for="pay-patient">Patient *</label>
            <select id="pay-patient" bind:value={payPatientId} required>
              {#each patients as p}
                <option value={p.id}>{p.first_name} {p.last_name}</option>
              {/each}
            </select>
          </div>
          <div class="form-field">
            <label class="form-label" for="pay-date">Date *</label>
            <input id="pay-date" type="date" bind:value={payDate} required />
          </div>
        </div>

        <div class="form-grid-2">
          <div class="form-field">
            <label class="form-label" for="pay-amount">Amount *</label>
            <input
              id="pay-amount"
              type="number"
              bind:value={payAmount}
              step="0.01"
              min="0.01"
              placeholder="0.00"
              required
            />
          </div>
          <div class="form-field">
            <label class="form-label" for="pay-method">Method *</label>
            <select id="pay-method" bind:value={payMethod}>
              {#each PAYMENT_METHODS as m}
                <option value={m}
                  >{m.replace("_", " ").replace(/\b\w/g, (c) => c.toUpperCase())}</option
                >
              {/each}
            </select>
          </div>
        </div>

        <div class="form-field">
          <label class="form-label" for="pay-claim">Link to Claim (optional)</label>
          <select id="pay-claim" bind:value={payClaimId}>
            <option value="">— None —</option>
            {#each claims.filter((c) => c.patient_id === payPatientId) as c}
              <option value={c.id}
                >{c.date_of_service} — {c.insurance_carrier || "No carrier"} ({fmt(
                  claimTotal(c)
                )})</option
              >
            {/each}
          </select>
        </div>

        <div class="form-field">
          <label class="form-label" for="pay-notes">Notes</label>
          <input id="pay-notes" type="text" bind:value={payNotes} placeholder="Optional notes" />
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" onclick={() => (showPaymentModal = false)}
            >Cancel</button
          >
          <button type="submit" class="btn btn-primary">Record Payment</button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- ══════════════════════════════════════════════════════════════════════════
     BUNDLE MODAL
═══════════════════════════════════════════════════════════════════════════ -->
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

        <!-- Bundle items -->
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

<style>
  /* ── Layout ──────────────────────────────────────────────────────────── */
  .billing-view {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .billing-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }
  .billing-title {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--color-slate-50, #f8fafc);
    margin: 0;
  }
  .billing-subtitle {
    font-size: 0.85rem;
    color: #64748b;
    margin: 0.2rem 0 0;
  }
  .billing-header-actions {
    display: flex;
    gap: 0.5rem;
  }

  /* ── Inner Tabs ──────────────────────────────────────────────────────── */
  .billing-tabs {
    display: flex;
    gap: 0.25rem;
    border-bottom: 1px solid #1e293b;
    padding-bottom: 0;
  }
  .billing-tab-btn {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.55rem 1rem;
    font-size: 0.85rem;
    font-weight: 600;
    color: #64748b;
    background: transparent;
    border: none;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
    transition:
      color 0.15s,
      border-color 0.15s;
    border-radius: 0.375rem 0.375rem 0 0;
  }
  .billing-tab-btn:hover {
    color: #94a3b8;
  }
  .billing-tab-active {
    color: #38bdf8 !important;
    border-bottom-color: #38bdf8 !important;
  }

  /* ── Toolbar ─────────────────────────────────────────────────────────── */
  .billing-toolbar {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  .billing-filter-select {
    max-width: 18rem;
  }

  /* ── Loading / Empty ─────────────────────────────────────────────────── */
  .billing-loading {
    color: #64748b;
    font-size: 0.9rem;
    padding: 2rem 0;
    text-align: center;
  }
  .billing-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
    padding: 4rem 0;
    color: #475569;
    text-align: center;
  }
  .billing-empty-sm {
    padding: 2rem 0;
  }
  .billing-empty-icon {
    width: 3rem;
    height: 3rem;
    opacity: 0.5;
  }

  /* ── Claims Table ────────────────────────────────────────────────────── */
  .claims-table-wrap {
    overflow-x: auto;
    border-radius: 0.75rem;
    border: 1px solid #1e293b;
  }
  .claims-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
  }
  .claims-table thead tr {
    background: #0f172a;
  }
  .claims-table th {
    padding: 0.75rem 1rem;
    text-align: left;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    color: #64748b;
    text-transform: uppercase;
    border-bottom: 1px solid #1e293b;
  }
  .claim-row {
    border-bottom: 1px solid #1e293b;
    transition: background 0.1s;
  }
  .claim-row:hover {
    background: #0f172a;
  }
  .claims-table td {
    padding: 0.75rem 1rem;
    color: #cbd5e1;
    vertical-align: middle;
  }
  .claim-date {
    color: #94a3b8;
    font-variant-numeric: tabular-nums;
  }
  .claim-fee {
    font-weight: 700;
    color: #f8fafc;
    font-variant-numeric: tabular-nums;
  }
  .claim-actions {
    display: flex;
    gap: 0.25rem;
  }

  .line-items-preview {
    display: flex;
    gap: 0.25rem;
    align-items: center;
    flex-wrap: wrap;
  }

  /* ── Payments ────────────────────────────────────────────────────────── */
  .payments-layout {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .balance-panel {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .balance-patient-select {
    max-width: 20rem;
  }
  .balance-cards {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
  }
  .balance-card {
    flex: 1;
    min-width: 9rem;
    padding: 1rem 1.25rem;
    border-radius: 0.75rem;
    border: 1px solid #1e293b;
    background: #0f172a;
  }
  .balance-label {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: #64748b;
    margin-bottom: 0.35rem;
  }
  .balance-amount {
    font-size: 1.4rem;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
  }
  .balance-billed .balance-amount {
    color: #94a3b8;
  }
  .balance-paid .balance-amount {
    color: #34d399;
  }
  .balance-outstanding {
    border-color: #f97316;
  }
  .balance-outstanding .balance-amount {
    color: #f97316;
  }
  .balance-clear .balance-amount {
    color: #34d399;
  }

  .payment-log-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }
  .payment-log-title {
    font-size: 1rem;
    font-weight: 700;
    color: #f1f5f9;
    margin: 0;
  }
  .payment-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .payment-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
    border-radius: 0.625rem;
    border: 1px solid #1e293b;
    background: #0f172a;
    transition: background 0.1s;
  }
  .payment-row:hover {
    background: #111827;
  }
  .payment-row-left {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .payment-row-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  .payment-amount {
    font-size: 1rem;
    font-weight: 700;
    color: #f8fafc;
    font-variant-numeric: tabular-nums;
  }
  .payment-claim-ref {
    font-size: 0.75rem;
    color: #64748b;
    font-family: monospace;
  }

  /* ── Bundles ─────────────────────────────────────────────────────────── */
  .bundle-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(22rem, 1fr));
    gap: 1rem;
  }
  .bundle-card {
    border: 1px solid #1e293b;
    border-radius: 0.75rem;
    background: #0f172a;
    display: flex;
    flex-direction: column;
    gap: 0;
    overflow: hidden;
    transition: border-color 0.15s;
  }
  .bundle-card:hover {
    border-color: #334155;
  }
  .bundle-archived {
    opacity: 0.6;
  }

  .bundle-card-header {
    padding: 1rem 1rem 0.75rem;
    border-bottom: 1px solid #1e293b;
  }
  .bundle-card-title-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .bundle-name {
    font-weight: 700;
    color: #f1f5f9;
    font-size: 0.95rem;
  }
  .bundle-description {
    color: #64748b;
    font-size: 0.8rem;
    margin: 0.3rem 0 0;
  }

  .shortname-badge {
    font-family: ui-monospace, "Cascadia Code", monospace;
    font-size: 0.78rem;
    font-weight: 700;
    background: linear-gradient(135deg, #1e3a5f, #0c4a6e);
    color: #38bdf8;
    border: 1px solid #0369a1;
    padding: 0.15rem 0.55rem;
    border-radius: 0.375rem;
    letter-spacing: 0.04em;
  }

  .bundle-items-list {
    padding: 0.5rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    flex: 1;
  }
  .bundle-item-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.82rem;
  }
  .bundle-item-desc {
    flex: 1;
    color: #94a3b8;
  }
  .bundle-item-fee {
    font-variant-numeric: tabular-nums;
    color: #f8fafc;
    font-weight: 600;
  }

  .bundle-card-footer {
    padding: 0.65rem 1rem;
    border-top: 1px solid #1e293b;
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: #080f1a;
  }
  .bundle-total {
    font-size: 0.82rem;
    color: #64748b;
  }
  .bundle-total strong {
    color: #f8fafc;
    font-size: 0.9rem;
  }
  .bundle-card-actions {
    display: flex;
    gap: 0.25rem;
  }

  /* ── Shared Badges ───────────────────────────────────────────────────── */
  .ada-badge {
    font-family: ui-monospace, "Cascadia Code", monospace;
    font-size: 0.72rem;
    font-weight: 700;
    background: #1e293b;
    color: #7dd3fc;
    padding: 0.1rem 0.45rem;
    border-radius: 0.3rem;
    border: 1px solid #334155;
  }
  .status-badge {
    display: inline-flex;
    align-items: center;
    font-size: 0.72rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    padding: 0.15rem 0.55rem;
    border-radius: 9999px;
    text-transform: capitalize;
  }

  /* ── Action Buttons ──────────────────────────────────────────────────── */
  .action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.9rem;
    height: 1.9rem;
    border-radius: 0.375rem;
    background: transparent;
    border: 1px solid #334155;
    color: #94a3b8;
    cursor: pointer;
    transition: all 0.15s;
  }
  .action-btn:hover {
    background: #1e293b;
    color: #f1f5f9;
  }
  .action-btn-danger:hover {
    background: rgba(239, 68, 68, 0.12);
    border-color: rgba(239, 68, 68, 0.4);
    color: #ef4444;
  }

  /* ── Modal ───────────────────────────────────────────────────────────── */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    backdrop-filter: blur(4px);
  }
  .modal-box {
    background: #0f172a;
    border: 1px solid #1e293b;
    border-radius: 1rem;
    width: 100%;
    max-width: 36rem;
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 0 25px 60px rgba(0, 0, 0, 0.6);
  }
  .modal-wide {
    max-width: 64rem;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.25rem 1.5rem 0;
  }
  .modal-title {
    font-size: 1.1rem;
    font-weight: 700;
    color: #f1f5f9;
    margin: 0;
  }
  .modal-close {
    background: none;
    border: none;
    color: #64748b;
    font-size: 1rem;
    cursor: pointer;
    padding: 0.25rem;
    transition: color 0.15s;
  }
  .modal-close:hover {
    color: #f1f5f9;
  }
  .modal-body {
    padding: 1.25rem 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding-top: 0.5rem;
  }

  /* ── Form Helpers ────────────────────────────────────────────────────── */
  .form-grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }
  .form-grid-3 {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 0.75rem;
  }
  @media (max-width: 640px) {
    .form-grid-2,
    .form-grid-3 {
      grid-template-columns: 1fr;
    }
  }
  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }
  .form-label {
    font-size: 0.78rem;
    font-weight: 600;
    color: #94a3b8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .form-label-hint {
    font-size: 0.72rem;
    font-weight: 400;
    color: #64748b;
    text-transform: none;
    letter-spacing: 0;
  }
  .field-error {
    font-size: 0.78rem;
    color: #ef4444;
    margin: 0;
  }

  /* ── Line Items Editor ───────────────────────────────────────────────── */
  .line-items-section {
    border: 1px solid #1e293b;
    border-radius: 0.625rem;
    padding: 0.875rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    background: #080f1a;
  }
  .line-items-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .line-items-header-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .line-items-grid-header {
    display: grid;
    grid-template-columns: 6rem 1fr 4rem 5rem 5rem 5rem 2rem;
    gap: 0.4rem;
    font-size: 0.7rem;
    font-weight: 600;
    color: #475569;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 0 0.25rem;
  }
  .line-item-row {
    display: grid;
    grid-template-columns: 6rem 1fr 4rem 5rem 5rem 5rem 2rem;
    gap: 0.4rem;
    align-items: center;
  }
  .line-item-row input {
    padding: 0.3rem 0.5rem;
    font-size: 0.8rem;
  }
  .li-ada,
  .li-tooth,
  .li-fee {
    text-align: center;
  }
  .line-items-total {
    text-align: right;
    font-size: 0.85rem;
    color: #94a3b8;
    padding-top: 0.25rem;
  }
  .line-items-total strong {
    color: #f8fafc;
  }
  .line-items-empty {
    color: #475569;
    font-size: 0.82rem;
    text-align: center;
    padding: 1rem 0;
  }

  /* ── Bundle Items Editor ─────────────────────────────────────────────── */
  .bundle-items-grid-header {
    display: grid;
    grid-template-columns: 6rem 1fr 5rem 2rem;
    gap: 0.4rem;
    font-size: 0.7rem;
    font-weight: 600;
    color: #475569;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 0 0.25rem;
  }
  .bundle-item-edit-row {
    display: grid;
    grid-template-columns: 6rem 1fr 5rem 2rem;
    gap: 0.4rem;
    align-items: center;
  }
  .bundle-item-edit-row input {
    padding: 0.3rem 0.5rem;
    font-size: 0.8rem;
  }

  /* ── Bundle Lookup ───────────────────────────────────────────────────── */
  .bundle-lookup {
    display: flex;
    gap: 0.4rem;
    align-items: center;
  }
  .bundle-lookup-input {
    width: 11rem;
    padding: 0.3rem 0.6rem;
    font-family: ui-monospace, monospace;
    font-size: 0.82rem;
    letter-spacing: 0.04em;
  }
  .lookup-error {
    font-size: 0.78rem;
    color: #f87171;
    margin: 0;
  }

  /* Light theme */
  :global(html.light) .billing-title {
    color: #0f172a;
  }
  :global(html.light) .bundle-card,
  :global(html.light) .claims-table-wrap,
  :global(html.light) .payment-row,
  :global(html.light) .balance-card,
  :global(html.light) .line-items-section {
    background: #ffffff;
    border-color: #e2e8f0;
  }
  :global(html.light) .bundle-card-footer {
    background: #f8fafc;
  }
  :global(html.light) .claims-table thead tr {
    background: #f1f5f9;
  }
  :global(html.light) .claims-table th {
    color: #475569;
    border-color: #e2e8f0;
  }
  :global(html.light) .claims-table td {
    color: #0f172a;
  }
  :global(html.light) .claim-row:hover {
    background: #f8fafc;
  }
  :global(html.light) .payment-row:hover {
    background: #f8fafc;
  }
  :global(html.light) .modal-box {
    background: #ffffff;
    border-color: #cbd5e1;
  }
</style>
