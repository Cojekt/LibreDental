import { test, expect, onboard, createPatient } from "./fixtures";

// A failed chart load must never leave the previously selected patient's
// conditions on screen (they would appear under the new patient's name), and
// must surface the failure instead of silently showing an empty chart.
test("a failed chart load clears the previous patient and shows an error", async ({
  page,
  server,
}) => {
  await onboard(page, server);
  await createPatient(page, "Alpha", "One");
  await createPatient(page, "Beta", "Two");

  const id = (first: string) =>
    server.query<{ id: string }>("main", "SELECT id FROM patients WHERE first_name = ?", first)[0]
      .id;
  for (const [first, desc] of [
    ["Alpha", "ALPHA-ONLY-CONDITION"],
    ["Beta", "BETA-ONLY-CONDITION"],
  ]) {
    server.exec(
      `INSERT INTO dental_conditions (id, patient_id, tooth_number, description, status, created_at, updated_at)
       VALUES (?, ?, 3, ?, 'existing', 1, 1)`,
      first,
      id(first),
      desc
    );
  }
  // Corrupt Beta's chart so the backend scan fails when it is selected.
  server.exec("UPDATE dental_conditions SET surfaces = 'not-json' WHERE id = 'Beta'");

  await page.getByRole("button", { name: "Dental Charting", exact: true }).click();
  const select = page.locator("#chart-patient-select");

  await select.selectOption(id("Alpha"));
  await expect(page.getByText("ALPHA-ONLY-CONDITION").first()).toBeVisible();

  await select.selectOption(id("Beta"));
  // Alpha's condition must be gone, and an error must be shown to the user.
  await expect(page.getByText("ALPHA-ONLY-CONDITION")).toHaveCount(0);
  await expect(page.getByRole("alert")).toBeVisible();
});
