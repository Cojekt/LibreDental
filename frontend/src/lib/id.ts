// A version 4 UUID built with getRandomValues. crypto.randomUUID only exists in secure
// contexts (HTTPS or localhost), so it throws for LAN clients on plain http.
export function newUuid(): string {
  const b = crypto.getRandomValues(new Uint8Array(16));
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

// Builds a record ID like the backend's "<prefix>_" + uuid.NewString().
export function newId(prefix: string): string {
  return `${prefix}_${newUuid()}`;
}
