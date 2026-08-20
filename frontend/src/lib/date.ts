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
