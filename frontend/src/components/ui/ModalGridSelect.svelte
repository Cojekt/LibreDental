<script lang="ts" generics="T">
  import type { Snippet } from "svelte";
  import Modal from "./Modal.svelte";

  let {
    value = $bindable(),
    options,
    placeholder = "Select an option...",
    modalTitle = "Select Option",
    buttonContent,
    optionContent,
    buttonClass = "",
    hideChevron = false,
  } = $props<{
    value: T | undefined;
    options: any[];
    placeholder?: string;
    modalTitle?: string;
    buttonContent?: Snippet<[any]>;
    optionContent?: Snippet<[any]>;
    buttonClass?: string;
    hideChevron?: boolean;
  }>();

  let showModal = $state(false);

  let selectedOption = $derived(options.find((o) => o.value === value));
</script>

<button
  type="button"
  onclick={() => (showModal = true)}
  class="flex items-center justify-between w-full p-4 bg-slate-950 border border-slate-700 hover:border-slate-600 rounded-lg text-slate-100 transition-colors {buttonClass}"
>
  {#if selectedOption && buttonContent}
    {@render buttonContent(selectedOption)}
  {:else}
    <span class="text-slate-400">{placeholder}</span>
  {/if}
  {#if !hideChevron}
    <svg
      class="w-5 h-5 text-slate-500 shrink-0"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"
      ></path>
    </svg>
  {/if}
</button>

<Modal bind:showModal title={modalTitle}>
  <div class="grid grid-cols-2 sm:grid-cols-3 gap-4 p-2">
    {#each options as option}
      <button
        type="button"
        onclick={() => {
          value = option.value;
          showModal = false;
        }}
        class="flex flex-col items-center justify-center p-4 gap-3 rounded-xl border transition-all {value ===
        option.value
          ? 'bg-blue-600/20 border-blue-500 text-white shadow-[0_0_15px_rgba(59,130,246,0.3)]'
          : 'bg-slate-800/50 border-slate-700 text-slate-300 hover:bg-slate-700 hover:border-slate-500'}"
      >
        {#if optionContent}
          {@render optionContent(option)}
        {/if}
      </button>
    {/each}
  </div>
</Modal>
