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
  import EmptyState from "../../components/ui/EmptyState.svelte";
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
    providersLoaded = $state(false),
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
    providersLoaded = true;
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
  <div class="space-y-4" aria-busy={busy}>
    {#if error}<p role="alert" class="text-sm text-rose-400">{error}</p>{/if}
    <div
      class="flex flex-wrap items-center gap-4 rounded-xl border border-slate-800 bg-slate-900 p-4"
    >
      <div class="min-w-48 flex-1">
        <Input
          aria-label={m.insurance_search()}
          placeholder={m.insurance_search()}
          bind:value={query}
        />
      </div>
      <label class="flex items-center gap-2 text-sm text-slate-300 cursor-pointer select-none">
        <input type="checkbox" bind:checked={includeArchived} />
        {m.insurance_show_archived()}
      </label>
    </div>

    {#if !providersLoaded}
      {#if !error}<p class="py-6 text-center text-sm text-slate-400" role="status">
          {m.common_loading()}
        </p>{/if}
    {:else if providers.length === 0}
      <EmptyState
        icon="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
        title={m.insurance_empty_title()}
        subtitle={m.insurance_empty_sub()}
      />
    {:else if shownProviders.length === 0}
      <p class="py-6 text-center text-sm text-slate-400">{m.insurance_no_results()}</p>
    {:else}
      <div
        class="flex max-h-52 flex-wrap gap-2 overflow-y-auto rounded-xl border border-slate-800 bg-slate-900/40 p-3"
      >
        {#each shownProviders as p (p.id)}
          <button
            type="button"
            class="rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 {providerID ===
            p.id
              ? 'border-sky-500 bg-sky-500/10 text-sky-300'
              : 'border-slate-700 bg-slate-900 text-slate-200 hover:border-slate-600'}"
            disabled={busy}
            onclick={() => {
              providerID = p.id;
              editingProvider = false;
            }}
            aria-pressed={providerID === p.id}
          >
            {p.name}
            {#if p.archived}
              <span class="ml-1 text-xs text-slate-500">({m.insurance_archived()})</span>
            {/if}
          </button>
        {/each}
      </div>
    {/if}

    <form
      onsubmit={(e) => {
        e.preventDefault();
        void saveProvider();
      }}
      class="flex flex-wrap items-center gap-2"
    >
      <div class="min-w-48 flex-1">
        <Input
          aria-label={m.insurance_provider_name()}
          placeholder={m.insurance_provider_name()}
          bind:value={providerName}
          required
          maxlength={160}
        />
      </div>
      <button class="btn btn-primary shrink-0 whitespace-nowrap" disabled={busy}
        >{editingProvider ? m.common_save() : m.insurance_add_provider()}</button
      >
      {#if editingProvider}
        <button
          type="button"
          class="btn btn-secondary shrink-0"
          disabled={busy}
          onclick={() => {
            editingProvider = false;
            providerName = "";
          }}>{m.common_cancel()}</button
        >
      {/if}
    </form>

    {#if provider}
      <div class="flex flex-wrap items-center gap-2 border-t border-slate-800 pt-4 text-slate-100">
        <strong class="mr-auto text-base">{provider.name}</strong>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          disabled={busy}
          onclick={() => {
            editingProvider = true;
            providerName = provider.name;
          }}>{m.insurance_rename()}</button
        >
        <button
          type="button"
          class="btn btn-secondary btn-sm"
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
      {#if provider.archived}
        <p role="status" class="text-sm text-amber-400">{m.insurance_unavailable()}</p>
      {/if}
      <div class="max-h-96 overflow-auto rounded-xl border border-slate-800 bg-slate-900/40">
        <table class="w-full text-left text-sm text-slate-300">
          <thead
            class="text-xs font-semibold uppercase tracking-wider text-slate-400 [&_th]:sticky [&_th]:top-0 [&_th]:z-10 [&_th]:bg-slate-900 [&_th]:shadow-[inset_0_-1px_0_var(--color-slate-800)]"
          >
            <tr>
              <th scope="col" class="px-4 py-3">{m.billing_th_code()}</th>
              <th scope="col" class="px-4 py-3">{m.billing_th_base_fee()}</th>
              <th scope="col" class="px-4 py-3">{m.insurance_plan_fee()}</th>
              <th scope="col" class="px-4 py-3">{m.patients_th_actions()}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            {#each procedures.filter((p) => !code || p.code === code) as p (p.code)}
              {@const fee = fees.find((f) => f.code === p.code)}
              <tr class="hover:bg-slate-900/50 transition-colors">
                <td class="px-4 py-3">
                  <span class="font-mono font-bold text-sky-400">{p.code}</span>
                  <span class="text-slate-200">{p.description}</span>
                </td>
                <td class="px-4 py-3 font-mono text-slate-400"
                  >{formatCurrency(p.default_fee, countryMeta?.default_currency)}</td
                >
                <td class="px-4 py-3">
                  <Input
                    class="min-w-28"
                    type="number"
                    min="0"
                    step="0.01"
                    placeholder={fee ? "" : m.insurance_missing_fee()}
                    aria-label={m.insurance_fee_for({ code: p.code })}
                    value={amounts[p.code] ?? ""}
                    disabled={loadingFees || busy}
                    oninput={(e) =>
                      (amounts[p.code] = e.currentTarget.validity.valid
                        ? e.currentTarget.valueAsNumber
                        : NaN)}
                  />
                </td>
                <td class="px-4 py-3">
                  <div class="flex flex-wrap gap-2">
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm"
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
                        class="btn btn-secondary btn-sm"
                        disabled={busy}
                        onclick={() =>
                          run(async () => {
                            await InsuranceService.DeleteFee(auth.token, providerID, cc, p.code);
                            delete amounts[p.code];
                            await reloadFees();
                          })}>{m.insurance_remove_fee()}</button
                      >{/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
{/if}
