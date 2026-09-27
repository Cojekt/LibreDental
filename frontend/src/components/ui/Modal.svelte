<script module lang="ts">
  // Open modals, innermost last, so Escape only dismisses the one on top (e.g. a confirm
  // dialog opened over an edit form).
  const openModals: symbol[] = [];
</script>

<script lang="ts">
  import { untrack, type Snippet } from "svelte";
  import { m } from "../../paraglide/messages.js";

  let {
    showModal = $bindable(false),
    title,
    subtitle,
    icon,
    maxWidth = "max-w-xl",
    preventDismiss = false,
    children,
    footer,
  } = $props<{
    showModal: boolean;
    title?: string;
    subtitle?: string;
    icon?: string;
    maxWidth?: string;
    preventDismiss?: boolean;
    children?: Snippet;
    footer?: Snippet;
  }>();

  const modalId = Symbol("modal");
  let dialogEl = $state<HTMLDivElement | null>(null);
  // Only a press that starts on the backdrop may dismiss: selecting text in a field and
  // releasing over the backdrop must not throw away the form.
  let pressStartedOnBackdrop = false;

  $effect(() => {
    if (!showModal) return;
    openModals.push(modalId);
    // Untracked so re-binding the element never re-runs this and reorders the stack.
    untrack(() => {
      if (dialogEl && !dialogEl.contains(document.activeElement)) {
        dialogEl.focus();
      }
    });
    return () => {
      const idx = openModals.indexOf(modalId);
      if (idx !== -1) openModals.splice(idx, 1);
    };
  });

  function handleBackdropPointerDown(e: PointerEvent) {
    pressStartedOnBackdrop = e.target === e.currentTarget;
  }

  function handleBackdropClick(e: MouseEvent) {
    const startedOnBackdrop = pressStartedOnBackdrop;
    pressStartedOnBackdrop = false;
    if (preventDismiss || !startedOnBackdrop || e.target !== e.currentTarget) return;
    showModal = false;
  }

  function handleWindowKeydown(e: KeyboardEvent) {
    if (!showModal || e.key !== "Escape" || e.defaultPrevented) return;
    if (openModals[openModals.length - 1] !== modalId) return;
    e.preventDefault();
    if (preventDismiss) return;
    showModal = false;
  }
</script>

<svelte:window onkeydown={handleWindowKeydown} />

{#if showModal}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4 backdrop-blur-sm animate-fadeIn"
    onpointerdown={handleBackdropPointerDown}
    onclick={handleBackdropClick}
    role="presentation"
  >
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
    <div
      class={`w-full ${maxWidth} rounded-2xl border border-slate-700 bg-slate-900 p-6 shadow-2xl overflow-y-auto max-h-[90vh] text-slate-100 dark-modal-box`}
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      {#snippet closeButton()}
        <button
          type="button"
          onclick={() => !preventDismiss && (showModal = false)}
          disabled={preventDismiss}
          class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white transition-colors cursor-pointer border-none bg-transparent text-lg font-bold disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label={m.common_close()}
        >
          ✕
        </button>
      {/snippet}

      {#if title || icon}
        <div class="flex items-center justify-between border-b border-slate-800 pb-4 mb-5">
          <div class="flex items-center gap-3">
            {#if icon}
              <div
                class="flex h-9 w-9 items-center justify-center rounded-xl bg-sky-500/20 text-sky-400 border border-sky-500/30 font-bold text-base shrink-0"
              >
                {icon}
              </div>
            {/if}
            <div>
              {#if title}
                <h2 class="m-0 text-lg font-bold text-white tracking-tight">{title}</h2>
              {/if}
              {#if subtitle}
                <p class="m-0 text-xs text-slate-400 mt-0.5">{subtitle}</p>
              {/if}
            </div>
          </div>
          {@render closeButton()}
        </div>
      {:else}
        <div class="flex justify-end mb-2">
          {@render closeButton()}
        </div>
      {/if}

      {#if children}
        {@render children()}
      {/if}

      {#if footer}
        <div class="flex items-center justify-end gap-3 border-t border-slate-800 pt-4 mt-6">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}
