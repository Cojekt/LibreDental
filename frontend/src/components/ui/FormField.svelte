<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    label,
    forId,
    required = false,
    error,
    helpText,
    labelClass = "text-slate-300",
    children,
  } = $props<{
    label?: string;
    forId?: string;
    required?: boolean;
    error?: string;
    helpText?: string;
    labelClass?: string;
    children?: Snippet;
  }>();
</script>

<div class="flex flex-col gap-1.5 w-full">
  {#if label}
    <label for={forId} class={`text-xs font-semibold ${labelClass} flex items-center gap-1`}>
      <span>{label}</span>
      {#if required}
        <span class="text-sky-400 font-bold text-xs" title="Required field">*</span>
      {/if}
    </label>
  {/if}

  {#if children}
    {@render children()}
  {/if}

  {#if error}
    <p class="text-[11px] text-rose-400 m-0">⚠️ {error}</p>
  {:else if helpText}
    <p class="text-[11px] text-slate-400 m-0">{helpText}</p>
  {/if}
</div>
