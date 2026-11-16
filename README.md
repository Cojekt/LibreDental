<p align="center">
  <img src="frontend/public/sourceicon.svg" alt="LibreDental logo" width="128" height="128">
</p>

<h1 align="center">LibreDental</h1>

<p align="center"><strong>Open-Source Modern Dental Practice Management System</strong></p>

[![Code Review & Build (CI)](https://github.com/Cojekt/libredental/actions/workflows/ci.yml/badge.svg)](https://github.com/Cojekt/libredental/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Cojekt/libredental?include_prereleases)](https://github.com/Cojekt/libredental/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Cojekt/libredental)](https://github.com/Cojekt/libredental)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**LibreDental** is a fast, resilient, open-source dental practice management platform. Built as a native desktop app for Linux and Windows with local-first reliability, LibreDental gives practices total freedom, security, and ownership over their patient data.

## What is LibreDental?

LibreDental addresses the growing need for modern, open-source dental practice management software. Built natively for desktop performance, it combines zero-latency local operation with modern web interface user experience.

### Core Capabilities

* **Patient Management:** Demographics, clinical history, insurance tracking, and patient documents and X-rays (including DICOM).
* **Interactive Odontogram & Restorative Charting:** Visual tooth mapping with surface-level restoration history and treatment planning.
* **Scheduling & Appointments:** Multi-provider, multi-operatory calendar with day/week/month/agenda views, status tracking, and appointment reminders over SMS and email.
* **Billing & Ledger:** Patient accounts and payments (including write-offs), fee schedules and insurance fee overrides, insurance claims (X12 837D, with optional clearinghouse submission), and analytics.
* **Audit Trail:** A separate audit log recording who changed patient, clinical, and billing data and when.
* **Headless / Server Mode:** Option to run centrally on a local clinic server while lightweight client instances connect over the local network.
* **Local-First Data Storage:** Uses embedded SQLite files in the OS user config directory for zero-latency local operation and simple file-based backups.

> *Note: Features evolve rapidly across active Alpha iterations. Check the [Releases page](https://github.com/Cojekt/libredental/releases) for the latest release notes.*

## Our Philosophy & Commitment

### 100% Free & Open-Source Core

* **Forever Free:** The core LibreDental codebase will **always** remain 100% free and open-source under the Apache 2.0 license.
* **Our Commitment:** We solemnly promise **never to lock open-source software behind proprietary update fees**, restrict self-hosting capabilities, require mandatory licensing keys, or relicense the core project under restrictive terms.
* **No Vendor Lock-In:** Your practice data belongs entirely to you. Local installations will never be throttled, feature-gated, or dependent on third-party cloud servers.

## Tech Stack & Architecture

* **Framework:** [Wails v3](https://v3.wails.io) (Native OS windowing + IPC)
* **Backend:** Go 1.26
* **Frontend:** Svelte 5 + Vite + TypeScript + Tailwind CSS v4, i18n via Paraglide
* **Database:** Embedded SQLite via pure-Go `modernc.org/sqlite` (zero external database setup needed), migrations via goose + Atlas

## Getting Started & Contributing

LibreDental is built as a **[Wails v3](https://v3.wails.io)** application combining a Go backend with a modern Svelte 5 / Vite frontend. We welcome contributions from developers, dental professionals, UI/UX designers, and open-source advocates!

### Prerequisites

* **Go 1.26+**
* **Node.js 22+** (npm, pnpm, yarn, or bun)
* **Other dependencies**:
  * **Linux**: `build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev`
  * **Windows**: [WebView2 runtime](https://developer.microsoft.com/microsoft-edge/webview2/) (preinstalled on Windows 10/11)
* **Supported desktop platforms**: Linux and Windows. macOS desktop builds are not currently supported.
* **Global CLI Tools**:

  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  go install github.com/go-task/task/v3/cmd/task@latest
  ```

### Quick Start & Development Workflow

1. **Fork & Clone**: Fork the repository and clone it to your local machine:

   ```bash
   git clone https://github.com/Cojekt/libredental.git && cd libredental
   ```

2. **Install Dependencies**:

   ```bash
   go mod download
   cd frontend && npm install && cd ..
   ```

   > **IMPORTANT** \
   > `wails3@latest` above may land ahead of the version this repo's `go.mod` actually pins, and a mismatched CLI/runtime is a common source of confusing "method not binding" errors. After cloning, run `task deps:sync-wails` (see [Command Reference](#command-reference)) to align the `wails3` CLI and `@wailsio/runtime` to the exact version `go.mod` already requires.

3. **Create a Feature Branch**:

   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Development**: Always use **`task desktop`** for active development. It handles live-reloading for frontend assets, automatic Go binding generation, and Wails runtime synchronization.
5. **Code Formatting**: Before opening a Pull Request, run:

   ```bash
   task format
   ```

   > **IMPORTANT** \
   > Running `task format` runs both `gofmt` for Go backend files and Prettier for frontend files. PRs with unformatted code will fail automated CI formatting checks.

6. **Submit a Pull Request**: Push your branch to your fork and submit a PR against `main` with a description of your changes and test steps.

## Command Reference

All project build, development, and formatting tasks are managed through [Task](https://taskfile.dev).

| Command | Description |
| :--- | :--- |
| `task desktop` | Starts Wails v3 live-reload mode for the desktop application. |
| `task server` | Builds and runs the application in server mode (HTTP server at <http://localhost:4242>). |
| `task artifact` | Builds all local artifacts (desktop binary, server binary, and demo data). |
| `task demo` | Generates a standalone pre-populated save archive (`dist/libredental-demo-data.zip`) for testing. |
| `task format` | Formats Go backend files with `gofmt -w -s` and frontend code with Prettier. **Must be run before creating PRs.** |
| `task test` | Runs the Go tests, frontend type/Svelte check, Vitest component tests, and Playwright end-to-end tests (`task e2e`). |
| `task e2e` | Builds the server binary and runs the Playwright end-to-end tests against it. |
| `task check` | Runs the same quality checks as CI (`gofmt -l`, `go vet`, Go tests, bindings generation, Prettier `format:check`, `svelte-check`, Vitest, and end-to-end tests). Run it before opening a PR to catch CI failures locally. |
| `task build` / `task package` | Production desktop build / installers for the current OS (Linux or Windows). |
| `task deps:update` | Safely updates Go modules and frontend npm packages together, keeping the `wails3` CLI and `@wailsio/runtime` pinned to the same Wails version, then regenerates bindings and verifies the repo still builds and tests pass. |
| `task deps:sync-wails` | Aligns the installed `wails3` CLI and `@wailsio/runtime` to whatever Wails version `go.mod` already pins, without upgrading any dependencies. Useful right after cloning. |

## License

LibreDental is licensed under the [Apache License, Version 2.0](LICENSE).

*LibreDental™ is a trademark claiming common law rights. The LibreDental name and logo are trademarks of the LibreDental project.*
