# LibreDental

> **Open-Source Modern Dental Practice Management System**

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-brightgreen.svg)]()
[![Stack](https://img.shields.io/badge/Stack-Wails_v3_|Go_|Svelte_5-007ACC.svg)](#tech-stack--architecture)

**LibreDental** is a fast, resilient, open-source dental practice management platform. Built natively for cross-platform desktop performance with local-first reliability, LibreDental™ gives practices total freedom, security, and ownership over their patient data.

---

## What is LibreDental?

LibreDental addresses the growing need for modern, open-source dental practice management software. Built natively for desktop performance, it combines zero-latency local operation with modern web interface user experience.

### Core Capabilities
* **Patient Management:** Demographics, clinical history, insurance tracking, and patient communication records.
* **Interactive Odontogram & Restorative Charting:** Visual tooth mapping with surface-level restoration history and treatment planning.
* **Scheduling & Appointments:** Multi-provider, multi-operatory calendar with real-time status updates and conflict checks.
* **Billing & Ledger:** Patient accounts, fee schedules, insurance claim management, line-item adjustments, and reporting.
* **Headless / Server Mode:** Option to run centrally on a local clinic server while lightweight client instances connect over the local network.
* **Local-First Data Storage:** Uses SQLite for local zero-latency operation and simple file-based backup workflows.

> *Note: Features evolve rapidly across active Alpha iterations. Check the [Releases page](https://github.com/Cojekt/libredental/releases) for the latest release notes.*
---

## Our Philosophy & Commitment

### 100% Free & Open-Source Core

* **Forever Free:** The core LibreDental™ codebase will **always** remain 100% free and open-source under the Apache 2.0 license.
* **Our Commitment:** We solemnly promise **never to lock open-source software behind proprietary update fees**, restrict self-hosting capabilities, require mandatory licensing keys, or relicense the core project under restrictive terms.
* **No Vendor Lock-In:** Your practice data belongs entirely to you. Local installations will never be throttled, feature-gated, or dependent on third-party cloud servers.

---

## Tech Stack & Architecture

* **Framework:** [Wails v3](https://v3.wails.io) (Native OS windowing + IPC)
* **Backend:** Go 1.22+
* **Frontend:** Svelte 5 + Vite + TypeScript
* **Database:** Embedded SQLite (zero external database setup needed)

---

## Getting Started & Contributing

LibreDental™ is built as a **[Wails v3](https://v3.wails.io)** application combining a Go backend with a modern Svelte 5 / Vite frontend. We welcome contributions from developers, dental professionals, UI/UX designers, and open-source advocates!

### Prerequisites

- **Go 1.22+**
- **Node.js 20+** (npm, pnpm, yarn, or bun)
- **Other dependencies**:
  - **Linux**: `build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev`
  - **macOS**: Xcode Command Line Tools (`xcode-select --install`)
  - **Windows**: C compiler (MinGW-w64 or MSVC)
- **Global CLI Tools**:

  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  go install github.com/go-task/task/v3/cmd/task@latest
  ```

### Quick Start & Development Workflow

1. **Fork & Clone**: Fork the repository and clone it to your local machine:

   ```bash
   git clone https://github.com/LibreDental/libredental.git && cd libredental
   ```

2. **Install Dependencies**:

   ```bash
   go mod download
   cd frontend && npm install && cd ..
   ```

3. **Create a Feature Branch**:

   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Development**: Always use **`task dev`** for active development. It handles live-reloading for frontend assets, automatic Go binding generation, and Wails runtime synchronization.
5. **Code Formatting**: Before opening a Pull Request, run:

   ```bash
   task format
   ```

   > **IMPORTANT** \
   > Running `task format` runs both `gofmt` for Go backend files and Prettier for frontend files. PRs with unformatted code will fail automated CI formatting checks.

6. **Submit a Pull Request**: Push your branch to your fork and submit a PR against `main` with a description of your changes and test steps.

---

## Command Reference

All project build, development, and formatting tasks are managed through [Task](https://taskfile.dev).

| Command | Description |
| :--- | :--- |
| `task dev` | **Primary Development Command.** Starts Wails v3 live-reload mode with Vite hot-module replacement and Go hot-rebuilding. |
| `task format` | Formats Go backend files with `gofmt -w -s` and frontend code with Prettier. **Must be run before creating PRs.** |
| `task build` | Builds the production desktop binary for your OS in `./bin/libredental`. |
| `task build:server` | Compiles the headless HTTP server mode binary (`./bin/libredental-server`). |
| `task demo` | Generates a pre-populated SQLite demo database (`libredental.db`) for testing. |
| `task package` | Creates production OS installer packages (`.deb`, `.rpm`, `AppImage`, `.dmg`, `.exe`). |
| `task run` | Executes the compiled application binary from `./bin/libredental`. |

---

## License

LibreDental is licensed under the [Apache License, Version 2.0](LICENSE).

_LibreDental™ is a trademark claiming common law rights. The LibreDental name and logo are trademarks of the LibreDental project._
