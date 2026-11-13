import { defineConfig } from "@playwright/test";

// E2E tests drive the real server binary (see e2e/fixtures.ts) and only cover what the Go
// service tests and the mocked-binding Vitest tests cannot: main.go wiring, the Wails binding
// layer (JSON shapes, dates, cents, error codes) and persistence across that boundary.
export default defineConfig({
  testDir: "e2e",
  testMatch: "*.spec.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? "github" : "list",
  use: {
    locale: "en-US",
    trace: "retain-on-failure",
  },
});
