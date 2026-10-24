const assert = require("node:assert/strict");
const { spawn } = require("node:child_process");
const { mkdtemp, mkdir, writeFile } = require("node:fs/promises");
const { tmpdir } = require("node:os");
const path = require("node:path");
const net = require("node:net");
const { DatabaseSync } = require("node:sqlite");
const { chromium } = require("playwright");
const headed = process.env.LIBREDENTAL_E2E_HEADED === "1";
const launchOptions = { headless: !headed, slowMo: headed ? 250 : 0 };

(async () => {
  const binary = process.env.LIBREDENTAL_E2E_BINARY;
  assert(binary, "Set LIBREDENTAL_E2E_BINARY to a built LibreDental server binary");
  const dir = await mkdtemp(path.join(tmpdir(), "libredental-e2e-"));
  const config = path.join(dir, "config");
  await mkdir(config);
  const port = await new Promise((resolve) => {
    const socket = net.createServer();
    socket.listen(0, "127.0.0.1", () => {
      const port = socket.address().port;
      socket.close(() => resolve(port));
    });
  });
  const url = `http://127.0.0.1:${port}`;
  let child, browser;
  const results = [],
    errors = [];
  const start = async () => {
    child = spawn(path.resolve(binary), [], {
      env: {
        ...process.env,
        XDG_CONFIG_HOME: config,
        APPDATA: config,
        LIBREDENTAL_HOST: "127.0.0.1",
        LIBREDENTAL_PORT: String(port),
      },
      stdio: ["ignore", "pipe", "pipe"],
    });
    child.on("error", (e) => errors.push(e.message));
    child.stdout.on("data", () => {});
    child.stderr.on("data", () => {});
    for (let i = 0; i < 100; i++) {
      if (child.exitCode != null) throw Error(`Server exited: ${child.exitCode}`);
      try {
        if ((await fetch(url)).ok) return;
      } catch {}
      await new Promise((r) => setTimeout(r, 100));
    }
    throw Error("Server startup timed out");
  };
  const stop = async () => {
    if (!child || child.exitCode != null) return;
    await new Promise((resolve) => {
      child.once("exit", resolve);
      child.kill();
    });
  };
  const pass = (s) => {
    results.push(s);
    console.log("PASS", s);
  };
  try {
    await start();
    browser = await chromium.launch(launchOptions);
    const context = await browser.newContext({ viewport: { width: 1440, height: 1100 } });
    await context.tracing.start({ screenshots: true, snapshots: true, sources: true });
    const page = await context.newPage();
    page.on("pageerror", (e) => errors.push(e.message));
    const button = (name) => page.getByRole("button", { name, exact: true });
    const dialog = () =>
      page
        .getByRole("dialog")
        .filter({ has: page.getByRole("heading", { name: "Review and confirm procedure price" }) });
    const final = (amount) =>
      page.getByText(`Final amount to save: $${amount}`, { exact: true }).waitFor();
    const catalog = async () => {
      await button("Billing & Claims").click();
      await button("Insurance Fee Overrides").click();
    };
    const setFee = async (p, code, amount) => {
      const row = p.getByRole("row").filter({ hasText: `${code} —` });
      await row.getByRole("spinbutton").fill(String(amount));
      await row.getByRole("button", { name: "Save", exact: true }).click();
      await row.getByRole("button", { name: "Remove override", exact: true }).waitFor();
      await p.waitForFunction(() => !document.querySelector('[aria-busy="true"]'));
    };
    const chart = async () => {
      await button("Dental Charting").click();
      await page
        .getByRole("combobox")
        .selectOption({ label: "Synthetic PricingPatient (1/15/1990)" });
    };
    const openPrice = async (tooth, code) => {
      await page.getByRole("button", { name: new RegExp(`^${tooth}(?:\\s|$)`) }).click();
      await page
        .getByRole("combobox")
        .filter({ has: page.locator(`option[value="${code}"]`) })
        .selectOption(code);
      await page.getByRole("dialog").getByRole("button", { name: "Save", exact: true }).click();
      await button("Confirm").waitFor();
      await page.getByText(/National Base Fee: \$/).waitFor();
    };
    const choose = async (plan) => {
      await button(plan).click();
      await button("Review selected insurance price").click();
    };
    const confirm = async () => {
      await button("Confirm").click();
      await page
        .getByRole("heading", { name: "Review and confirm procedure price" })
        .waitFor({ state: "hidden" });
    };
    const cancel = async () => {
      await dialog().getByRole("button", { name: "Cancel", exact: true }).click();
      await page.getByRole("dialog").getByRole("button", { name: "Cancel", exact: true }).click();
    };
    await page.goto(url);
    await button("Continue").click();
    await page.locator("#onboard-prov-name").fill("Synthetic Verification Dentist");
    await page.locator("#onboard-prov-pin").fill("8675");
    await page.locator("#onboard-prov-pin-confirm").fill("8675");
    await button("Save & Complete Setup").click();
    await page
      .getByRole("heading", { name: "Welcome to LibreDental!" })
      .waitFor({ state: "hidden" });
    pass("Synthetic dentist onboarding and authentication");
    const loggedOut = await context.newPage();
    await loggedOut.goto(url);
    await loggedOut.getByRole("button", { name: "Not logged in", exact: true }).waitFor();
    assert.equal(
      await loggedOut.getByRole("button", { name: "Billing & Claims", exact: true }).count(),
      0
    );
    await loggedOut.screenshot({ path: path.join(dir, "logged-out-navigation.png") });
    await loggedOut.close();
    pass("Logged-out navigation prevents billing access");
    await page.bringToFront();
    await catalog();
    for (const [insurer, evaluation, cleaning] of [
      ["Insurance A", 40, 60],
      ["Insurance B", 20, 0],
    ]) {
      await page
        .getByRole("textbox", { name: "Insurance provider name", exact: true })
        .fill(insurer);
      await button("Add provider").click();
      await setFee(page, "D0120", evaluation);
      await setFee(page, "D1110", cleaning);
    }
    await page.getByRole("textbox", { name: "Search insurance providers" }).fill("Insurance B");
    assert(await button("Insurance B").isVisible());
    pass("Independent insurance providers, procedure fees, zero-dollar fee and search");
    await button("Patients").click();
    await button("New Patient").click();
    await page.locator("#fname").fill("Synthetic");
    await page.locator("#lname").fill("PricingPatient");
    await page.locator("#dob").focus();
    await page.locator("#dob").fill("1990-01-15");
    await page.locator("#phone").fill("2025550100");
    await button("Save Patient Record").click();
    await page.getByRole("heading", { name: "Add New Patient" }).waitFor({ state: "hidden" });

    await chart();
    await openPrice(3, "D0120");
    assert.equal(await page.getByText("Latest Used", { exact: true }).count(), 0);
    await page.getByText("National Base Fee: $65.00", { exact: true }).waitFor();
    await button("Insurance A").click();
    await button("Review selected insurance price").click();
    await final("40.00");
    await confirm();
    await button("Bill Charted Procedures").click();
    await button("Generating Claim...").waitFor({ state: "hidden" });
    pass(
      "New patient, original reference price, explicit insurance confirmation, treatment and chart-to-claim"
    );
    await openPrice(4, "D1110");
    await choose("Insurance A");
    await final("60.00");
    await choose("Insurance B");
    await final("0.00");
    await button("Confirm").dblclick();
    await page
      .getByRole("heading", { name: "Review and confirm procedure price" })
      .waitFor({ state: "hidden" });
    pass("Explicit provider selection across procedures and zero-dollar confirmation");
    await openPrice(5, "D0140");
    await choose("Insurance B");
    await page.getByText("No configured fee", { exact: true }).first().waitFor();
    assert(await button("Confirm").isDisabled());
    await cancel();
    await openPrice(6, "D0140");
    await button("Standard pricing").click();
    await final("85.00");
    await confirm();
    pass("Missing fee blocks confirmation; cancellation; deliberate standard pricing");
    await catalog();
    await button("Insurance A").click();
    await setFee(page, "D0120", 55);
    await button("Claims").click();
    await page.getByRole("row").filter({ hasText: "$40.00" }).waitFor();
    await button("New Claim").click();
    await page.locator("#cl-patient").selectOption({ label: "Synthetic PricingPatient" });
    await button("+ Add Row").click();
    await page.getByRole("textbox", { name: "Procedure Code", exact: true }).fill("D0120");
    await page
      .getByRole("textbox", { name: "Description", exact: true })
      .fill("Synthetic direct evaluation");
    await page.getByRole("dialog").getByRole("button", { name: "New Claim", exact: true }).click();
    await button("Standard pricing").click();
    await final("65.00");
    await choose("Insurance A");
    await final("55.00");
    const other = await context.newPage();
    const ss = await page.evaluate(() => Object.fromEntries(Object.entries(sessionStorage)));
    await other.addInitScript((s) => {
      for (const [k, v] of Object.entries(s)) sessionStorage.setItem(k, v);
    }, ss);
    await other.goto(url);
    await other.getByRole("button", { name: "Billing & Claims", exact: true }).click();
    await other.getByRole("button", { name: "Insurance Fee Overrides", exact: true }).click();
    await other.getByRole("button", { name: "Insurance A", exact: true }).click();
    await setFee(other, "D0120", 70);
    await button("Confirm").click();
    await page
      .getByText("The price changed. Review the current amount and confirm again.", { exact: true })
      .first()
      .waitFor();
    await button("Standard pricing").click();
    await final("65.00");
    await choose("Insurance A");
    await final("70.00");
    await confirm();
    await page
      .getByRole("heading", { name: "New Claim", exact: true })
      .waitFor({ state: "hidden" });
    assert.equal(
      await page
        .getByText("The price changed. Review the current amount and confirm again.", {
          exact: true,
        })
        .count(),
      0
    );
    await page.getByRole("row").filter({ hasText: "$40.00" }).waitFor();
    await page.getByRole("row").filter({ hasText: "$70.00" }).waitFor();
    pass(
      "Direct claim saves current fee; concurrent price change rejects stale review; historical claim remains unchanged"
    );
    await other.getByRole("button", { name: "Archive", exact: true }).click();
    await chart();
    await openPrice(7, "D0120");
    await page.getByRole("checkbox", { name: "Show archived entries" }).check();
    await button("Insurance A (Archived)").click();
    await button("Review selected insurance price").click();
    await page
      .getByText("This insurance provider is unavailable or archived. Choose another source.", {
        exact: true,
      })
      .first()
      .waitFor();
    assert(await button("Confirm").isDisabled());
    await cancel();
    await other.getByRole("checkbox", { name: "Show archived entries" }).check();
    await other.getByRole("button", { name: "Restore", exact: true }).click();
    await other.getByRole("button", { name: "Rename", exact: true }).click();
    await other
      .getByRole("textbox", { name: "Insurance provider name", exact: true })
      .fill("Insurance A Renamed");
    await other.getByRole("button", { name: "Save", exact: true }).first().click();
    pass("Archived provider cannot be charged; restore and rename preserve historical prices");
    await page
      .getByRole("row")
      .filter({ hasText: "D0120" })
      .getByText("$40.00", { exact: true })
      .waitFor();
    assert.equal(await page.getByRole("row").filter({ hasText: "D1110" }).count(), 1);
    assert.equal(await page.getByRole("row").filter({ hasText: "Tooth 5" }).count(), 0);
    await page.screenshot({ path: path.join(dir, "treatments.png") });
    await context.tracing.stop({ path: path.join(dir, "workflow-trace.zip") });
    await browser.close();
    browser = null;
    await stop();
    const databasePath = path.join(config, "LibreDental", "libredental.db");
    const fixture = new DatabaseSync(databasePath);
    const patientID = fixture
      .prepare("SELECT id FROM patients WHERE first_name=? AND last_name=?")
      .get("Synthetic", "PricingPatient").id;
    for (const [id, code, fee] of [
      ["legacy-claim", "D0120", 12345],
      ["legacy-manual-claim", "", 6789],
    ]) {
      fixture
        .prepare(
          "INSERT INTO claims(id,patient_id,date_of_service,status,notes,line_items,created_at,updated_at) VALUES(?,?,?,'draft',?,?,?,?)"
        )
        .run(
          id,
          patientID,
          "2026-01-01",
          id,
          JSON.stringify([{ id: id + "-line", ada_code: code, description: id, fee }]),
          "2026-01-01",
          "2026-01-01"
        );
    }
    fixture.close();
    function savedClaim(id) {
      const db = new DatabaseSync(databasePath, { readOnly: true });
      try {
        return db.prepare("SELECT * FROM claims WHERE id=?").get(id);
      } finally {
        db.close();
      }
    }
    await start();
    browser = await chromium.launch(launchOptions);
    const restartContext = await browser.newContext();
    await restartContext.tracing.start({ screenshots: true, snapshots: true, sources: true });
    const p = await restartContext.newPage();
    await p.goto(url);
    await p.getByRole("button", { name: "Not logged in", exact: true }).click();
    await p.getByRole("button", { name: /Synthetic Verification Dentist/ }).click();
    await p.locator("#pin-input").fill("8675");
    await p.getByRole("button", { name: "Sign In", exact: true }).click();
    await p.getByRole("button", { name: "Billing & Claims", exact: true }).click();
    await p.getByRole("row").filter({ hasText: "$40.00" }).waitFor();
    await p.getByRole("row").filter({ hasText: "$70.00" }).waitFor();
    await p.screenshot({ path: path.join(dir, "claims-after-restart.png") });
    pass("Restart persistence; old and new claim amounts remain distinct");
    async function editClaimRow(amount) {
      await p
        .getByRole("row")
        .filter({ hasText: `$${amount}` })
        .getByRole("button", { name: "Edit", exact: true })
        .click();
      await p.getByRole("heading", { name: "Edit Claim", exact: true }).waitFor();
    }
    async function saveWithoutReview() {
      await p.getByRole("button", { name: "Save Changes", exact: true }).click();
      await p
        .getByRole("heading", { name: "Edit Claim", exact: true })
        .waitFor({ state: "hidden" });
      assert.equal(
        await p
          .getByRole("heading", { name: "Review and confirm procedure price", exact: true })
          .count(),
        0
      );
    }
    await editClaimRow("123.45");
    await p.locator("#cl-notes").fill("Legacy notes-only correction");
    await saveWithoutReview();
    assert.equal(JSON.parse(savedClaim("legacy-claim").line_items)[0].fee, 12345);
    assert.equal(JSON.parse(savedClaim("legacy-claim").line_items)[0].pricing, undefined);
    pass("Legacy notes-only edit preserves the saved fee without price confirmation");
    await editClaimRow("67.89");
    await p.locator("#cl-notes").fill("Legacy manual notes-only correction");
    await saveWithoutReview();
    assert.equal(JSON.parse(savedClaim("legacy-manual-claim").line_items)[0].ada_code, "");
    assert.equal(JSON.parse(savedClaim("legacy-manual-claim").line_items)[0].fee, 6789);
    pass("Legacy manual blank procedure code and fee survive an unrelated edit");
    const historicalDB = new DatabaseSync(databasePath, { readOnly: true });
    const historical = historicalDB
      .prepare("SELECT id,line_items FROM claims WHERE json_extract(line_items,'$[0].fee')=4000")
      .get();
    historicalDB.close();
    await editClaimRow("40.00");
    await p
      .locator("#cl-notes")
      .fill("Insured notes-only correction after fee changes and renames");
    await saveWithoutReview();
    assert.deepEqual(
      JSON.parse(savedClaim(historical.id).line_items),
      JSON.parse(historical.line_items)
    );
    pass(
      "Insured notes-only edit preserves historical amounts and source names after catalog changes"
    );
    await editClaimRow("123.45");
    await p.getByRole("button", { name: "+ Add Row", exact: true }).click();
    await p.getByRole("textbox", { name: "Procedure Code", exact: true }).last().fill("D0120");
    await p.getByRole("button", { name: "Save Changes", exact: true }).click();
    await p
      .getByRole("heading", { name: "Review and confirm procedure price", exact: true })
      .waitFor();
    await p.getByRole("button", { name: "Insurance A Renamed", exact: true }).click();
    await p.getByRole("button", { name: "Review selected insurance price", exact: true }).click();
    await p.getByText("Final amount to save: $70.00", { exact: true }).waitFor();
    assert.equal(JSON.parse(savedClaim("legacy-claim").line_items).length, 1);
    await p.getByRole("button", { name: "Confirm", exact: true }).click();
    await p.getByRole("heading", { name: "Edit Claim", exact: true }).waitFor({ state: "hidden" });
    const amended = JSON.parse(savedClaim("legacy-claim").line_items);
    assert.deepEqual(
      amended.map((li) => li.fee),
      [12345, 7000]
    );
    assert.equal(amended[0].pricing, undefined);
    pass("New line alone requires confirmation; the legacy line remains unchanged");
    await editClaimRow("67.89");
    await p.getByRole("spinbutton", { name: "Fee", exact: true }).fill("222");
    await p.getByRole("button", { name: "Save Changes", exact: true }).click();
    await p
      .getByRole("heading", { name: "Review and confirm procedure price", exact: true })
      .waitFor();
    await p.getByRole("button", { name: "Entered manual fee ($222.00)", exact: true }).click();
    await p.getByText("Final amount to save: $222.00", { exact: true }).waitFor();
    assert.equal(JSON.parse(savedClaim("legacy-manual-claim").line_items)[0].fee, 6789);
    await p.getByRole("button", { name: "Confirm", exact: true }).click();
    await p.getByRole("heading", { name: "Edit Claim", exact: true }).waitFor({ state: "hidden" });
    assert.equal(JSON.parse(savedClaim("legacy-manual-claim").line_items)[0].fee, 22200);
    pass("Deliberate manual repricing changes the saved amount only after confirmation");
    await p.getByRole("button", { name: "New Claim", exact: true }).click();
    await p.getByRole("button", { name: "+ Add Row", exact: true }).click();
    await p
      .getByRole("textbox", { name: "Description", exact: true })
      .fill("Synthetic zero manual charge");
    await p.getByRole("dialog").getByRole("button", { name: "New Claim", exact: true }).click();
    await p.getByText("Final amount to save: $0.00", { exact: true }).waitFor();
    await p.getByRole("button", { name: "Confirm", exact: true }).click();
    await p.getByRole("heading", { name: "New Claim", exact: true }).waitFor({ state: "hidden" });
    const zeroDB = new DatabaseSync(databasePath, { readOnly: true });
    const zeroClaim = zeroDB
      .prepare(
        "SELECT line_items FROM claims WHERE line_items LIKE '%Synthetic zero manual charge%'"
      )
      .get();
    zeroDB.close();
    assert.equal(JSON.parse(zeroClaim.line_items)[0].fee, 0);
    assert.equal(JSON.parse(zeroClaim.line_items)[0].pricing.kind, "manual");
    pass("Uncatalogued zero-dollar manual charge requires confirmation and saves zero");

    await p.getByRole("button", { name: "Bundles", exact: true }).click();
    await p.getByRole("button", { name: "Create Bundle", exact: true }).click();
    await p.locator("#b-shortname").fill("synthetic-bundle");
    await p.locator("#b-name").fill("Synthetic manual bundle");
    await p.getByRole("button", { name: "+ Add Item", exact: true }).click();
    await p.getByRole("textbox", { name: "Procedure Code", exact: true }).fill("D0120");
    await p.getByPlaceholder("Description", { exact: true }).fill("Synthetic bundle evaluation");
    await p.getByRole("spinbutton", { name: "Fee", exact: true }).fill("12.34");
    await p.getByRole("button", { name: "Save Bundle", exact: true }).click();
    await p
      .getByRole("heading", { name: "Create Bundle", exact: true })
      .waitFor({ state: "hidden" });
    await p.getByRole("button", { name: "Claims", exact: true }).click();
    await p.getByRole("button", { name: "New Claim", exact: true }).click();
    await p.locator("#cl-patient").selectOption({ label: "Synthetic PricingPatient" });
    await p.getByPlaceholder("Bundle (crwn)", { exact: true }).fill("synthetic-bundle");
    await p.getByRole("button", { name: "Apply", exact: true }).click();
    await p.getByRole("spinbutton", { name: "Fee", exact: true }).waitFor();
    await p.getByRole("dialog").getByRole("button", { name: "New Claim", exact: true }).click();
    await p.getByText("Final amount to save: $12.34", { exact: true }).waitFor();
    await p.getByRole("button", { name: "Confirm", exact: true }).click();
    await p.getByRole("heading", { name: "New Claim", exact: true }).waitFor({ state: "hidden" });
    await p.getByRole("row").filter({ hasText: "$12.34" }).waitFor();
    pass("Existing bundle entry retains its template fee instead of adopting insurance pricing");
    await p.getByRole("button", { name: "Payments", exact: true }).click();
    await p.getByRole("button", { name: "Record Payment", exact: true }).click();
    await p.locator("#pay-patient").selectOption({ label: "Synthetic PricingPatient" });
    await p.locator("#pay-amount").fill("5.67");
    await p
      .getByRole("dialog")
      .getByRole("button", { name: "Record Payment", exact: true })
      .click();
    await p
      .getByRole("heading", { name: "Record Payment", exact: true })
      .waitFor({ state: "hidden" });
    await p.getByText("$5.67", { exact: true }).first().waitFor();
    const paymentDB = new DatabaseSync(databasePath, { readOnly: true });
    assert.equal(
      paymentDB
        .prepare("SELECT amount FROM payments WHERE patient_id=? ORDER BY created_at DESC LIMIT 1")
        .get(patientID).amount,
      567
    );
    paymentDB.close();
    pass("Existing payment recording works alongside insurance-priced and historical claims");
    await p.getByRole("button", { name: "Insurance Fee Overrides", exact: true }).click();
    await p.getByRole("button", { name: "Add provider", exact: true }).waitFor();
    const sessionCopy = await p.evaluate(() => Object.fromEntries(Object.entries(sessionStorage)));
    const logoutTab = await p.context().newPage();
    await logoutTab.addInitScript((session) => {
      for (const [k, v] of Object.entries(session)) sessionStorage.setItem(k, v);
    }, sessionCopy);
    await logoutTab.goto(url);
    await logoutTab
      .getByRole("button", { name: "Synthetic Verification Dentist", exact: true })
      .click();
    await logoutTab.getByRole("button", { name: "Sign Out", exact: true }).click();
    await p.getByRole("button", { name: "Fee Schedules", exact: true }).click();
    await p.getByRole("button", { name: "Insurance Fee Overrides", exact: true }).click();
    await p.getByText("Sign in to make changes", { exact: true }).waitFor();
    assert.equal(await p.getByRole("button", { name: "Add provider", exact: true }).count(), 0);
    await p.screenshot({ path: path.join(dir, "invalid-session-insurance.png") });
    await p.getByRole("button", { name: "Claims", exact: true }).click();
    await p.getByRole("button", { name: "New Claim", exact: true }).click();
    await p.locator("#cl-patient").selectOption({ label: "Synthetic PricingPatient" });
    await p.getByRole("button", { name: "+ Add Row", exact: true }).click();
    await p.getByRole("textbox", { name: "Procedure Code", exact: true }).fill("D0120");
    await p.getByRole("dialog").getByRole("button", { name: "New Claim", exact: true }).click();
    await p.getByText("Sign in to make changes", { exact: true }).waitFor();
    assert.equal(await p.getByRole("button", { name: "Confirm", exact: true }).count(), 0);
    await p.screenshot({ path: path.join(dir, "invalid-session-pricing.png") });
    pass("Revoked session hides insurance editing and direct-claim price confirmation");
    await restartContext.tracing.stop({ path: path.join(dir, "session-trace.zip") });

    assert.deepEqual(errors, []);
  } catch (e) {
    results.push(`FAILED: ${e.stack}`);
    process.exitCode = 1;
    console.error(e);
  } finally {
    if (browser) await browser.close();
    await stop();
    await writeFile(path.join(dir, "results.json"), JSON.stringify({ results, errors }, null, 2));
    console.log("Test artifacts:", dir);
  }
})();
