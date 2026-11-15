import { afterEach, expect, it, vi } from "vitest";
import { newId, newUuid } from "../src/lib/id";

const uuidV4 = "[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}";

afterEach(() => {
  vi.unstubAllGlobals();
});

it("builds a version 4 UUID", () => {
  expect(newUuid()).toMatch(new RegExp(`^${uuidV4}$`));
});

it("builds a prefixed version 4 UUID", () => {
  expect(newId("appt")).toMatch(new RegExp(`^appt_${uuidV4}$`));
});

it("does not repeat when called back to back", () => {
  const ids = new Set(Array.from({ length: 1000 }, () => newId("pay")));
  expect(ids.size).toBe(1000);
});

it("works without crypto.randomUUID, as on a LAN client over plain http", () => {
  const real = globalThis.crypto;
  vi.stubGlobal("crypto", { getRandomValues: real.getRandomValues.bind(real) });
  expect(newUuid()).toMatch(new RegExp(`^${uuidV4}$`));
});
