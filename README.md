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

* **Everything is free right now**, and the core LibreDental™ software will **always** remain 100% free and open-source under the Apache 2.0 license.
* **Our Commitment:** We solemnly promise **never to do what Open Dental did**. We will never lock open-source software behind proprietary update fees, restrict self-hosting capabilities, enforce mandatory subscription keys, or rug-pull the community with restrictive relicensing.

### Future Sustainability Plan

To support long-term development, maintenance, and support:

* Far down the road, funds *may* be raised by offering an **optional online hosted (SaaS) service** for practices that prefer a managed cloud setup without maintaining local servers.
* This hosted service is **a while away** and will always remain completely optional. Self-hosted and local installations will never be crippled or paywalled.

---

## 🛡️ Architecture & HIPAA Compliance

LibreDental™ is designed from the ground up using **Wails v3** (Go + modern Web frontend) to support a **100% local-first desktop application mode**.

### Local-First Security & HIPAA Advantages

Running LibreDental™ fully on the dentist's computer or local practice network (LAN) provides significant advantages for HIPAA (Health Insurance Portability and Accountability Act) compliance:

* **Zero Cloud ePHI Risk:** Protected Health Information (ePHI) stays entirely on the practice's local machine or local server. No third-party cloud servers receive or store patient records.
* **No Vendor BAA Required for Storage:** Because LibreDental operates as local software with no vendor telemetry or remote data collection, practices maintain 100% custody of their data.
* **Built-in Compliance Controls:**
  * **Encryption at Rest:** Support for SQLCipher / local disk encryption for all stored patient data and x-rays.
  * **Role-Based Access Control (RBAC):** Granular user permissions and auto-lock screen timeouts.
  * **Immutable Audit Logs:** Full tracking of ePHI access, modifications, exports, and deletions.
  * **Local Encrypted Backups:** Automated daily backups to local or practice-controlled storage.

---

## 🚀 Getting Started

LibreDental™ is a standard **[Wails v3](https://v3.wails.io)** application (Go 1.22+ & Svelte 5 / Vite).

### Prerequisites

* **Go 1.22+**, **Node.js 20+**, and a **C Compiler / WebKit** (Linux: `build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev` | macOS: `xcode-select --install` | Windows: MinGW/MSVC).
* **CLI Tools**:

  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  go install github.com/go-task/task/v3/cmd/task@latest
  ```

### Quick Start

```bash
# Clone repository & install dependencies
git clone https://github.com/LibreDental/libredental.git && cd libredental
go mod download
cd frontend && npm install && cd ..

# Run live-reload development mode
task dev # (or: wails3 dev)

# Production build & packaging
task build       # Build desktop binary to ./bin/libredental
task package     # Create OS installer packages (.deb, .rpm, AppImage, etc.)
task run:server  # Run headless HTTP server mode
```

## 🤝 Contributing

We welcome contributions from developers, dental professionals, and open-source enthusiasts! Please feel free to open issues, submit pull requests, or start discussions.

---

## 📜 License

LibreDental™ is licensed under the [Apache License, Version 2.0](LICENSE).

*LibreDental™ is a trademark claiming common law rights. The LibreDental name and logo are trademarks of the LibreDental project.*
