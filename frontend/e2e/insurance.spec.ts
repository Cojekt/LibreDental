import { test, expect, onboard, createPatient, type Page } from "./fixtures";

// The rest of insurance pricing is covered by internal/storage/sqlite/insurance_repo_test.go
// and component-tests/insurance.test.ts. These cover what needs both a real backend and the real UI.

const button = (p: Page, name: string) => p.getByRole("button", { name, exact: true });

async function openInsuranceFees(p: Page) {
  await button(p, "Billing & Claims").click();
  await button(p, "Insurance Fees").click();
}

async function setFee(p: Page, code: string, amount: number) {
  const input = p.getByRole("spinbutton", { name: `Fee for ${code}`, exact: true });
  const row = p.getByRole("row").filter({ has: input });
  await input.fill(String(amount));
  await row.getByRole("button", { name: "Save", exact: true }).click();
  await expect(row.getByRole("button", { name: "Remove override", exact: true })).toBeVisible();
  await p.waitForFunction(() => !document.querySelector('[aria-busy="true"]'));
}

async function reviewWith(p: Page, source: string, amount: string) {
  await button(p, source).click();
  await button(p, "Review selected insurance price").click();
  await expect(p.getByText(`Final amount to save: $${amount}`, { exact: true })).toBeVisible();
}

// A second tab signed in as the same provider: a separate client against the same backend.
async function secondClient(p: Page, url: string) {
  const session = await p.evaluate(() => Object.fromEntries(Object.entries(sessionStorage)));
  const other = await p.context().newPage();
  await other.addInitScript((s) => {
    for (const [k, v] of Object.entries(s)) sessionStorage.setItem(k, v);
  }, session);
  await other.goto(url);
  return other;
}

test("a fee changed by another client invalidates a pending price review", async ({
  page,
  server,
}) => {
  await onboard(page, server);
  await openInsuranceFees(page);
  await page.getByRole("textbox", { name: "Insurance provider name", exact: true }).fill("Plan A");
  await button(page, "Add provider").click();
  await setFee(page, "D0120", 55);
  await createPatient(page, "Pricing", "Patient");

  await button(page, "Billing & Claims").click();
  await button(page, "Claims").click();
  await button(page, "New Claim").click();
  await page.locator("#cl-patient").selectOption({ label: "Pricing Patient" });
  await button(page, "+ Add Row").click();
  await page.getByRole("textbox", { name: "Procedure Code", exact: true }).fill("D0120");
  await page.getByRole("textbox", { name: "Description", exact: true }).fill("Evaluation");
  await page.getByRole("dialog").getByRole("button", { name: "New Claim", exact: true }).click();
  await reviewWith(page, "Plan A", "55.00");

  const other = await secondClient(page, server.url);
  await openInsuranceFees(other);
  await button(other, "Plan A").click();
  await setFee(other, "D0120", 70);

  await button(page, "Confirm").click();
  await expect(
    page
      .getByText("The price changed. Review the current amount and confirm again.", {
        exact: true,
      })
      .first()
  ).toBeVisible();
  expect(server.query("main", "SELECT id FROM claims")).toEqual([]);

  await button(page, "Standard pricing").click();
  await reviewWith(page, "Plan A", "70.00");
  await button(page, "Confirm").click();
  await expect(page.getByRole("heading", { name: "New Claim", exact: true })).toBeHidden();

  const [claim] = server.query<{ line_items: string }>("main", "SELECT line_items FROM claims");
  expect(JSON.parse(claim.line_items).map((li: { fee: number }) => li.fee)).toEqual([7000]);
});

// Claims saved before insurance pricing existed have no pricing snapshot. The frontend must send
// them back in a shape the backend accepts as unchanged, without demanding a new price review.
test("a notes-only edit to a pre-pricing claim keeps its saved fee", async ({ page, server }) => {
  await onboard(page, server);
  await createPatient(page, "Legacy", "Patient");
  const [patient] = server.query<{ id: string }>("main", "SELECT id FROM patients");
  server.exec(
    `INSERT INTO claims (id, patient_id, date_of_service, status, notes, line_items, created_at, updated_at)
     VALUES ('legacy-claim', ?, '2026-01-01', 'draft', '', ?, '2026-01-01', '2026-01-01')`,
    patient.id,
    JSON.stringify([
      { id: "legacy-line", ada_code: "D0120", description: "Legacy", fee: 12345 },
      { id: "legacy-manual", ada_code: "", description: "Legacy manual", fee: 6789 },
    ])
  );

  await button(page, "Billing & Claims").click();
  await button(page, "Claims").click();
  await page
    .getByRole("row")
    .filter({ hasText: "$191.34" })
    .getByRole("button", { name: "Edit", exact: true })
    .click();
  await page.locator("#cl-notes").fill("Notes-only correction");
  await button(page, "Save Changes").click();
  await expect(page.getByRole("heading", { name: "Edit Claim", exact: true })).toBeHidden();
  await expect(
    page.getByRole("heading", { name: "Review and confirm procedure price" })
  ).toHaveCount(0);

  const [claim] = server.query<{ notes: string; line_items: string }>(
    "main",
    "SELECT notes, line_items FROM claims WHERE id = 'legacy-claim'"
  );
  expect(claim.notes).toBe("Notes-only correction");
  type Line = { ada_code: string; fee: number; pricing?: unknown };
  const lines = JSON.parse(claim.line_items) as Line[];
  expect(lines.map(({ ada_code, fee, pricing }) => ({ ada_code, fee, pricing }))).toEqual([
    { ada_code: "D0120", fee: 12345, pricing: undefined },
    { ada_code: "", fee: 6789, pricing: undefined },
  ]);
});
