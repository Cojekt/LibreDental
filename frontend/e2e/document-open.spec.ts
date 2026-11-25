import { test, expect, onboard } from "./fixtures";

// A stored document must never run as script in the app's origin. In browser
// (server) mode, "Open" previously navigated a same-origin blob: tab to the
// document's content type, so an uploaded document whose type renders as active
// content executed with access to the signed-in session (token, bound methods).
// Both text/html and image/svg+xml can do this, so both must download instead.
const activeTypes = [
  { name: "evil.html", mimeType: "text/html", body: "<script>REPLACE</" + "script>" },
  {
    name: "evil.svg",
    mimeType: "image/svg+xml",
    body: '<svg xmlns="http://www.w3.org/2000/svg"><script>REPLACE</' + "script></svg>",
  },
];

for (const kind of activeTypes) {
  test(`opening a ${kind.mimeType} document does not execute it in the app origin`, async ({
    page,
    context,
    server,
  }) => {
    await onboard(page, server);
    await page.getByRole("button", { name: "My Clinic", exact: true }).click();
    await page.getByRole("button", { name: "Clinic Documents", exact: true }).click();

    await page.getByRole("button", { name: "Upload Document", exact: true }).click();
    const payload = kind.body.replace(
      "REPLACE",
      'document.title = "XSS-TOKEN:" + sessionStorage.getItem("sessionToken");'
    );
    await page.locator("#doc-file").setInputFiles({
      name: kind.name,
      mimeType: kind.mimeType,
      buffer: Buffer.from(payload),
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
    // so the script never runs. Poll (rather than sleep) so that if the old
    // behaviour navigated and executed, the canary title is still caught.
    if (popup) {
      await expect
        .poll(async () => (popup.isClosed() ? "closed" : await popup.title().catch(() => "")), {
          timeout: 2000,
        })
        .not.toContain("XSS-TOKEN:");
    }
    for (const p of context.pages()) {
      if (p.isClosed()) continue;
      expect(await p.title().catch(() => "")).not.toContain("XSS-TOKEN:");
    }
    // The document is handed to the user as a download, not rendered inline.
    expect(download).not.toBeNull();
  });
}
