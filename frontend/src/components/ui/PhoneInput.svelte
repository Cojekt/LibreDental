<script lang="ts">
  import Input from "./Input.svelte";
  import type { HTMLInputAttributes } from "svelte/elements";
  import { m } from "../../paraglide/messages.js";

  interface Props extends HTMLInputAttributes {
    value?: string;
    error?: boolean;
    fullWidth?: boolean;
  }

  let {
    value = $bindable(""),
    placeholder = "555-019-2834",
    error = false,
    oninput,
    ...restProps
  }: Props = $props();

  let isDirty = $state(false);

  // Phone validation: basic length check for 10 digits after cleaning
  const digitsOnly = $derived(value.replace(/\D/g, ""));
  const isValid = $derived(!isDirty || value === "" || digitsOnly.length >= 10);
  const hasError = $derived(error || !isValid);

  function handleInput(e: Event & { currentTarget: EventTarget & HTMLInputElement }) {
    isDirty = true;

    // Get cursor position to attempt keeping it roughly in place, though Svelte bind can reset it
    const target = e.target as HTMLInputElement;
    let input = target.value.replace(/\D/g, "");

    let formatted = input;
    if (input.length > 6) {
      formatted = `${input.substring(0, 3)}-${input.substring(3, 6)}-${input.substring(6)}`;
    } else if (input.length > 3) {
      formatted = `${input.substring(0, 3)}-${input.substring(3)}`;
    }

    value = formatted;

    if (oninput) {
      oninput(e);
    }
  }
</script>

<Input {...restProps} type="tel" bind:value {placeholder} error={hasError} oninput={handleInput} />
