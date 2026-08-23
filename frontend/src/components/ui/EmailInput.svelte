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
    placeholder = "jane.smith@example.com",
    error = false,
    oninput,
    ...restProps
  }: Props = $props();

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  let isDirty = $state(false);

  // Validate only after user starts typing
  const isValid = $derived(!isDirty || value === "" || emailRegex.test(value));
  const hasError = $derived(error || !isValid);

  function handleInput(e: Event & { currentTarget: EventTarget & HTMLInputElement }) {
    isDirty = true;
    if (oninput) {
      oninput(e);
    }
  }
</script>

<Input
  {...restProps}
  type="email"
  bind:value
  pattern={hasError ? emailRegex.source : undefined}
  title={hasError ? m.validation_email_invalid() : undefined}
  {placeholder}
  error={hasError}
  oninput={handleInput}
/>
