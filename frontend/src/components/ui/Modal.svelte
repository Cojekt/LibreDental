<script lang="ts">
  import type { Snippet } from "svelte";

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

  function handleBackdropClick() {
    if (preventDismiss) return;
    showModal = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (preventDismiss) return;
    if (e.key === "Escape") {
      showModal = false;
    }
  }
</script>

{#if showModal}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4 backdrop-blur-sm animate-fadeIn"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
    role="presentation"
  >
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
    <div
      class={`w-full ${maxWidth} rounded-2xl border border-slate-700 bg-slate-900 p-6 shadow-2xl overflow-y-auto max-h-[90vh] text-slate-100 dark-modal-box`}
      onclick={(e) => e.stopPropagation()}
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
          aria-label="Close modal"
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
