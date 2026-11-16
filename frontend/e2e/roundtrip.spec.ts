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

test("editing appointment notes preserves the exact schedule", async ({ page, server }) => {
  await onboard(page, server);
  await createPatient(page, "Appointment", "Patient");

  server.exec(
    `INSERT INTO operatories (id, name, created_at, updated_at)
     VALUES ('precision-room', 'Precision Room', 0, 0)`
  );
  // Tomorrow at 01:00 UTC falls in the afternoon in the browser's timezone.
  const startSeconds = (Math.floor(Date.now() / 86_400_000) + 1) * 86_400 + 3_600;
  server.exec(
    `INSERT INTO appointments (
       id, patient_id, provider_id, operatory_id, start_time, end_time, status, created_at, updated_at
     )
     SELECT 'precision-appointment', patients.id, providers.id, 'precision-room',
       CAST(? AS INTEGER) * 1000000000 + 500000123,
       (CAST(? AS INTEGER) + 3600) * 1000000000 + 750000456, 'scheduled', 0, 0
     FROM patients CROSS JOIN providers WHERE patients.first_name = 'Appointment'`,
    startSeconds,
    startSeconds
  );
  const readAppointment = () =>
    server.query<{ start: string; end: string; notes: string }>(
      "main",
      `SELECT CAST(start_time AS TEXT) AS start, CAST(end_time AS TEXT) AS end, notes
       FROM appointments WHERE id = 'precision-appointment'`
    )[0];
  const original = readAppointment();

  await page.reload();
  await page.getByRole("button", { name: "Appointments", exact: true }).click();
  await page.getByRole("button", { name: "Agenda View", exact: true }).click();
  await page
    .getByRole("row")
    .filter({ hasText: "Appointment Patient" })
    .getByRole("button", { name: "Edit", exact: true })
    .click();
  await page.locator("#appt-notes").fill("Updated notes");
  await page.getByRole("button", { name: "Save Changes", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Edit Appointment" })).toBeHidden();

  expect(readAppointment()).toEqual({ ...original, notes: "Updated notes" });
});
