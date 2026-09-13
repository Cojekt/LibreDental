<script lang="ts">
  import Modal from "./Modal.svelte";

  import { m } from "../../paraglide/messages.js";
  import { getLocaleVersion } from "../../lib/locale.svelte.js";

  let {
    showModal = $bindable(false),
    title = undefined,
    message = undefined,
    confirmText = undefined,
    cancelText = undefined,
    onConfirm,
  } = $props<{
    showModal: boolean;
    title?: string;
    message?: string;
    confirmText?: string;
    cancelText?: string;
    onConfirm: () => void | boolean | Promise<void | boolean>;
  }>();

  // Defaults must re-derive when the language changes; a plain default
  // parameter is only evaluated once, when the component is first created.
  const displayTitle = $derived.by(() => {
    getLocaleVersion();
    return title ?? m.common_confirm();
  });
  const displayMessage = $derived.by(() => {
    getLocaleVersion();
    return message ?? m.common_confirm_msg();
  });
  const displayConfirmText = $derived.by(() => {
    getLocaleVersion();
    return confirmText ?? m.common_confirm();
  });
  const displayCancelText = $derived.by(() => {
    getLocaleVersion();
    return cancelText ?? m.common_cancel();
  });

  let loading = $state(false);

  async function handleConfirm() {
    loading = true;
    try {
      const res = await onConfirm();
      if (res !== false) {
        showModal = false;
      }
    } catch {
      // Keep modal open if onConfirm throws or fails
    } finally {
      loading = false;
    }
  }
</script>

<Modal bind:showModal preventDismiss={loading} title={displayTitle} maxWidth="max-w-md">
  <div class="py-4 text-slate-300 text-sm">
    {displayMessage}
  </div>

  {#snippet footer()}
    <button
      type="button"
      onclick={() => (showModal = false)}
      class="rounded-xl bg-slate-800 px-5 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-700 transition-colors disabled:opacity-50"
      disabled={loading}
    >
      {displayCancelText}
    </button>
    <button
      type="button"
      onclick={handleConfirm}
      class="rounded-xl bg-rose-600 px-5 py-2 text-sm font-semibold text-white hover:bg-rose-500 transition-colors disabled:opacity-50 flex items-center gap-2"
      disabled={loading}
    >
      {#if loading}
        <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
        ></span>
      {/if}
      {displayConfirmText}
    </button>
  {/snippet}
</Modal>
