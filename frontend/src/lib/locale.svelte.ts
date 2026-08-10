/**
 * locale.svelte.ts — Reactive Paraglide locale wrapper for LibreDental.
 *
 * Because LibreDental runs as a Wails desktop SPA with no URL-based routing,
 * we use `setLocale(tag, { reload: false })` and drive re-renders manually.
 *
 * Svelte 5 rule: exported $state must not be reassigned from outside.
 * We expose the state via a getter function `getLocaleVersion()` and mutation
 * through `applyLocale()` — which keeps the state change internal.
 *
 * Usage in components:
 *   import { m } from '../paraglide/messages.js';
 *   import { getLocaleVersion } from '$lib/locale.svelte.js';
 *
 *   // Read the version to create a reactive dependency:
 *   <span>{getLocaleVersion(), m.settings_title()}</span>
 */

import { setLocale, getLocale, locales } from "../paraglide/runtime.js";
import { SystemSettingsService } from "@bindings/services/index.js";

// Internal reactive counter — not directly exported (Svelte 5 module state rule).
let _localeVersion = $state(0);

/** Returns the current locale version counter. Subscribe to this in expressions to force re-render. */
export function getLocaleVersion(): number {
  return _localeVersion;
}

/**
 * Apply a locale tag immediately without a page reload.
 * Increments the reactive version counter so components re-evaluate their `m.*()` calls.
 */
export function applyLocale(tag: string) {
  setLocale(tag as any, { reload: false });
  _localeVersion++;
}

/**
 * Resolve the effective language preference via Go backend and activate it.
 * Called during app mount or settings updates.
 */
export async function initLocale(): Promise<void> {
  try {
    const effective = await SystemSettingsService.GetEffectiveLocale([...locales] as string[]);
    applyLocale(effective || "en");
  } catch {
    applyLocale("en");
  }
}

/** Returns the currently active Paraglide locale tag (e.g. "en"). */
export function currentLocale(): string {
  return getLocale() as string;
}
