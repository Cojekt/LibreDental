import { test as base, expect, type Page } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import path from "node:path";
import { DatabaseSync } from "node:sqlite";

const binary = path.resolve(
  import.meta.dirname,
  process.env.LIBREDENTAL_E2E_BINARY ??
    `../../dist/libredental-server${process.platform === "win32" ? ".exe" : ""}`
);

// The server sits west of UTC and browsers in roundtrip.spec.ts far east of it, so a
// date-only value serialized as a local timestamp on either side lands on another day.
export const SERVER_TZ = "Pacific/Honolulu";

export const PROVIDER = { name: "E2E Dentist", pin: "8675" };

type Server = {
  url: string;
  query: <T>(db: "main" | "audit", sql: string, ...params: (string | number)[]) => T[];
  // Writes directly to the main database, for fixtures the current UI can no longer create.
  exec: (sql: string, ...params: (string | number)[]) => void;
};

function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = createServer();
    srv.once("error", reject);
    srv.listen(0, "127.0.0.1", () => {
      const { port } = srv.address() as { port: number };
      srv.close(() => resolve(port));
    });
  });
}

async function waitForServer(url: string, child: ChildProcess) {
  for (let i = 0; i < 100; i++) {
    if (child.exitCode !== null) throw new Error(`Server exited with code ${child.exitCode}`);
    try {
      if ((await fetch(url)).ok) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error("Server did not start within 10s");
}

// Each test gets its own server process and empty data directory, so tests are isolated and
// every run exercises migrations against a fresh install.
export const test = base.extend<{ server: Server }>({
  server: async ({}, use) => {
    const dir = await mkdtemp(path.join(tmpdir(), "libredental-e2e-"));
    const port = await freePort();
    const url = `http://127.0.0.1:${port}`;
    const child = spawn(binary, [], {
      env: {
        ...process.env,
        TZ: SERVER_TZ,
        XDG_CONFIG_HOME: dir,
        APPDATA: dir,
        LIBREDENTAL_HOST: "127.0.0.1",
        LIBREDENTAL_PORT: String(port),
      },
      stdio: ["ignore", "ignore", "pipe"],
    });
    let stderr = "";
    child.stderr!.on("data", (d) => (stderr += d));
    try {
      await waitForServer(url, child);
    } catch (e) {
      child.kill();
      throw new Error(`${(e as Error).message}\n${stderr}`);
    }

    const dataDir = path.join(dir, "LibreDental");
    const files = { main: "libredental.db", audit: "audit.db" };
    const query: Server["query"] = (db, sql, ...params) => {
      const conn = new DatabaseSync(path.join(dataDir, files[db]), { readOnly: true });
      try {
        return conn.prepare(sql).all(...params) as any;
      } finally {
        conn.close();
      }
    };

    const exec: Server["exec"] = (sql, ...params) => {
      const conn = new DatabaseSync(path.join(dataDir, files.main));
      try {
        conn.prepare(sql).run(...params);
      } finally {
        conn.close();
      }
    };

    await use({ url, query, exec });

    await new Promise((resolve) => {
      child.once("exit", resolve);
      child.kill();
    });
    await rm(dir, { recursive: true, force: true });
  },
  page: async ({ page }, use) => {
    const errors: string[] = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await use(page);
    expect(errors, "uncaught errors in the page").toEqual([]);
  },
});

export { expect };
export type { Page };

export async function onboard(page: Page, server: Server) {
  await page.goto(server.url);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await page.locator("#onboard-prov-name").fill(PROVIDER.name);
  await page.locator("#onboard-prov-pin").fill(PROVIDER.pin);
  await page.locator("#onboard-prov-pin-confirm").fill(PROVIDER.pin);
  await page.getByRole("button", { name: "Save & Complete Setup", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Welcome to LibreDental!" })).toBeHidden();
}

export async function createPatient(page: Page, first: string, last: string, dob = "1990-01-15") {
  await page.getByRole("button", { name: "Patients", exact: true }).click();
  await page.getByRole("button", { name: "New Patient", exact: true }).click();
  await page.locator("#fname").fill(first);
  await page.locator("#lname").fill(last);
  // Empty date inputs render as text until focused, to show the country date format.
  await page.locator("#dob").focus();
  await page.locator("#dob").fill(dob);
  await page.locator("#phone").fill("2025550100");
  await page.getByRole("button", { name: "Save Patient Record", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Add New Patient" })).toBeHidden();
}
