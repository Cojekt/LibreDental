import { expect, it } from "vitest";
import { newId } from "../src/lib/id";

it("builds a prefixed version 4 UUID", () => {
  expect(newId("appt")).toMatch(
    /^appt_[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
  );
});

it("does not repeat when called back to back", () => {
  const ids = new Set(Array.from({ length: 1000 }, () => newId("pay")));
  expect(ids.size).toBe(1000);
});
