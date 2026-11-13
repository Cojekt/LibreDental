import { test, expect, onboard, createPatient } from "./fixtures";

// UTC+13 in January (not Kiritimati, which was UTC-10 until 1995): local midnight on the
// birth date is the previous day in UTC and on the server (see SERVER_TZ).
test.use({ timezoneId: "Pacific/Auckland" });

test("patient date of birth and payment cents survive the round trip, with audit entries", async ({
  page,
  server,
}) => {
  await onboard(page, server);

  await createPatient(page, "Roundtrip", "Patient");

  const [patient] = server.query<{ id: string; dob: string }>(
    "main",
    "SELECT id, date(date_of_birth) AS dob FROM patients WHERE last_name = 'Patient'"
  );
  expect(patient.dob).toBe("1990-01-15");
  await expect(page.getByRole("row").filter({ hasText: "Roundtrip" })).toContainText("1/15/1990");

  await page.getByRole("button", { name: "Billing & Claims", exact: true }).click();
  await page.getByRole("button", { name: "Payments", exact: true }).click();
  await page.getByRole("button", { name: "Record Payment", exact: true }).click();
  await page.locator("#pay-patient").selectOption({ label: "Roundtrip Patient" });
  await page.locator("#pay-amount").fill("5.67");
  await page.getByRole("dialog").getByRole("button", { name: "Record Payment" }).click();
  await expect(page.getByRole("heading", { name: "Record Payment" })).toBeHidden();
  await expect(page.getByText("$5.67").first()).toBeVisible();

  const payments = server.query<{ amount: number }>(
    "main",
    "SELECT amount FROM payments WHERE patient_id = ?",
    patient.id
  );
  expect(payments).toEqual([{ amount: 567 }]);

  const [provider] = server.query<{ id: string }>("main", "SELECT id FROM providers");
  const audit = server.query<{ resource: string; user_id: string }>(
    "audit",
    "SELECT resource, user_id FROM audit_logs WHERE patient_id = ? AND action = 'CREATE'",
    patient.id
  );
  expect(audit).toEqual(
    expect.arrayContaining([
      { resource: "patient_demographics", user_id: provider.id },
      { resource: "payment", user_id: provider.id },
    ])
  );
});
