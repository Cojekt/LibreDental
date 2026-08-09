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
  import Modal from "../../components/ui/Modal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import StatusBadge from "../../components/ui/StatusBadge.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";

  let { patients = [], countryMeta = null } = $props<{
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

<div class="space-y-6">
  <div class="flex justify-end">
    <button
      type="button"
      class="btn btn-primary text-xs flex items-center gap-1.5"
      onclick={openNewPayment}
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
        <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {m.billing_btn_record_payment()}
    </button>
  </div>

  <div class="space-y-4">
    <div class="max-w-xs space-y-1.5">
      <FormField label="Select Patient for Ledger & Balance" forId="balance-patient">
        <select
          id="balance-patient"
          bind:value={balancePatientId}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3.5 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          <option value="">— Select patient —</option>
          {#each patients as p}
            <option value={p.id}>{p.first_name} {p.last_name}</option>
          {/each}
        </select>
      </FormField>
    </div>

    {#if patientBalance && balancePatientId}
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="rounded-xl border border-slate-800 bg-slate-900/60 p-4 space-y-1">
          <div class="text-xs font-semibold uppercase tracking-wider text-slate-400">
            Total Billed
          </div>
          <div class="text-2xl font-extrabold text-slate-300 font-mono">
            {fmt(patientBalance.total_billed)}
          </div>
        </div>
        <div class="rounded-xl border border-slate-800 bg-slate-900/60 p-4 space-y-1">
          <div class="text-xs font-semibold uppercase tracking-wider text-slate-400">
            Total Paid
          </div>
          <div class="text-2xl font-extrabold text-emerald-400 font-mono">
            {fmt(patientBalance.total_paid)}
          </div>
        </div>
        <div
          class={`rounded-xl border p-4 space-y-1 ${patientBalance.outstanding > 0 ? "border-amber-500/40 bg-amber-950/20" : "border-slate-800 bg-slate-900/60"}`}
        >
          <div class="text-xs font-semibold uppercase tracking-wider text-slate-400">
            Outstanding Balance
          </div>
          <div
            class={`text-2xl font-extrabold font-mono ${patientBalance.outstanding > 0 ? "text-amber-400" : "text-emerald-400"}`}
          >
            {fmt(patientBalance.outstanding)}
          </div>
        </div>
      </div>
    {:else if balancePatientId}
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="rounded-xl border border-slate-800 bg-slate-900/60 p-4 space-y-1">
          <div class="text-xs font-semibold uppercase tracking-wider text-slate-400">
            Outstanding Balance
          </div>
          <div class="text-2xl font-extrabold text-emerald-400 font-mono">$0.00</div>
        </div>
      </div>
    {/if}
  </div>

  <div class="space-y-3 pt-2">
    <div class="flex items-center justify-between border-b border-slate-800 pb-3">
      <h3 class="text-base font-bold text-slate-100 m-0">Payment Remittances & Log</h3>
      {#if loadingPayments}
        <span class="text-slate-400 text-xs font-medium">Loading…</span>
      {/if}
    </div>

    {#if payments.length === 0 && !loadingPayments}
      <EmptyState title={`No payments recorded${balancePatientId ? " for this patient" : ""}.`} />
    {:else}
      <div class="space-y-2">
        {#each payments as pay (pay.id)}
          <div
            class="flex items-center justify-between p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 hover:border-slate-700 transition-colors"
          >
            <div class="flex items-center gap-3">
              <span class="text-base font-bold text-slate-100 font-mono">{fmt(pay.amount)}</span>
              <StatusBadge variant={pay.method} label={pay.method.replace("_", " ")} />
              {#if pay.claim_id}
                <span
                  class="text-xs font-mono text-slate-500 bg-slate-800 px-2 py-0.5 rounded border border-slate-700"
                  >Claim #{pay.claim_id.slice(-6)}</span
                >
              {/if}
            </div>
            <div class="flex items-center gap-4 text-xs">
              <span class="text-slate-300 font-medium">{patientName(pay.patient_id)}</span>
              <span class="text-slate-500 font-mono">{pay.date}</span>
              {#if pay.notes}
                <span class="text-slate-400 italic max-w-xs truncate">{pay.notes}</span>
              {/if}
              <button
                class="p-1.5 text-slate-400 hover:text-rose-400 rounded-lg hover:bg-slate-800 transition-colors"
                onclick={() => deletePayment(pay.id)}
                title="Delete payment"
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
        {/each}
      </div>
    {/if}
  </div>
</div>

<!-- PAYMENT MODAL -->
<Modal
  bind:showModal={showPaymentModal}
  title="Record Payment"
  subtitle="Record a patient payment, copay, or insurance remittance"
  icon="💵"
  maxWidth="max-w-md"
>
  <form onsubmit={savePayment} class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <FormField label="Patient" forId="pay-patient" required>
        <select
          id="pay-patient"
          bind:value={payPatientId}
          required
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          {#each patients as p}
            <option value={p.id}>{p.first_name} {p.last_name}</option>
          {/each}
        </select>
      </FormField>
      <FormField label="Date" forId="pay-date" required>
        <Input id="pay-date" type="date" bind:value={payDate} required />
      </FormField>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <FormField label="Amount" forId="pay-amount" required>
        <Input
          id="pay-amount"
          type="number"
          bind:value={payAmount}
          step="0.01"
          min="0.01"
          placeholder="0.00"
          required
        />
      </FormField>
      <FormField label="Method" forId="pay-method" required>
        <select
          id="pay-method"
          bind:value={payMethod}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          {#each PAYMENT_METHODS as m}
            <option value={m}>{m.replace("_", " ").replace(/\b\w/g, (c) => c.toUpperCase())}</option
            >
          {/each}
        </select>
      </FormField>
    </div>

    <FormField label="Link to Claim (optional)" forId="pay-claim">
      <select
        id="pay-claim"
        bind:value={payClaimId}
        class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
      >
        <option value="">— None —</option>
        {#each claims.filter((c) => c.patient_id === payPatientId) as c}
          <option value={c.id}>
            {c.date_of_service} — {c.insurance_carrier || "No carrier"} ({fmt(claimTotal(c))})
          </option>
        {/each}
      </select>
    </FormField>

    <FormField label="Notes" forId="pay-notes">
      <Input id="pay-notes" type="text" bind:value={payNotes} placeholder="Optional notes" />
    </FormField>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
        onclick={() => (showPaymentModal = false)}
      >
        Cancel
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        Record Payment
      </button>
    </div>
  </form>
</Modal>
