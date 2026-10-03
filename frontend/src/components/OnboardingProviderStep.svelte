<script lang="ts">
  import type { Provider } from "@bindings/domain/models.js";
  import { PracticeConfigService, AuditService } from "@bindings/services/index.js";
  import { auth } from "../stores/auth.svelte.js";
  import { m } from "../paraglide/messages.js";
  import FormField from "./ui/FormField.svelte";
  import Input from "./ui/Input.svelte";

  let { onback, oncomplete, onalreadyinitialized } = $props<{
    onback?: () => void;
    oncomplete: () => void;
    onalreadyinitialized: () => void;
  }>();

  let name = $state("");
  let role = $state("dentist");
  let licenseNumber = $state("");
  let color = $state("#3b82f6");
  let pin = $state("");
  let pinConfirm = $state("");
  let errorMsg = $state("");
  let isSubmitting = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (isSubmitting) return;
    errorMsg = "";

    if (!name.trim()) {
      errorMsg = m.onboarding_provider_error_name_required();
      return;
    }

    if (!/^[0-9]{4}$/.test(pin)) {
      errorMsg = m.onboarding_provider_error_pin_format();
      return;
    }
    if (pin !== pinConfirm) {
      errorMsg = m.onboarding_provider_error_pin_mismatch();
      return;
    }

    isSubmitting = true;
    try {
      const token = await PracticeConfigService.CreateInitialProvider({
        name: name.trim(),
        role,
        license_number: licenseNumber.trim(),
        color,
        pin,
      } as unknown as Provider);

      const provider = await AuditService.GetSessionUser(token);
      if (!provider) throw new Error("session not found for initial provider");
      auth.commitSession(provider, token);
      oncomplete();
    } catch (err: any) {
      // Another client on the LAN finished onboarding first; that account must be used.
      if (err?.message?.includes("clinic already has an active provider")) {
        onalreadyinitialized();
        return;
      }
      console.error("Failed to create initial provider:", err);
      errorMsg = m.onboarding_provider_error_failed();
    } finally {
      isSubmitting = false;
    }
  }
</script>

<h3 class="m-0 mb-2 text-xl font-bold text-slate-100">{m.onboarding_provider_title()}</h3>
<p class="text-base text-slate-300 leading-relaxed mb-8">{m.onboarding_provider_body()}</p>

<form onsubmit={handleSubmit} class="flex flex-col gap-5">
  <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
    <FormField label={m.prov_name_label()} forId="onboard-prov-name" required>
      <Input
        id="onboard-prov-name"
        type="text"
        bind:value={name}
        required
        disabled={isSubmitting}
        placeholder={m.prov_name_placeholder()}
      />
    </FormField>

    <FormField label={m.prov_role_label()} forId="onboard-prov-role">
      <select
        id="onboard-prov-role"
        bind:value={role}
        disabled={isSubmitting}
        class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
      >
        <option value="dentist">{m.prov_role_dentist()}</option>
        <option value="hygienist">{m.prov_role_hygienist()}</option>
        <option value="assistant">{m.prov_role_assistant()}</option>
        <option value="staff">{m.prov_role_staff()}</option>
      </select>
    </FormField>

    <FormField label={m.prov_license_label()} forId="onboard-prov-license">
      <Input
        id="onboard-prov-license"
        type="text"
        bind:value={licenseNumber}
        disabled={isSubmitting}
        placeholder={m.prov_license_placeholder()}
      />
    </FormField>

    <FormField label={m.prov_color_badge_label()} forId="onboard-prov-color">
      <input
        id="onboard-prov-color"
        type="color"
        bind:value={color}
        disabled={isSubmitting}
        class="h-9 w-16 cursor-pointer rounded border border-slate-700 bg-slate-950 p-1"
      />
    </FormField>

    <FormField
      label={m.prov_pin_label()}
      forId="onboard-prov-pin"
      required
      helpText={m.onboarding_provider_pin_hint()}
    >
      <Input
        id="onboard-prov-pin"
        type="password"
        bind:value={pin}
        required
        disabled={isSubmitting}
        placeholder={m.prov_pin_placeholder()}
        pattern="[0-9]*"
        inputmode="numeric"
        maxlength={4}
        minlength={4}
        autocomplete="new-password"
      />
    </FormField>

    <FormField
      label={m.onboarding_provider_pin_confirm_label()}
      forId="onboard-prov-pin-confirm"
      required
    >
      <Input
        id="onboard-prov-pin-confirm"
        type="password"
        bind:value={pinConfirm}
        required
        disabled={isSubmitting}
        pattern="[0-9]*"
        inputmode="numeric"
        maxlength={4}
        minlength={4}
        autocomplete="new-password"
      />
    </FormField>
  </div>

  {#if errorMsg}
    <p class="m-0 text-sm font-semibold text-rose-400" role="alert">{errorMsg}</p>
  {/if}

  <div class="flex flex-col-reverse md:flex-row md:justify-between gap-3 mt-4">
    {#if onback}
      <button
        type="button"
        onclick={onback}
        disabled={isSubmitting}
        class="px-6 py-3 rounded-lg text-base font-semibold text-slate-300 hover:text-white hover:bg-slate-800 transition-colors disabled:opacity-50 cursor-pointer"
      >
        {m.common_back()}
      </button>
    {:else}
      <span></span>
    {/if}
    <button
      type="submit"
      disabled={isSubmitting}
      class="w-full md:w-auto px-10 rounded-lg bg-blue-600 hover:bg-blue-500 py-3 text-base font-semibold text-white shadow-lg shadow-blue-600/20 transition-all disabled:opacity-50 cursor-pointer"
    >
      {isSubmitting ? m.onboarding_provider_submitting() : m.onboarding_submit()}
    </button>
  </div>
</form>
