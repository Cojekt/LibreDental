import { currentLocale } from "./locale.svelte.js";

/**
 * Format an integer cent amount as a localized currency string.
 * Falls back to a plain decimal string if the currency code is missing or unsupported.
 */
export function formatCurrency(cents: number, currencyCode?: string | null): string {
  const curr = currencyCode || "";
  if (!curr) return (cents / 100).toFixed(2);
  try {
    // narrowSymbol: a Canadian clinic's own currency reads "$120.00", not "CA$120.00".
    return new Intl.NumberFormat(currentLocale(), {
      style: "currency",
      currency: curr,
      currencyDisplay: "narrowSymbol",
    }).format(cents / 100);
  } catch {
    return `${(cents / 100).toFixed(2)}`;
  }
}
