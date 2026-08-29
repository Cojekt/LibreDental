<script lang="ts">
  import type { Provider } from "@bindings/domain/models.js";
  import Modal from "./ui/Modal.svelte";
  import { auth } from "../stores/auth.svelte.js";
  import { m } from "../paraglide/messages.js";
  import { PracticeConfigService, AuditService } from "@bindings/services/index.js";

  let {
    showModal = $bindable(false),
    providers = [],
    onlogout,
  } = $props<{
    showModal: boolean;
    providers: Provider[];
    onlogout?: () => void;
  }>();

  let selectedProvider = $state<Provider | null>(null);
  let pinInput = $state("");
  let errorMsg = $state("");
  let isLoggingIn = $state(false);

  function selectProvider(p: Provider) {
    selectedProvider = p;
    pinInput = "";
    errorMsg = "";
  }

  $effect(() => {
    if (!showModal) {
      selectedProvider = null;
      pinInput = "";
      errorMsg = "";
    }
  });

  async function handleLogin(e: Event) {
    e.preventDefault();
    if (!selectedProvider || isLoggingIn) return;

    const currentAttemptProvider = selectedProvider;
    const currentPin = pinInput;
    isLoggingIn = true;

    try {
      const token = await AuditService.CreateSession(currentAttemptProvider.id, currentPin);

      if (selectedProvider !== currentAttemptProvider || !showModal) {
        AuditService.DestroySession(token).catch(console.error);
        return;
      }

      auth.commitSession(currentAttemptProvider, token);

      if (selectedProvider !== currentAttemptProvider || !showModal) return;

      showModal = false;
      selectedProvider = null;
      pinInput = "";
      errorMsg = "";
    } catch (err: any) {
      if (selectedProvider !== currentAttemptProvider || !showModal) return;
      if (
        err &&
        err.message &&
        (err.message.includes("incorrect pin") || err.message.includes("pin not set"))
      ) {
        errorMsg = m.staff_login_incorrect_pin();
      } else {
        errorMsg = "Failed to create session";
      }
    } finally {
      isLoggingIn = false;
    }
  }

  function handleLogout() {
    auth.logout();
    showModal = false;
    selectedProvider = null;
    pinInput = "";
    if (onlogout) onlogout();
  }
</script>

<Modal
  bind:showModal
  title={m.staff_login_title()}
  subtitle={m.staff_login_subtitle()}
  icon="🔑"
  maxWidth="max-w-md"
>
  {#if auth.currentStaffId}
    <div class="space-y-4">
      <div
        class="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4 flex items-center justify-between"
      >
        <div>
          <h4 class="text-emerald-400 font-bold text-sm">{m.staff_login_current()}</h4>
          <p class="text-white font-semibold text-lg">
            {auth.currentStaff?.name || auth.currentStaffId}
          </p>
        </div>
      </div>
      <button
        type="button"
        onclick={handleLogout}
        class="w-full rounded-xl border border-rose-500/30 bg-rose-500/20 py-3 font-bold text-rose-400 hover:bg-rose-500/30 transition-colors"
      >
        {m.staff_login_signout()}
      </button>
    </div>
    <div class="my-6 border-t border-slate-800"></div>
    <p class="text-slate-400 text-sm font-semibold mb-3">{m.staff_login_switch()}</p>
  {/if}

  {#if !selectedProvider}
    <div class="grid grid-cols-1 gap-2 max-h-[400px] overflow-y-auto pr-2 custom-scrollbar">
      {#each providers.filter((p: Provider) => p.is_active) as p}
        <button
          type="button"
          onclick={() => selectProvider(p)}
          class="flex items-center gap-3 w-full rounded-xl border border-slate-700 bg-slate-800/50 p-3 hover:bg-slate-700 hover:border-slate-600 transition-colors text-left"
        >
          <div
            class="flex h-10 w-10 items-center justify-center rounded-full text-white font-bold text-sm shadow-md flex-shrink-0"
            style="background-color: {p.color || '#3b82f6'};"
          >
            {p.name.charAt(0)}
          </div>
          <div>
            <h4 class="font-bold text-slate-100">{p.name}</h4>
            <p class="text-xs font-medium text-sky-400 capitalize">{p.role}</p>
          </div>
        </button>
      {/each}
      {#if providers.filter((p: Provider) => p.is_active).length === 0}
        <p class="text-slate-400 text-sm text-center py-4">{m.staff_login_no_providers()}</p>
      {/if}
    </div>
  {:else}
    <form onsubmit={handleLogin} class="space-y-4">
      <div
        class="flex items-center gap-3 mb-6 p-3 rounded-xl bg-slate-800/50 border border-slate-700"
      >
        <button
          type="button"
          onclick={() => (selectedProvider = null)}
          class="text-slate-400 hover:text-white mr-2"
          aria-label="Go back"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="w-5 h-5"
          >
            <line x1="19" y1="12" x2="5" y2="12"></line>
            <polyline points="12 19 5 12 12 5"></polyline>
          </svg>
        </button>
        <div
          class="flex h-10 w-10 items-center justify-center rounded-full text-white font-bold text-sm shadow-md"
          style="background-color: {selectedProvider.color || '#3b82f6'};"
        >
          {selectedProvider.name.charAt(0)}
        </div>
        <div>
          <h4 class="font-bold text-slate-100">{selectedProvider.name}</h4>
          <p class="text-xs font-medium text-sky-400 capitalize">{selectedProvider.role}</p>
        </div>
      </div>

      <div>
        <label for="pin-input" class="block text-sm font-semibold text-slate-300 mb-2"
          >{m.staff_login_enter_pin()}</label
        >
        <input
          id="pin-input"
          type="password"
          bind:value={pinInput}
          disabled={isLoggingIn}
          placeholder="****"
          maxlength="4"
          inputmode="numeric"
          pattern="[0-9]*"
          class="w-full rounded-xl border {errorMsg
            ? 'border-rose-500'
            : 'border-slate-700'} bg-slate-900 px-4 py-3 text-lg tracking-[0.5em] text-center text-white focus:border-sky-500 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
        />
        {#if errorMsg}
          <p class="text-rose-400 text-xs font-semibold mt-2 text-center">{errorMsg}</p>
        {/if}
      </div>

      <button
        type="submit"
        disabled={isLoggingIn}
        class="w-full btn btn-primary py-3 font-bold text-sm shadow-md shadow-sky-500/20 mt-4 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isLoggingIn ? "..." : m.staff_login_signin()}
      </button>
    </form>
  {/if}
</Modal>
