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

// Request sequence counter to avoid race conditions during rapid language switching.
let _currentRequestId = 0;

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
 * Called during app mount.
 */
export async function initLocale(): Promise<void> {
  const reqId = ++_currentRequestId;
  try {
    const isDesktop = await SystemSettingsService.IsDesktopMode().catch(() => false);
    let effective = "";

    if (!isDesktop) {
      const localLang = localStorage.getItem("language") || "system";
      const targetTag = localLang === "system" ? navigator.language || "en" : localLang;
      effective = await SystemSettingsService.GetEffectiveLocaleFromTag(targetTag, [
        ...locales,
      ] as string[]);
    }

    if (!effective) {
      effective = await SystemSettingsService.GetEffectiveLocale([...locales] as string[]);
    }

    if (reqId !== _currentRequestId) return;
    applyLocale(effective || "en");
  } catch {
    if (reqId !== _currentRequestId) return;
    applyLocale("en");
  }
}

/**
 * Persist user's selected language preference ("system" or BCP 47 tag) and update active locale.
 * Throws on persistence or resolution failure so callers can revert transient UI selection.
 */
export async function setLanguagePreference(lang: string): Promise<void> {
  const reqId = ++_currentRequestId;
  const prevLocal = localStorage.getItem("language");

  try {
    const isDesktop = await SystemSettingsService.IsDesktopMode().catch(() => false);
    if (isDesktop) {
      await SystemSettingsService.SetLanguage(lang);
    }

    const targetTag = !isDesktop && lang === "system" ? navigator.language || "en" : lang;

    const effective = isDesktop
      ? await SystemSettingsService.GetEffectiveLocale([...locales] as string[])
      : await SystemSettingsService.GetEffectiveLocaleFromTag(targetTag, [...locales] as string[]);

    if (reqId !== _currentRequestId) return;

    localStorage.setItem("language", lang);
    applyLocale(effective || "en");
  } catch (err) {
    if (reqId !== _currentRequestId) return;
    if (prevLocal !== null) {
      localStorage.setItem("language", prevLocal);
    } else {
      localStorage.removeItem("language");
    }
    throw err;
  }
}

/** Returns the currently active Paraglide locale tag (e.g. "en"). Reactive to locale changes. */
export function currentLocale(): string {
  getLocaleVersion();
  return getLocale() as string;
}
