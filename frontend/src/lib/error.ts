import { m } from "../paraglide/messages.js";

// Translated messages for the error codes the backend attaches to known errors. Must match
// errorCodes in internal/services/error_codes.go.
const errorMessages: Record<string, () => string> = {
  unauthorized: m.error_unauthorized,
  not_found: m.error_not_found,
  conflict: m.error_conflict,
  last_active_provider: m.error_last_active_provider,
  already_initialized: m.error_already_initialized,
  provider_inactive: m.error_provider_inactive,
  incorrect_pin: m.staff_login_incorrect_pin,
  bridges_unavailable: m.integrations_bridges_unavailable,
  bridge_path_required: m.error_bridge_path_required,
  bridge_path_not_absolute: m.error_bridge_path_not_absolute,
  bridge_batch_file: m.error_bridge_batch_file,
  bridge_not_configured: m.error_bridge_not_configured,
  bridge_invalid_args: m.error_bridge_invalid_args,
};

// errorCode returns the backend's code for a failed call, if the error has one.
export function errorCode(err: any): string | undefined {
  const code = err?.cause?.code;
  return typeof code === "string" ? code : undefined;
}

export function handleError(err: any, fallback: string): string {
  const code = errorCode(err);
  if (code && errorMessages[code]) {
    return errorMessages[code]();
  }
  if (err instanceof Error && err.message.trim() !== "") {
    return err.message;
  }
  if (err && typeof err.message === "string" && err.message.trim() !== "") {
    return err.message;
  }
  if (typeof err === "string" && err.trim() !== "") {
    return err;
  }
  return fallback;
}
