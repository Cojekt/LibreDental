import { test, expect, onboard, PROVIDER } from "./fixtures";

test("fresh install onboards a provider and signs them in", async ({ page, server }) => {
  await onboard(page, server);
  await expect(page.getByRole("button", { name: PROVIDER.name, exact: true })).toBeVisible();

  const providers = server.query<{ name: string }>("main", "SELECT name FROM providers");
  expect(providers.map((p) => p.name)).toEqual([PROVIDER.name]);
});

// The frontend reads the error code from err.cause.code; this shape is produced by
// services.MarshalError and the Wails transport, which the Vitest mocks only imitate.
test("backend error codes reach the UI as translated messages", async ({
  page,
  browser,
  server,
}) => {
  await onboard(page, server);

  const other = await (await browser.newContext()).newPage();
  await other.goto(server.url);
  await other.getByRole("button", { name: "Not logged in", exact: true }).click();
  await other.getByRole("button", { name: new RegExp(PROVIDER.name) }).click();
  await other.locator("#pin-input").fill("0000");
  await other.getByRole("button", { name: "Sign In", exact: true }).click();
  await expect(other.getByText("Incorrect PIN", { exact: true })).toBeVisible();

  await other.locator("#pin-input").fill(PROVIDER.pin);
  await other.getByRole("button", { name: "Sign In", exact: true }).click();
  await expect(other.getByRole("button", { name: PROVIDER.name, exact: true })).toBeVisible();
});
