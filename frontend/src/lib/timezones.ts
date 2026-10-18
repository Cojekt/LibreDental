// IANA timezones for the countries LibreDental supports. A fixed list avoids depending on
// Intl.supportedValuesOf("timeZone"), which isn't available in every webview.
const ZONES_BY_COUNTRY: Record<string, string[]> = {
  US: [
    "America/New_York",
    "America/Chicago",
    "America/Denver",
    "America/Phoenix",
    "America/Los_Angeles",
    "America/Anchorage",
    "Pacific/Honolulu",
  ],
  CA: [
    "America/St_Johns",
    "America/Halifax",
    "America/Toronto",
    "America/Winnipeg",
    "America/Regina",
    "America/Edmonton",
    "America/Vancouver",
  ],
  GB: ["Europe/London"],
  AU: [
    "Australia/Perth",
    "Australia/Darwin",
    "Australia/Adelaide",
    "Australia/Brisbane",
    "Australia/Sydney",
    "Australia/Hobart",
  ],
  DE: ["Europe/Berlin"],
  FR: ["Europe/Paris"],
};

/** The timezone of this computer, as the browser or webview reports it. */
export function getComputerTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "";
  } catch {
    return "";
  }
}

/**
 * Timezones to offer for a country. The current value and this computer's timezone are
 * included even when they aren't in the list, so an existing setting is never hidden.
 */
export function timezoneOptions(country: string, ...include: string[]): string[] {
  const zones = [...(ZONES_BY_COUNTRY[country] ?? [])];
  for (const zone of include) {
    if (zone && !zones.includes(zone)) zones.push(zone);
  }
  return zones;
}
