<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title = "",
    subtitle = "",
    icon = "",
    badgeText = "",
    badgeCount = undefined,
    children,
  } = $props<{
    title: string;
    subtitle?: string;
    icon?: string;
    badgeText?: string;
    badgeCount?: number | string;
    children?: Snippet;
  }>();
</script>

<div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-800 pb-3">
  <div class="flex items-center gap-3">
    {#if icon}
      <div
        class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-500 text-white shadow-md shadow-sky-500/20 text-lg"
      >
        {#if icon.length <= 4}
          <span>{icon}</span>
        {:else}
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-5 w-5"
          >
            <path d={icon} stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        {/if}
      </div>
    {/if}

    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2 m-0">
        {title}
        {#if badgeCount !== undefined}
          <span
            class="text-xs font-semibold text-slate-300 bg-slate-800 px-2.5 py-0.5 rounded-full border border-slate-700"
          >
            <span class="text-sky-400 font-bold">{badgeCount}</span>
          </span>
        {/if}
      </h2>
      {#if subtitle}
        <p class="text-xs text-slate-400 mt-0.5 m-0">{subtitle}</p>
      {/if}
    </div>
  </div>

  {#if badgeText && badgeCount === undefined}
    <div
      class="text-xs font-semibold text-slate-300 bg-slate-800/90 px-3.5 py-1.5 rounded-xl border border-slate-700/80 shadow-sm"
    >
      {badgeText}
    </div>
  {/if}

  {#if children}
    <div class="flex items-center gap-3">
      {@render children()}
    </div>
  {/if}
</div>
