<script lang="ts">
  import { InsuranceService, AuditService } from "@bindings/services/index.js";
  import type { CountryConfig, PricingSource } from "@bindings/domain/index.js";
  import { auth } from "../stores/auth.svelte.js";
  import { m } from "../paraglide/messages.js";
  import { formatCurrency } from "$lib/currency.js";
  import { handleError, errorCode } from "$lib/error.js";
  import Modal from "./ui/Modal.svelte";
  import InsuranceFeeOverridesSection from "../views/billing/InsuranceFeeOverridesSection.svelte";
  let {
    showModal = $bindable(false),
    contextID = "",
    patientID,
    patientName,
    code,
    description,
    countryMeta = null,
    clinicianID = "",
    manualFee = 0,
    preferManual = false,
    onconfirm,
  } = $props<{
    showModal?: boolean;
    contextID?: string;
    patientID: string;
    patientName: string;
    code: string;
    description: string;
    countryMeta?: CountryConfig | null;
    clinicianID?: string;
    manualFee?: number;
    preferManual?: boolean;
    onconfirm: (source: PricingSource) => Promise<boolean | void>;
  }>();
  let source = $state<PricingSource | null>(null),
    reference = $state<PricingSource | null>(null);
  let selectedProvider = $state(""),
    kind = $state("standard"),
    showAll = $state(true),
    busy = $state(false),
    error = $state("");
  let verifiedToken = $state("");
  const authorized = $derived(!!auth.token && verifiedToken === auth.token);
  $effect(() => {
    const token = auth.token;
    verifiedToken = "";
    if (!showModal || !token) return;
    let current = true;
    AuditService.GetSessionUser(token)
      .then((staff) => {
        if (current && staff && auth.token === token) verifiedToken = token;
      })
      .catch(() => {});
    return () => {
      current = false;
    };
  });
  let request = 0;
  const currency = $derived(countryMeta?.default_currency || "USD");
  function input(k: string, plan = "") {
    return {
      kind: k,
      patient_id: patientID,
      country_code: countryMeta?.code || "US",
      code,
      clinician_id: clinicianID,
      provider_id: plan,
      amount: manualFee,
    } as PricingSource;
  }
  async function review(k: string, plan = "") {
    if (!authorized) return;
    const gen = ++request;
    kind = k;
    source = null;
    error = "";
    try {
      const res = await InsuranceService.Preview(auth.token, input(k, plan));
      if (gen === request) source = res;
    } catch (e) {
      if (errorCode(e) === "unauthorized") verifiedToken = "";
      if (gen === request) error = handleError(e, m.insurance_unavailable());
    }
  }
  $effect(() => {
    if (!showModal || !authorized) {
      request++;
      source = null;
      reference = null;
      return;
    }
    const patient = patientID,
      procedure = code;
    void initialize(patient, procedure, contextID);
  });
  async function initialize(patient: string, procedure: string, context: string) {
    const gen = ++request;
    source = null;
    reference = null;
    error = "";
    busy = true;
    selectedProvider = "";
    showAll = true;
    try {
      const standard = await InsuranceService.Preview(auth.token, input("manual"));
      if (gen !== request || patient !== patientID || procedure !== code || context !== contextID)
        return;
      if (!standard) throw new Error(m.insurance_error());
      reference = standard;
      const k =
        preferManual || !standard.has_base || manualFee !== standard.standard_fee
          ? "manual"
          : "standard";
      selectedProvider = "";
      showAll = true;
      await review(k, selectedProvider);
    } catch (e) {
      if (errorCode(e) === "unauthorized") verifiedToken = "";
      if (gen === request) error = handleError(e, m.insurance_error());
    } finally {
      busy = false;
    }
  }
  $effect(() => {
    if (kind === "insurance" && source && selectedProvider !== source.provider_id) source = null;
  });
  async function confirm() {
    if (!authorized || !source || busy) return;
    busy = true;
    error = "";
    try {
      const close = await onconfirm(source);
      if (close !== false) showModal = false;
    } catch (e) {
      if (errorCode(e) === "unauthorized") verifiedToken = "";
      error = handleError(e, m.insurance_error());
      source = null;
    } finally {
      busy = false;
    }
  }
</script>

<Modal
  bind:showModal
  title={m.insurance_confirm_title()}
  maxWidth="max-w-3xl"
  preventDismiss={busy}
>
  {#if !authorized}
    <p class="text-sm text-amber-400" role="status">{m.clinic_signin_to_edit()}</p>
    <div class="flex justify-end mt-4">
      <button type="button" class="btn btn-secondary" onclick={() => (showModal = false)}
        >{m.common_cancel()}</button
      >
    </div>
  {:else}
    <div class="space-y-4">
      <p><strong>{patientName}</strong> · {code} — {description}</p>
      <p>
        {m.billing_th_base_fee()}:
        <strong
          >{reference
            ? reference.has_base
              ? formatCurrency(reference.base_fee, currency)
              : m.insurance_no_base()
            : m.common_loading()}</strong
        >
      </p>
      {#if reference && reference.standard_fee !== reference.base_fee}<p>
          {m.insurance_standard()}: {formatCurrency(reference.standard_fee, currency)}
        </p>{/if}
      {#if showAll}
        <div class="flex gap-2">
          <button
            type="button"
            class="btn btn-secondary"
            disabled={busy}
            onclick={() => review("standard")}>{m.insurance_standard()}</button
          ><button
            type="button"
            class="btn btn-secondary"
            disabled={busy}
            onclick={() => review("manual")}
            >{m.insurance_manual()} ({formatCurrency(manualFee, currency)})</button
          >
        </div>
        <InsuranceFeeOverridesSection {countryMeta} {code} bind:providerID={selectedProvider} />
        <button
          type="button"
          class="btn btn-secondary"
          disabled={busy || !selectedProvider}
          onclick={() => review("insurance", selectedProvider)}>{m.insurance_review_plan()}</button
        >
      {/if}
      {#if error}<p role="alert" class="text-rose-400">{error}</p>{/if}
      {#if source}
        <div class="border border-slate-700 rounded-xl p-3" role="status">
          <p>
            {source.kind === "insurance"
              ? source.provider_name
              : source.kind === "standard"
                ? m.insurance_standard()
                : m.insurance_manual()}
          </p>
          <p>{m.insurance_final()}: <strong>{formatCurrency(source.amount, currency)}</strong></p>
        </div>
      {/if}
      <div class="flex justify-end gap-3">
        <button
          type="button"
          class="btn btn-secondary"
          disabled={busy}
          onclick={() => (showModal = false)}>{m.common_cancel()}</button
        ><button type="button" class="btn btn-primary" disabled={busy || !source} onclick={confirm}
          >{busy ? m.common_loading() : m.common_confirm()}</button
        >
      </div>
    </div>
  {/if}
</Modal>
