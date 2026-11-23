import { test, expect } from "./fixtures";

// The Wails runtime Application object (Quit, Hide, Show) is not a bound service, so it
// skips session checks; a same-origin Quit used to stop the server for every workstation.
test("server refuses runtime Application calls and keeps serving", async ({
  page,
  request,
  server,
}) => {
  const quit = await request.post(`${server.url}/wails/runtime`, {
    headers: { Origin: server.url },
    data: { object: 2, method: 2 },
  });
  expect(quit.status()).toBe(403);

  const health = await request.get(`${server.url}/health`);
  expect(health.status()).toBe(200);

  await page.goto(server.url);
  await expect(page.getByRole("button", { name: "Continue", exact: true })).toBeVisible();
});
