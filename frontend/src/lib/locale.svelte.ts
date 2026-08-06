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

import { setLocale, getLocale, locales } from '../paraglide/runtime.js';
import { SystemSettingsService } from '../../bindings/github.com/LibreDental/libredental/pkg/services/index.js';

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
 * Resolve the saved language preference and activate it.
 * Called once during app mount.
 *
 * @param savedLang  From SystemSettingsService.GetLanguage() —
 *                   a BCP 47 tag (e.g. "en", "fr") or "system".
 */
export async function initLocale(savedLang: string): Promise<void> {
  let tag = savedLang;

  if (tag === 'system' || tag === '') {
    try {
      const osLocale = await SystemSettingsService.GetSystemLocale();
      tag = osLocale || 'en';
    } catch {
      tag = 'en';
    }
  }

  // Normalize: find the best-matching supported locale.
  // With English-only, everything maps to "en". Adding "fr" to settings.json
  // and creating messages/fr.json will automatically make French selectable.
  const match: string =
    (locales as readonly string[]).find((l) => l === tag) ??
    (locales as readonly string[]).find((l) => tag.startsWith(l)) ??
    (locales as readonly string[]).find((l) => l === tag.split('-')[0]) ??
    'en';

  applyLocale(match);
}

/** Returns the currently active Paraglide locale tag (e.g. "en"). */
export function currentLocale(): string {
  return getLocale() as string;
}
