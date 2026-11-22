import { test, expect, onboard } from "./fixtures";

// A stored document must never run as script in the app's origin. In browser
// (server) mode, "Open" previously navigated a same-origin blob: tab to the
// document's content type, so an uploaded HTML file executed with access to
// the signed-in session (token, bound methods). Non-inline types must download.
test("opening an HTML document does not execute it in the app origin", async ({
  page,
  context,
  server,
}) => {
  await onboard(page, server);
  await page.getByRole("button", { name: "My Clinic", exact: true }).click();
  await page.getByRole("button", { name: "Clinic Documents", exact: true }).click();

  await page.getByRole("button", { name: "Upload Document", exact: true }).click();
  const evil =
    '<script>document.title = "XSS-TOKEN:" + sessionStorage.getItem("sessionToken");</' + "script>";
  await page.locator("#doc-file").setInputFiles({
    name: "evil.html",
    mimeType: "text/html",
    buffer: Buffer.from(evil),
  });
  await page.locator("#doc-name").fill("Evil Doc");
  await page.getByRole("button", { name: "Upload", exact: true }).click();
  await expect(page.getByText("Evil Doc").first()).toBeVisible();

  const popupPromise = context.waitForEvent("page", { timeout: 4000 }).catch(() => null);
  const downloadPromise = page.waitForEvent("download", { timeout: 4000 }).catch(() => null);
  await page.getByRole("button", { name: "Open", exact: true }).click();

  const download = await downloadPromise;
  const popup = await popupPromise;
  // With the fix the blank popup is closed and nothing navigates to the blob,
  // so the script never runs; a closed popup is itself a pass. Without the fix
  // the popup navigates to the HTML and its title becomes the exfiltrated token.
  if (popup && !popup.isClosed()) {
    await popup.waitForTimeout(500).catch(() => {});
  }
  for (const p of context.pages()) {
    if (p.isClosed()) continue;
    expect(await p.title().catch(() => "")).not.toContain("XSS-TOKEN:");
  }
  // The document is handed to the user as a download, not rendered inline.
  expect(download).not.toBeNull();
});
