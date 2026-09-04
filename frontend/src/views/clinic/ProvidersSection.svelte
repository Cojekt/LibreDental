<script lang="ts">
  import type { Provider, Timecard } from "@bindings/domain/models.js";
  import { TimecardService } from "@bindings/services/index.js";
  import { untrack } from "svelte";
  import Modal from "../../components/ui/Modal.svelte";
  import ConfirmModal from "../../components/ui/ConfirmModal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import EmailInput from "../../components/ui/EmailInput.svelte";
  import PhoneInput from "../../components/ui/PhoneInput.svelte";
  import EmptyState from "../../components/ui/EmptyState.svelte";
  import { m } from "../../paraglide/messages.js";
  import TimecardsModal from "./TimecardsModal.svelte";

  let {
    providers = [],
    openAddProviderModal,
    openEditProviderModal,
    handleDeleteProvider,
    handleSaveProvider,
    showProviderModal = $bindable(false),
    isEditingProvider = false,
    provName = $bindable(""),
    provRole = $bindable("dentist"),
    provSpecialty = $bindable(""),
    provLicense = $bindable(""),
    provEmail = $bindable(""),
    provPhone = $bindable(""),
    provColor = $bindable("#3b82f6"),
    provPin = $bindable(""),
    provIsActive = $bindable(true),
    provHourlyRate = $bindable(0.0),
  } = $props<{
    providers: Provider[];
    openAddProviderModal: () => void;
    openEditProviderModal: (p: Provider) => void;
    handleDeleteProvider: (id: string) => void;
    handleSaveProvider: (e: Event) => void;
    showProviderModal: boolean;
    isEditingProvider: boolean;
    provName: string;
    provRole: string;
    provSpecialty: string;
    provLicense: string;
    provEmail: string;
    provPhone: string;
    provColor: string;
    provPin: string;
    provIsActive: boolean;
    provHourlyRate: number;
  }>();

  let activeTimecards = $state<Record<string, Timecard | null | undefined>>({});
  let totalOwed = $state<Record<string, number | undefined>>({});
  let inFlightAction = $state<Record<string, "clockIn" | "clockOut" | null>>({});
  let providerGen: Record<string, number> = {};

  let showTimecardsModal = $state(false);
  let selectedProviderId = $state("");
  let selectedProviderName = $state("");

  let searchQuery = $state("");
  let statusFilter = $state("all"); // 'all', 'active', 'inactive'

  let filteredProviders = $derived(
    providers.filter((p: Provider) => {
      if (statusFilter === "active" && !p.is_active) return false;
      if (statusFilter === "inactive" && p.is_active) return false;

      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        return (
          p.name.toLowerCase().includes(q) ||
          (p.role || "").toLowerCase().includes(q) ||
          (p.email || "").toLowerCase().includes(q) ||
          (p.phone || "").toLowerCase().includes(q) ||
          (p.specialty || "").toLowerCase().includes(q) ||
          (p.license_number || "").toLowerCase().includes(q)
        );
      }
      return true;
    })
  );

  $effect(() => {
    const currentProviders = providers;
    untrack(() => {
      loadProviderStates(currentProviders);
    });
  });

  async function loadProviderStates(provs: Provider[] = providers) {
    for (const p of provs) {
      const gen = (providerGen[p.id] = (providerGen[p.id] || 0) + 1);
      try {
        const tc = await TimecardService.GetActiveTimecard(p.id);
        if (providerGen[p.id] === gen) {
          activeTimecards[p.id] = tc;
        }
      } catch (e) {
        if (providerGen[p.id] === gen) {
          activeTimecards[p.id] = undefined;
        }
      }
      try {
        const owed = await TimecardService.GetTotalOwed(p.id);
        if (providerGen[p.id] === gen) {
          totalOwed[p.id] = owed;
        }
      } catch (e) {
        if (providerGen[p.id] === gen) {
          totalOwed[p.id] = undefined;
        }
      }
    }
  }

  async function clockIn(pId: string) {
    if (inFlightAction[pId]) return;
    inFlightAction[pId] = "clockIn";
    const gen = (providerGen[pId] = (providerGen[pId] || 0) + 1);
    try {
      const tc = await TimecardService.ClockIn(pId);
      if (providerGen[pId] === gen) {
        activeTimecards[pId] = tc;
      }
      const prov = providers.find((p: Provider) => p.id === pId);
      await loadProviderStates(prov ? [prov] : providers);
    } catch (e) {
      console.error("Clock In failed", e);
    } finally {
      inFlightAction[pId] = null;
    }
  }

  async function clockOut(pId: string) {
    if (inFlightAction[pId]) return;
    inFlightAction[pId] = "clockOut";
    const gen = (providerGen[pId] = (providerGen[pId] || 0) + 1);
    try {
      await TimecardService.ClockOut(pId);
      if (providerGen[pId] === gen) {
        activeTimecards[pId] = null;
      }
      const prov = providers.find((p: Provider) => p.id === pId);
      await loadProviderStates(prov ? [prov] : providers);
    } catch (e) {
      console.error("Clock Out failed", e);
    } finally {
      inFlightAction[pId] = null;
    }
  }

  let showConfirmPay = $state(false);
  let providerToPay = $state<string | null>(null);

  function promptPay(id: string) {
    providerToPay = id;
    showConfirmPay = true;
  }

  async function executePay() {
    if (!providerToPay) return;
    try {
      await TimecardService.PaySalary(providerToPay);
      await loadProviderStates();
    } catch (e) {
      console.error("Pay Salary failed", e);
    } finally {
      providerToPay = null;
    }
  }
