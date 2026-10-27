<script lang="ts">
  import { InsuranceService, BillingService, AuditService } from "@bindings/services/index.js";
  import type {
    CountryConfig,
    InsuranceProvider,
    InsuranceFee,
    ProcedureCode,
  } from "@bindings/domain/index.js";
  import { auth } from "../../stores/auth.svelte.js";
  import { m } from "../../paraglide/messages.js";
  import { handleError, errorCode } from "$lib/error.js";
  import { formatCurrency } from "$lib/currency.js";
  import Input from "../../components/ui/Input.svelte";
  let {
    countryMeta = null,
    providerID = $bindable(""),
    code = "",
  }: { countryMeta?: CountryConfig | null; providerID?: string; code?: string } = $props();
  let providers = $state<InsuranceProvider[]>([]),
    fees = $state<InsuranceFee[]>([]),
    procedures = $state<ProcedureCode[]>([]);
  let query = $state(""),
    providerName = $state(""),
    error = $state(""),
    busy = $state(false),
    includeArchived = $state(false);
  let editingProvider = $state(false);
  let amounts = $state<Record<string, number>>({});
  const cc = $derived(countryMeta?.code || "US");
  const provider = $derived(providers.find((p) => p.id === providerID));
  const shownProviders = $derived(
    providers.filter(
      (p) =>
        (includeArchived || !p.archived) &&
        (!query || p.name.toLowerCase().includes(query.toLowerCase()))
    )
  );
  async function load() {
    providers =
      ((await InsuranceService.ListProviders(auth.token))?.filter(
        Boolean
      ) as InsuranceProvider[]) || [];
  }
  async function run(action: () => Promise<unknown>) {
    if (!authorized) return;
    busy = true;
    error = "";
    try {
      await action();
      await load();
    } catch (e) {
      if (errorCode(e) === "unauthorized") verifiedToken = "";
      error = handleError(e, m.insurance_error());
    } finally {
      busy = false;
    }
  }
  let verifiedToken = $state("");
  const authorized = $derived(!!auth.token && verifiedToken === auth.token);
  $effect(() => {
    const token = auth.token;
    verifiedToken = "";
    if (!token) return;
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
  $effect(() => {
    if (!authorized) return;
    void run(async () => {
      procedures =
        ((await BillingService.ListProcedureCodes(cc, ""))?.filter(Boolean) as ProcedureCode[]) ||
        [];
    });
  });
  let loadingFees = $state(false);
  let generation = 0;
  $effect(() => {
    const id = providerID,
      country = cc,
      gen = ++generation;
    fees = [];
    amounts = {};
    loadingFees = !!id;
    if (!authorized || !id) return;
    InsuranceService.ListFees(auth.token, id, country)
      .then((list) => {
        if (gen !== generation) return;
        fees = (list?.filter(Boolean) as InsuranceFee[]) || [];
        loadingFees = false;
        amounts = Object.fromEntries(fees.map((f) => [f.code, f.fee / 100]));
      })
      .catch((e) => {
        if (gen === generation) {
          if (errorCode(e) === "unauthorized") verifiedToken = "";
          loadingFees = false;
          error = handleError(e, m.insurance_error());
        }
      });
  });
  async function saveProvider() {
    await run(async () => {
      const p = await InsuranceService.SaveProvider(auth.token, {
        id: editingProvider ? providerID : "",
        name: providerName,
        archived: (provider?.archived && editingProvider) || false,
      } as InsuranceProvider);
      if (!p) throw new Error(m.insurance_error());
      providerID = p.id;
      providerName = "";
      editingProvider = false;
    });
  }
  async function reloadFees() {
    fees =
      ((await InsuranceService.ListFees(auth.token, providerID, cc))?.filter(
        Boolean
      ) as InsuranceFee[]) || [];
  }
</script>

{#if !authorized}
  <p class="text-sm text-amber-400" role="status">{m.clinic_signin_to_edit()}</p>
{:else}
  <div class="space-y-3" aria-busy={busy}>
    {#if error}<p role="alert" class="text-rose-400">{error}</p>{/if}
    <Input
      aria-label={m.insurance_search()}
      placeholder={m.insurance_search()}
      bind:value={query}
    />
    <label class="text-sm"
      ><input type="checkbox" bind:checked={includeArchived} /> {m.insurance_show_archived()}</label
    >
    <div class="max-h-52 overflow-y-auto space-y-2 border border-slate-700 rounded-xl p-3">
      {#each shownProviders as p (p.id)}
        <div>
          <button
            type="button"
            class="btn btn-secondary text-sm"
            disabled={busy}
            onclick={() => {
              providerID = p.id;
              editingProvider = false;
            }}
            aria-pressed={providerID === p.id}
            >{p.name}{p.archived ? ` (${m.insurance_archived()})` : ""}</button
          >
        </div>
      {/each}
      {#if shownProviders.length === 0}<p>{m.insurance_no_results()}</p>{/if}
    </div>
    <form
      onsubmit={(e) => {
        e.preventDefault();
        void saveProvider();
      }}
      class="flex gap-2"
    >
      <Input
        aria-label={m.insurance_provider_name()}
        placeholder={m.insurance_provider_name()}
        bind:value={providerName}
        required
        maxlength={160}
      />
      <button class="btn btn-secondary" disabled={busy}
        >{editingProvider ? m.common_save() : m.insurance_add_provider()}</button
      >
    </form>
    {#if provider}
      <div class="flex gap-2 items-center flex-wrap">
        <strong>{provider.name}</strong>
        <button
          type="button"
          class="btn btn-secondary text-xs"
          disabled={busy}
          onclick={() => {
            editingProvider = true;
            providerName = provider.name;
          }}>{m.insurance_rename()}</button
        >
        <button
          type="button"
          class="btn btn-secondary text-xs"
          disabled={busy}
          onclick={() =>
            run(() =>
              InsuranceService.SaveProvider(auth.token, {
                ...provider,
                archived: !provider.archived,
              })
            )}>{provider.archived ? m.insurance_restore() : m.insurance_archive()}</button
        >
      </div>
    {/if}
    {#if provider}
      {#if provider.archived}<p role="status">{m.insurance_unavailable()}</p>{/if}
      <div class="max-h-64 overflow-auto">
        <table class="w-full text-sm">
          <thead
            ><tr
              ><th scope="col">{m.billing_th_code()}</th><th scope="col"
                >{m.billing_th_base_fee()}</th
              ><th scope="col">{m.insurance_plan_fee()}</th><th scope="col"
                >{m.patients_th_actions()}</th
              ></tr
            ></thead
          ><tbody>
            {#each procedures.filter((p) => !code || p.code === code) as p (p.code)}
              {@const fee = fees.find((f) => f.code === p.code)}
              <tr
                ><td class="p-2">{p.code} — {p.description}</td><td class="p-2"
                  >{formatCurrency(p.default_fee, countryMeta?.default_currency)}</td
                ><td class="p-2">
                  {#if !fee}<span>{m.insurance_missing_fee()}</span>{/if}
                  <input
                    class="input w-full"
                    type="number"
                    min="0"
                    step="0.01"
                    aria-label={m.insurance_fee_for({ code: p.code })}
                    value={amounts[p.code] ?? ""}
                    disabled={loadingFees || busy}
                    oninput={(e) =>
                      (amounts[p.code] = e.currentTarget.validity.valid
                        ? e.currentTarget.valueAsNumber
                        : NaN)}
                  />
                </td><td class="p-2"
                  ><button
                    type="button"
                    class="btn btn-secondary"
                    disabled={busy ||
                      provider.archived ||
                      amounts[p.code] === undefined ||
                      !Number.isFinite(amounts[p.code]) ||
                      amounts[p.code] < 0}
                    onclick={() =>
                      run(async () => {
                        await InsuranceService.SaveFee(auth.token, {
                          provider_id: providerID,
                          country_code: cc,
                          code: p.code,
                          fee: Math.round(amounts[p.code] * 100),
                        } as InsuranceFee);
                        await reloadFees();
                      })}>{m.common_save()}</button
                  >
                  {#if fee}<button
                      type="button"
                      class="btn btn-secondary"
                      disabled={busy}
                      onclick={() =>
                        run(async () => {
                          await InsuranceService.DeleteFee(auth.token, providerID, cc, p.code);
                          delete amounts[p.code];
                          await reloadFees();
                        })}>{m.insurance_remove_fee()}</button
                    >{/if}
                </td></tr
              >
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
{/if}
