<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  interface Props extends HTMLInputAttributes {
    value?: string | number;
    error?: boolean;
    fullWidth?: boolean;
    dateFormat?: string;
  }

  let {
    value = $bindable(""),
    type = "text",
    placeholder = "",
    required = false,
    disabled = false,
    id,
    error = false,
    fullWidth = true,
    dateFormat,
    class: extraClass = "",
    ...restProps
  }: Props = $props();

  let isFocused = $state(false);

  // Compute effective placeholder for date inputs based on country date format if provided
  const effectivePlaceholder = $derived(
    placeholder || (type === "date" && dateFormat ? dateFormat.toLowerCase() : "")
  );

  // Use "text" mode when date value is empty and unfocused so custom dynamic placeholder is displayed
  const effectiveType = $derived(type === "date" && !value && !isFocused ? "text" : type);
</script>

<input
  {id}
  type={effectiveType}
  bind:value
  placeholder={effectivePlaceholder}
  {required}
  {disabled}
  onfocus={(e) => {
    isFocused = true;
    restProps.onfocus?.(e);
  }}
  onblur={(e) => {
    isFocused = false;
    restProps.onblur?.(e);
  }}
  class={`rounded-xl border bg-slate-950/80 px-4 py-2.5 text-sm shadow-sm transition-all focus:outline-none disabled:opacity-50 disabled:bg-slate-950/40 disabled:cursor-not-allowed ${
    fullWidth ? "w-full" : ""
  } ${type === "date" && !value ? "text-slate-400 placeholder:text-slate-500" : "text-white"} ${
    error
      ? "border-rose-500/80 focus:border-rose-500 focus:ring-1 focus:ring-rose-500"
      : "border-slate-700 hover:border-slate-600 focus:border-sky-500 focus:ring-1 focus:ring-sky-500"
  } ${extraClass}`}
  {...restProps}
/>
