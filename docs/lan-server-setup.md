# LibreDental — LAN Server Setup Guide

> **Audience:** IT staff & system administrators.  
> **Supported OS:** Windows 10/11 & Linux (Ubuntu 22.04+).

## 1. Overview & Architecture

LibreDental offers two separate binaries for two distinct deployment models:

- **Desktop App (`libredental`):** Standalone single-machine desktop application for solo practices operating on a single PC with zero networking.
- **LAN Server (`libredental-server`):** Headless background service running on one central server PC that serves the web application over HTTP to browser clients.

> **Note:** Client workstations connect to the server using standard web browsers (Chrome, Edge, Firefox). Remote workstation desktop binaries cannot connect over the network to a server instance. However, running the desktop app locally on the server PC itself is fully supported alongside the server process (safely sharing `libredental.db` via SQLite WAL mode).

## 2. Server Requirements & OS Setup

### Requirements

- **CPU / RAM:** Dual-core (Quad-core recommended), 4 GB RAM.
- **OS:** Windows 10/11 or Ubuntu 22.04+ LTS.
- **Network:** Wired Ethernet with a **Static IP** (e.g. `192.168.1.100`, assigned via router DHCP reservation or OS network settings).

### Prevent System Sleep

The Server PC must **never enter sleep/suspend mode** while the practice is open (the display monitor may turn off).

- **Windows:** Settings → System → Power & Sleep → set **Sleep** to **Never**.
- **Linux:**

  ```bash
  sudo systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target
  ```

## 3. Server Installation & Firewall

Download `libredental-server.exe` (Windows) or `libredental-server` (Linux) from the LibreDental releases page.

### Linux (Ubuntu)

1. **Install Binary:**

   ```bash
   sudo mkdir -p /opt/libredental
   sudo cp libredental-server /opt/libredental/
   sudo chmod +x /opt/libredental/libredental-server
   ```

2. **Firewall (UFW):** Allow port 4242 from local LAN only:

   ```bash
   sudo ufw allow from 192.168.1.0/24 to any port 4242
   ```

3. **Autostart (systemd):** Create `/etc/systemd/system/libredental.service`:

   ```ini
   [Unit]
   Description=LibreDental LAN Server
   After=network.target

   [Service]
   Type=simple
   User=YOUR_USERNAME
   ExecStart=/opt/libredental/libredental-server
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target
   ```

   Enable and start:

   ```bash
   sudo systemctl daemon-reload && sudo systemctl enable --now libredental
   ```

### Windows

1. **Install Binary:** Copy `libredental-server.exe` to `C:\LibreDental\libredental-server.exe`.
2. **Firewall (PowerShell as Admin):**

   ```powershell
   New-NetFirewallRule -DisplayName "LibreDental LAN" -Direction Inbound -Protocol TCP -LocalPort 4242 -RemoteAddress 192.168.1.0/24 -Action Allow
   ```

3. **Autostart (Task Scheduler):**
   Create a task named `LibreDental Server` triggered **At startup**, running with highest privileges, launching `C:\LibreDental\libredental-server.exe`.

## 4. Workstation Connection & Data Backups

### Client Workstation Connection

On any computer on the local network, open a web browser and navigate to:

<http://192.168.1.100:4242>

*(Replace `192.168.1.100` with your server's static IP. Bookmark for easy access).*

### Data Location & Daily Backups

All patient data resides in a single SQLite database on the server PC:

- **Linux:** `~/.config/LibreDental/libredental.db`
- **Windows:** `C:\Users\<Username>\AppData\Roaming\LibreDental\libredental.db`

**Linux Hot Backup Script (`/opt/libredental/backup.sh`):**

```bash
#!/bin/bash
BACKUP_DIR="/mnt/backup/libredental"
mkdir -p "$BACKUP_DIR"
sqlite3 "$HOME/.config/LibreDental/libredental.db" ".backup '$BACKUP_DIR/libredental-$(date +%Y-%m-%d).db'"
find "$BACKUP_DIR" -name "libredental-*.db" -mtime +30 -delete
```

Schedule via `crontab -e`: `0 23 * * * /opt/libredental/backup.sh`

## 5. Quick Reference & Troubleshooting

- **Server Self-Access:** If the LAN network switch fails, the Server PC can still access the system at `http://localhost:4242`.
- **Concurrent Server & Desktop Execution:** Running the Desktop App (`libredental`) alongside the LAN Server (`libredental-server`) on the server machine under the same OS user account is supported and safe. Because LibreDental uses SQLite WAL mode (`journal_mode=WAL` with `busy_timeout=5000`), both binaries safely share access to `libredental.db` without data corruption.
- **Host & Port Customization:** The server binds to `0.0.0.0` (all interfaces) on port `4242` by default. Set `LIBREDENTAL_PORT=8080` or `LIBREDENTAL_HOST=127.0.0.1` (or a specific IP like `192.168.1.100`) to customize.
- **Service Management (Linux):** `sudo systemctl [status|restart|stop] libredental`
