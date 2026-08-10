<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title = "No data available",
    subtitle = "",
    icon = "",
    children,
  } = $props<{
    title?: string;
    subtitle?: string;
    icon?: string;
    children?: Snippet;
  }>();
</script>

<div
  class="flex flex-col items-center justify-center py-16 px-4 text-center border border-dashed border-slate-800 rounded-2xl bg-slate-900/40"
>
  {#if icon}
    {#if icon.startsWith("<svg") || icon.startsWith("M") || icon.includes("path")}
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
        class="h-10 w-10 text-slate-500 mb-3"
      >
        <path d={icon} stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    {:else}
      <span class="text-3xl mb-3">{icon}</span>
    {/if}
  {/if}

  <p class="text-base font-semibold text-slate-300 m-0">{title}</p>
  {#if subtitle}
    <p class="text-xs text-slate-500 mt-1 m-0 max-w-sm">{subtitle}</p>
  {/if}

  {#if children}
    <div class="mt-4 flex items-center justify-center gap-3">
      {@render children()}
    </div>
  {/if}
</div>
