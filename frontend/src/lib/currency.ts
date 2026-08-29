/**
 * Format an integer cent amount as a localized currency string.
 * Falls back to a plain decimal string if the currency code is missing or unsupported.
 */
export function formatCurrency(cents: number, currencyCode?: string | null): string {
  const curr = currencyCode || "";
  if (!curr) return (cents / 100).toFixed(2);
  try {
    return new Intl.NumberFormat("en-US", { style: "currency", currency: curr }).format(
      cents / 100
    );
  } catch {
    return `${(cents / 100).toFixed(2)}`;
  }
}