</script>

<div class="space-y-6">
  <div class="flex items-center gap-3 pb-2 border-b border-slate-800">
    <div class="relative w-full max-w-[480px] flex-1">
      <svg
        class="absolute left-3.5 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-slate-400 pointer-events-none z-10"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
      </svg>
      <input
        type="text"
        placeholder={m.prov_search_placeholder()}
        class="box-border w-full rounded-xl border border-slate-700 bg-slate-900 py-2.5 text-sm text-white focus:border-sky-500 focus:outline-none shadow-sm transition-all"
        style="padding-left: 2.75rem; padding-right: 0.75rem;"
        bind:value={searchQuery}
      />
    </div>
    <div
      class="flex items-center gap-1 rounded-xl border border-slate-800 bg-slate-900/90 p-1 shadow-sm select-none"
    >
      <button
        type="button"
        onclick={() => (statusFilter = "all")}
        class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${statusFilter === "all" ? "bg-slate-700 text-slate-200" : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"}`}
        >{m.common_all()}</button
      >
      <button
        type="button"
        onclick={() => (statusFilter = "active")}
        class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${statusFilter === "active" ? "bg-sky-500/20 text-sky-400 border border-sky-500/30 shadow-sm" : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"}`}
        >{m.common_active()}</button
      >
      <button
        type="button"
        onclick={() => (statusFilter = "inactive")}
        class={`px-3.5 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${statusFilter === "inactive" ? "bg-amber-500/20 text-amber-400 border border-amber-500/30 shadow-sm" : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"}`}
        >{m.common_disabled()}</button
      >
    </div>
  </div>

  {#if providers.length === 0}
    <EmptyState title={m.prov_empty_title()} subtitle={m.prov_empty_sub()} />
  {:else if filteredProviders.length === 0}
    <EmptyState title={m.prov_no_results_title()} subtitle={m.prov_no_results_desc()} />
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each filteredProviders as p}
        <div
          class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 relative group hover:border-slate-700 transition-colors"
        >
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-3">
              <div
                class="h-10 w-10 rounded-full flex items-center justify-center text-white font-bold text-sm shadow-md"
                style="background-color: {p.color || '#3b82f6'};"
              >
                {p.name.charAt(0)}
              </div>
              <div>
                <h4 class="text-sm font-bold text-slate-100">{p.name}</h4>
                <p class="text-xs text-sky-400 capitalize font-medium">
                  {p.role}
                  {p.specialty ? `• ${p.specialty}` : ""}
                </p>
              </div>
            </div>
            <span
              class={`px-2 py-0.5 text-[10px] font-bold rounded-full uppercase ${p.is_active ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" : "bg-slate-800 text-slate-500"}`}
              >{p.is_active ? m.common_active() : m.common_inactive()}
            </span>
          </div>

          {#if p.license_number || p.email || p.phone || p.hourly_rate || p.pin}
            <div class="text-xs text-slate-400 space-y-1 pt-2 border-t border-slate-800">
              {#if p.hourly_rate}
                <div class="font-medium text-emerald-400">
                  {m.prov_wage_prefix()} ${(p.hourly_rate / 100).toFixed(2)}{m.prov_wage_suffix()}
                </div>
              {/if}
              {#if p.license_number}
                <div>
                  {m.prov_license_prefix()}
                  <span class="text-slate-300 font-mono">{p.license_number}</span>
                </div>
              {/if}
              {#if p.email}
                <div>{p.email}</div>
              {/if}
              {#if p.phone}
                <div>{p.phone}</div>
              {/if}
              {#if p.pin}
                <div class="flex items-center gap-2">
                  <span>{m.prov_pin_display_label()}</span>
                  <span class="text-slate-500 font-mono tracking-widest">****</span>
                </div>
              {/if}
            </div>
          {/if}

          <div class="flex items-center justify-between pt-2 border-t border-slate-800/60 text-xs">
            <div>
              {#if inFlightAction[p.id] === "clockOut"}
                <button
                  type="button"
                  disabled
                  class="rounded bg-rose-500/20 text-rose-400 px-3 py-1 font-semibold transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {m.prov_clocking_out()}
                </button>
              {:else if inFlightAction[p.id] === "clockIn"}
                <button
                  type="button"
                  disabled
                  class="rounded bg-emerald-500/20 text-emerald-400 px-3 py-1 font-semibold transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {m.prov_clocking_in()}
                </button>
              {:else if activeTimecards[p.id] === undefined}
                <span class="text-slate-500 font-semibold italic">{m.common_loading()}</span>
              {:else if activeTimecards[p.id]}
                <button
                  type="button"
                  onclick={() => clockOut(p.id)}
                  class="rounded bg-rose-500/20 text-rose-400 px-3 py-1 font-semibold hover:bg-rose-500/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {m.prov_clock_out()}
                </button>
              {:else}
                <button
                  type="button"
                  onclick={() => clockIn(p.id)}
                  class="rounded bg-emerald-500/20 text-emerald-400 px-3 py-1 font-semibold hover:bg-emerald-500/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {m.prov_clock_in()}
                </button>
              {/if}
            </div>
            <div class="flex items-center gap-2">
              <button
                type="button"
                onclick={() => openEditProviderModal(p)}
                class="text-sky-400 hover:text-sky-300 font-semibold"
              >
                {m.patients_btn_edit()}
              </button>
              <button
                type="button"
                onclick={() => handleDeleteProvider(p.id)}
                class="text-rose-400 hover:text-rose-300 font-semibold"
              >
                {m.common_disable()}
              </button>
            </div>
          </div>

          <div class="bg-slate-800/40 rounded-lg p-3 mt-2 border border-slate-700/50">
            <div class="flex items-center justify-between">
              <div class="text-slate-300 text-xs font-semibold">
                {m.prov_total_owed()}
                {#if totalOwed[p.id] === undefined}
                  <span class="text-slate-500 text-sm ml-1">...</span>
                {:else}
                  <span class="text-emerald-400 text-sm ml-1"
                    >${((totalOwed[p.id] || 0) / 100).toFixed(2)}</span
                  >
                {/if}
              </div>
              <button
                type="button"
                onclick={() => promptPay(p.id)}
                class="bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 px-3 py-1 rounded text-xs font-bold transition-colors border border-emerald-500/30"
              >
                {m.prov_pay_salary()}
              </button>
            </div>
            <div class="mt-3 flex justify-end">
              <button
                type="button"
                onclick={() => {
                  selectedProviderId = p.id;
                  selectedProviderName = p.name;
                  showTimecardsModal = true;
                }}
                class="text-sky-400 hover:text-sky-300 text-xs font-semibold flex items-center gap-1"
              >
                {m.prov_view_timecards()}
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- PROVIDER MODAL -->
<Modal
  bind:showModal={showProviderModal}
  title={isEditingProvider ? m.prov_modal_subtitle() : m.prov_add_btn()}
  subtitle={m.prov_modal_subtitle()}
  maxWidth="max-w-md"
>
  <form onsubmit={handleSaveProvider} class="space-y-4">
    <FormField label={m.prov_name_label()} forId="prov-name" required>
      <Input
        id="prov-name"
        type="text"
        bind:value={provName}
        required
        placeholder={m.prov_name_placeholder()}
      />
    </FormField>

    <div class="grid grid-cols-2 gap-3">
      <FormField label={m.prov_role_label()} forId="prov-role">
        <select
          id="prov-role"
          bind:value={provRole}
          class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
        >
          <option value="dentist">{m.prov_role_dentist()}</option>
          <option value="hygienist">{m.prov_role_hygienist()}</option>
          <option value="assistant">{m.prov_role_assistant()}</option>
          <option value="staff">{m.prov_role_staff()}</option>
        </select>
      </FormField>

      <FormField label={m.prov_specialty_label()} forId="prov-specialty">
        <Input
          id="prov-specialty"
          type="text"
          bind:value={provSpecialty}
          placeholder={m.prov_specialty_placeholder()}
        />
      </FormField>
    </div>

    <FormField label={m.prov_license_label()} forId="prov-license">
      <Input
        id="prov-license"
        type="text"
        bind:value={provLicense}
        placeholder={m.prov_license_placeholder()}
      />
    </FormField>

    <div class="grid grid-cols-2 gap-3">
      <FormField label={m.prov_email_label()} forId="prov-email">
        <EmailInput
          id="prov-email"
          bind:value={provEmail}
          placeholder={m.prov_email_placeholder()}
        />
      </FormField>

      <FormField label={m.prov_phone_label()} forId="prov-phone">
        <PhoneInput id="prov-phone" bind:value={provPhone} placeholder="555-019-2834" />
      </FormField>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <FormField label={m.prov_hourly_rate_label()} forId="prov-hourly">
        <Input
          id="prov-hourly"
          type="number"
          min="0"
          step="0.01"
          bind:value={provHourlyRate}
          placeholder={m.prov_hourly_rate_placeholder()}
        />
      </FormField>

      <FormField label={m.prov_pin_label()} forId="prov-pin" required>
        <Input
          id="prov-pin"
          type="password"
          bind:value={provPin}
          placeholder={m.prov_pin_placeholder()}
          pattern="[0-9]*"
          inputmode="numeric"
          maxlength={4}
          minlength={4}
          required
        />
      </FormField>
    </div>

    <div class="flex items-center justify-between pt-2">
      <div>
        <label for="prov-color" class="block text-xs font-semibold text-slate-300 mb-1"
          >{m.prov_color_badge_label()}</label
        >
        <input
          id="prov-color"
          type="color"
          bind:value={provColor}
          class="h-9 w-16 cursor-pointer rounded border border-slate-700 bg-slate-950 p-1"
        />
      </div>

      <div class="flex items-center gap-2 pt-4">
        <input type="checkbox" id="prov-active" bind:checked={provIsActive} />
        <label for="prov-active" class="text-xs font-semibold text-slate-300 cursor-pointer"
          >{m.prov_active_label()}</label
        >
      </div>
    </div>

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        onclick={() => (showProviderModal = false)}
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
      >
        {m.common_cancel()}
      </button>
      <button type="submit" class="btn btn-primary text-xs px-5 py-2 cursor-pointer">
        {m.prov_save_btn()}
      </button>
    </div>
  </form>
</Modal>

<TimecardsModal
  bind:showModal={showTimecardsModal}
  providerId={selectedProviderId}
  providerName={selectedProviderName}
  onrefresh={loadProviderStates}
/>

<ConfirmModal
  bind:showModal={showConfirmPay}
  title={m.prov_pay_salary()}
  message={m.prov_confirm_pay_salary()}
  confirmText={m.billing_btn_record_payment()}
  onConfirm={executePay}
/>
