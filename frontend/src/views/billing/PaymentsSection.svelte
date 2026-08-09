<script lang="ts">
  import { BillingService } from "@bindings/services/index.js";
  import type {
    Patient,
    Claim,
    Payment,
    PatientBalance,
    CountryConfig,
  } from "@bindings/domain/index.js";
  import { PaymentMethod } from "@bindings/domain/index.js";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

  let {
    patients = [],
    countryMeta = null,
  } = $props<{
    patients: Patient[];
    countryMeta?: CountryConfig | null;
  }>();

  let payments = $state<Payment[]>([]);
  let claims = $state<Claim[]>([]);
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

  const PAYMENT_METHODS = ["cash", "check", "credit_card", "insurance", "write_off"];

  function patientName(id: string) {
    const p = patients.find((p: Patient) => p.id === id);
    return p ? `${p.first_name} ${p.last_name}` : id;
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

  export async function loadPayments() {
    loadingPayments = true;
    try {
      const res = await BillingService.ListPayments(balancePatientId);
      payments = (res?.filter(Boolean) as Payment[]) || [];
      const claimRes = await BillingService.ListClaims(balancePatientId);
      claims = (claimRes?.filter(Boolean) as Claim[]) || [];
    } catch (e) {
      console.error("Failed to load payments/claims:", e);
    } finally {
      loadingPayments = false;
    }
  }

  export async function loadBalance() {
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
    if (balancePatientId !== undefined) {
      loadPayments();
      loadBalance();
    }
  });

  export function openNewPayment() {
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
</script>

<div class="payments-layout space-y-4">
  <div class="flex justify-end mb-2">
    <button type="button" class="btn btn-primary" onclick={openNewPayment}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {m.billing_btn_record_payment()}
    </button>
  </div>

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

  <div class="payment-log">
    <div class="payment-log-header">
      <h3 class="payment-log-title">Payment Log</h3>
      {#if loadingPayments}
        <span class="text-slate-400 text-xs">Loading…</span>
      {/if}
    </div>

    {#if payments.length === 0 && !loadingPayments}
      <EmptyState
        title={`No payments recorded${balancePatientId ? " for this patient" : ""}.`}
      />
    {:else}
      <div class="payment-list">
        {#each payments as pay (pay.id)}
          <div class="payment-row">
            <div class="payment-row-left">
              <span class="payment-amount">{fmt(pay.amount)}</span>
              <StatusBadge variant={pay.method} label={pay.method.replace("_", " ")} />
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
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-3.5 w-3.5">
                  <path d="M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<!-- PAYMENT MODAL -->
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
                <option value={m}>{m.replace("_", " ").replace(/\b\w/g, (c) => c.toUpperCase())}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="form-field">
          <label class="form-label" for="pay-claim">Link to Claim (optional)</label>
          <select id="pay-claim" bind:value={payClaimId}>
            <option value="">— None —</option>
            {#each claims.filter((c) => c.patient_id === payPatientId) as c}
              <option value={c.id}>
                {c.date_of_service} — {c.insurance_carrier || "No carrier"} ({fmt(claimTotal(c))})
              </option>
            {/each}
          </select>
        </div>

        <div class="form-field">
          <label class="form-label" for="pay-notes">Notes</label>
          <input id="pay-notes" type="text" bind:value={payNotes} placeholder="Optional notes" />
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" onclick={() => (showPaymentModal = false)}>
            Cancel
          </button>
          <button type="submit" class="btn btn-primary">Record Payment</button>
        </div>
      </form>
    </div>
  </div>
{/if}
