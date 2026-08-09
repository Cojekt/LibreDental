<script lang="ts">
  type TabItem = {
    id: string;
    label: string;
    icon?: string;
    count?: number;
  };

  let {
    tabs = [],
    activeTab = $bindable(""),
    onselect,
  } = $props<{
    tabs: TabItem[];
    activeTab: string;
    onselect?: (id: string) => void;
  }>();

  function selectTab(id: string) {
    activeTab = id;
    if (onselect) {
      onselect(id);
    }
  }
</script>

<div class="flex flex-wrap items-center border-b border-slate-800 gap-1">
  {#each tabs as tab}
    <button
      type="button"
      onclick={() => selectTab(tab.id)}
      class={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-colors flex items-center gap-2 ${
        activeTab === tab.id
          ? "border-sky-400 text-sky-400"
          : "border-transparent text-slate-400 hover:text-slate-200"
      }`}
    >
      {#if tab.icon}
        {#if tab.icon.length <= 4}
          <span>{tab.icon}</span>
        {:else}
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-4 w-4"
          >
            <path d={tab.icon} stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        {/if}
      {/if}

      <span>{tab.label}</span>

      {#if tab.count !== undefined}
        <span
          class={`text-xs px-2 py-0.5 rounded-full font-medium ${
            activeTab === tab.id ? "bg-sky-500/20 text-sky-300" : "bg-slate-800 text-slate-400"
          }`}
        >
          {tab.count}
        </span>
      {/if}
    </button>
  {/each}
</div>
