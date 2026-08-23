<script lang="ts">
  import Input from "./Input.svelte";
  import type { HTMLInputAttributes } from "svelte/elements";
  import type { CountryConfig } from "@bindings/domain/models.js";
  import { m } from "../../paraglide/messages.js";

  interface Props extends HTMLInputAttributes {
    value?: string;
    error?: boolean;
    fullWidth?: boolean;
    countryMeta?: CountryConfig | null;
  }

  let {
    value = $bindable(""),
    placeholder = "000-00-0000",
    error = false,
    countryMeta = null,
    oninput,
    ...restProps
  }: Props = $props();

  let isDirty = $state(false);

  const isSSN = $derived(countryMeta?.national_id_type === "ssn" || !countryMeta);

  // SSN validation: 9 digits
  const digitsOnly = $derived(value.replace(/\D/g, ""));
  const isValid = $derived(!isDirty || value === "" || !isSSN || digitsOnly.length === 9);
  const hasError = $derived(error || !isValid);

  const idPattern = "^[0-9]{3}-[0-9]{2}-[0-9]{4}$";

  function handleInput(e: Event & { currentTarget: EventTarget & HTMLInputElement }) {
    isDirty = true;

    const target = e.target as HTMLInputElement;

    if (isSSN) {
      let input = target.value.replace(/\D/g, "").substring(0, 9);

      let formatted = input;
      if (input.length > 5) {
        formatted = `${input.substring(0, 3)}-${input.substring(3, 5)}-${input.substring(5)}`;
      } else if (input.length > 3) {
        formatted = `${input.substring(0, 3)}-${input.substring(3)}`;
      }

      value = formatted;
    } else {
      value = target.value;
    }

    if (oninput) {
      oninput(e);
    }
  }
</script>

<Input
  {...restProps}
  type="text"
  bind:value
  pattern={isSSN ? idPattern : undefined}
  title={isSSN ? m.validation_ssn_length() : undefined}
  {placeholder}
  error={hasError}
  oninput={handleInput}
/>
