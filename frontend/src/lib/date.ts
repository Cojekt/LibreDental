/**
 * Format a Date or ISO string into a local 'YYYY-MM-DD' date string.
 * This respects the browser's local timezone instead of defaulting to UTC (toISOString).
 */
export function getLocalDateString(input: Date | string = new Date()): string {
  if (typeof input === "string") {
    if (!input) return getLocalDateString(new Date());
    if (/^\d{4}-\d{2}-\d{2}$/.test(input)) return input;
    const parsed = new Date(input);
    if (isNaN(parsed.getTime())) return "";
    const yyyy = parsed.getFullYear();
    const mm = String(parsed.getMonth() + 1).padStart(2, "0");
    const dd = String(parsed.getDate()).padStart(2, "0");
    return `${yyyy}-${mm}-${dd}`;
  }

  if (isNaN(input.getTime())) return "";
  const yyyy = input.getFullYear();
  const mm = String(input.getMonth() + 1).padStart(2, "0");
  const dd = String(input.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

/**
 * Check if an ISO date/time string matches a local 'YYYY-MM-DD' date string.
 */
export function isSameDay(isoStr: string, dateStr: string): boolean {
  if (!isoStr || !dateStr) return false;
  return getLocalDateString(isoStr) === dateStr;
}

/**
 * Construct a Date object at 00:00:00 local time for a given 'YYYY-MM-DD' string.
 */
export function parseLocalDate(dateStr: string): Date {
  if (!dateStr) return new Date();
  const parts = dateStr.split("-").map(Number);
  if (parts.length < 3 || parts.some(isNaN)) return new Date();
  return new Date(parts[0], parts[1] - 1, parts[2], 0, 0, 0, 0);
}

/**
 * Returns today's local date as a 'YYYY-MM-DD' string.
 */
export function getTodayDateString(): string {
  return getLocalDateString(new Date());
}

/**
 * Resolve a stored date-only value (e.g. date of birth) to its 'YYYY-MM-DD' calendar date.
 * These are persisted as full timestamps: new records at noon UTC, older UI records at
 * local noon, and seeded/imported records at midnight UTC. Reading a midnight-UTC value in
 * local time lands on the previous day west of Greenwich, so UTC-anchored values (00:00 or
 * 12:00 UTC) are read as UTC dates and anything else falls back to the local date.
 */
export function getDateOnlyString(input?: string | null): string {
  if (!input) return "";
  if (/^\d{4}-\d{2}-\d{2}$/.test(input)) return input;
  const d = new Date(input);
  if (isNaN(d.getTime())) return "";
  const utcAnchored =
    d.getUTCMinutes() === 0 &&
    d.getUTCSeconds() === 0 &&
    d.getUTCMilliseconds() === 0 &&
    (d.getUTCHours() === 0 || d.getUTCHours() === 12);
  if (!utcAnchored) return getLocalDateString(d);
  const yyyy = d.getUTCFullYear();
  const mm = String(d.getUTCMonth() + 1).padStart(2, "0");
  const dd = String(d.getUTCDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

/**
 * Convert a 'YYYY-MM-DD' date-only value to the timestamp stored for it (noon UTC), which
 * getDateOnlyString reads back as the same calendar date in every timezone.
 */
export function dateOnlyToISO(dateStr: string): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return "";
  return `${dateStr}T12:00:00Z`;
}

/**
 * Format a stored date-only value for display without timezone shifting.
 */
export function formatDateOnly(input?: string | null, locale?: string): string {
  const dateStr = getDateOnlyString(input);
  if (!dateStr) return "";
  const [y, m, d] = dateStr.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString(locale, { timeZone: "UTC" });
}

/**
 * Whole years elapsed since a stored date-only value (e.g. age from date of birth), or null.
 */
export function calculateAge(input?: string | null, today: Date = new Date()): number | null {
  const dateStr = getDateOnlyString(input);
  if (!dateStr) return null;
  const [y, m, d] = dateStr.split("-").map(Number);
  let age = today.getFullYear() - y;
  const monthDiff = today.getMonth() + 1 - m;
  if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < d)) age--;
  return age >= 0 ? age : null;
}
