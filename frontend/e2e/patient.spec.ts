import { test, expect, onboard, createPatient, PROVIDER, type Page } from "./fixtures";

async function signInSecond(page: Page, url: string) {
  await page.goto(url);
  await page.getByRole("button", { name: "Not logged in", exact: true }).click();
  await page.getByRole("button", { name: new RegExp(PROVIDER.name) }).click();
  await page.locator("#pin-input").fill(PROVIDER.pin);
  await page.getByRole("button", { name: "Sign In", exact: true }).click();
  await expect(page.getByRole("button", { name: PROVIDER.name, exact: true })).toBeVisible();
}

async function openEdit(page: Page) {
  await page.getByRole("button", { name: "Patients", exact: true }).click();
  await page
    .getByRole("row")
    .filter({ hasText: "Concurrent" })
    .getByRole("button", { name: "Edit", exact: true })
    .click();
  await expect(page.getByRole("heading", { name: "Edit Patient" })).toBeVisible();
}

// Two staff editing the same patient must not silently clobber each other's
// changes: the backend row carries a version, and a stale save is rejected as a
// conflict rather than overwriting a field the second editor never saw.
test("a concurrent patient edit is rejected instead of losing data", async ({
  page,
  browser,
  server,
}) => {
  await onboard(page, server);
  await createPatient(page, "Concurrent", "Patient");

  const second = await (await browser.newContext()).newPage();
  // The page fixture only guards the first page's console; watch the second too.
  const secondErrors: string[] = [];
  second.on("pageerror", (e) => secondErrors.push(e.message));
  await signInSecond(second, server.url);

  // Both open the edit form at the same version.
  await openEdit(page);
  await openEdit(second);

  // First editor records a medical alert and saves successfully.
  await page.locator("#alerts").fill("Penicillin allergy");
  await page.getByRole("button", { name: "Save Changes", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Edit Patient" })).toBeHidden();

  // Second editor, working from the now-stale version, changes a different field.
  await second.locator("#city").fill("Szeged");
  await second.getByRole("button", { name: "Save Changes", exact: true }).click();

  // The stale save must be rejected: the second editor sees a conflict error
  // (which keeps the modal open) instead of silently overwriting. Waiting on
  // the banner is deterministic, unlike reading the row before the save lands.
  await expect(
    second.getByText("This record was changed by someone else. Reload it and try again.")
  ).toBeVisible();
  await expect(second.getByRole("heading", { name: "Edit Patient" })).toBeVisible();
  const [row] = server.query<{ alerts: string; city: string }>(
    "main",
    "SELECT medical_alerts AS alerts, COALESCE(city,'') AS city FROM patients WHERE last_name = 'Patient'"
  );
  expect(row.alerts).toContain("Penicillin allergy");
  expect(row.city).not.toBe("Szeged");
  expect(secondErrors, "uncaught errors on the second editor's page").toEqual([]);
});
