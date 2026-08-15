<script lang="ts">
  import Modal from "./Modal.svelte";

  import { m } from "../../paraglide/messages.js";

  let {
    showModal = $bindable(false),
    title = m.common_confirm(),
    message = m.common_confirm_msg(),
    confirmText = m.common_confirm(),
    cancelText = m.common_cancel(),
    onConfirm,
  } = $props<{
    showModal: boolean;
    title?: string;
    message?: string;
    confirmText?: string;
    cancelText?: string;
    onConfirm: () => void | Promise<void>;
  }>();

  let loading = $state(false);

  async function handleConfirm() {
    loading = true;
    try {
      await onConfirm();
      showModal = false;
    } finally {
      loading = false;
    }
  }
</script>

<Modal bind:showModal {title} icon="⚠️" maxWidth="max-w-md">
  <div class="py-4 text-slate-300 text-sm">
    {message}
  </div>

  {#snippet footer()}
    <button
      type="button"
      onclick={() => (showModal = false)}
      class="rounded-xl bg-slate-800 px-5 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-700 transition-colors disabled:opacity-50"
      disabled={loading}
    >
      {cancelText}
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
      {confirmText}
    </button>
  {/snippet}
</Modal>
