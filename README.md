# LibreDental™

> **Open-Source Modern Dental Practice Management System**

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Early_Development-orange.svg)](#project-status)

**LibreDental™** is a modern, open-source dental practice management software designed to give dental practices complete freedom, security, and ownership over their software and patient data.

---

## ⚠️ Project Status: Early Development

LibreDental™ is currently in **early active development**. We are laying the architectural foundation for a fast, resilient, and user-friendly desktop app. Features, APIs, and schemas are subject to rapid evolution as we build toward our initial alpha release.

---

## 💡 Our Philosophy & Commitment

### 100% Free & Open Source Core

- **Everything is free right now**, and the core LibreDental™ software will **always** remain 100% free and open-source under the Apache 2.0 license.
- **Our Commitment:** We solemnly promise **never to do what Open Dental did**. We will never lock open-source software behind proprietary update fees, restrict self-hosting capabilities, enforce mandatory subscription keys, or rug-pull the community with restrictive relicensing.

### Future Sustainability Plan

To support long-term development, maintenance, and support:

- Far down the road, funds _may_ be raised by offering an **optional online hosted (SaaS) service** for practices that prefer a managed cloud setup without maintaining local servers.
- This hosted service is **a while away** and will always remain completely optional. Self-hosted and local installations will never be crippled or paywalled.

---

## 🛡️ Architecture & HIPAA Compliance

LibreDental™ is designed from the ground up using **Wails v3** (Go + modern Web frontend) to support a **100% local-first desktop application mode**.

### Local-First Security & HIPAA Advantages

Running LibreDental™ fully on the dentist's computer or local practice network (LAN) provides significant advantages for HIPAA (Health Insurance Portability and Accountability Act) compliance:

- **Zero Cloud ePHI Risk:** Protected Health Information (ePHI) stays entirely on the practice's local machine or local server. No third-party cloud servers receive or store patient records.
- **No Vendor BAA Required for Storage:** Because LibreDental operates as local software with no vendor telemetry or remote data collection, practices maintain 100% custody of their data.
- **Built-in Compliance Controls:**
  - **Encryption at Rest:** Support for SQLCipher / local disk encryption for all stored patient data and x-rays.
  - **Role-Based Access Control (RBAC):** Granular user permissions and auto-lock screen timeouts.
  - **Immutable Audit Logs:** Full tracking of ePHI access, modifications, exports, and deletions.
  - **Local Encrypted Backups:** Automated daily backups to local or practice-controlled storage.

---

## 🚀 Getting Started & Contributing

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

   > [!IMPORTANT]
   > Running `task format` runs both `gofmt` for Go backend files and Prettier for frontend files. PRs with unformatted code will fail automated CI formatting checks.

6. **Submit a Pull Request**: Push your branch to your fork and submit a PR against `main` with a description of your changes and test steps.

---

## 🛠️ Command Reference

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

## 📜 License

LibreDental™ is licensed under the [Apache License, Version 2.0](LICENSE).

_LibreDental™ is a trademark claiming common law rights. The LibreDental name and logo are trademarks of the LibreDental project._
