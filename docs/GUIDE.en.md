# ftpsimp User Guide

> **Language / Язык:** English | [Русский](https://github.com/ynderboi/ftpsimp/blob/master/docs/GUIDE.md)

Share files over Wi‑Fi on your local network: a mini server runs on a PC or phone, and other devices open a Windows 98–style browser file explorer.

Current Windows release: **1.1.2** (`ftpsimp-setup-1.1.2.exe`).  
Current Android release: **1.1.1** (`ftpsimp-1.1.1.apk`).

## Contents

1. [Requirements](#requirements)
2. [Windows — install and run](#windows--install-and-run)
3. [Console control panel (TUI)](#console-control-panel-tui)
4. [Web explorer](#web-explorer)
5. [Android](#android)
6. [Security](#security)
7. [Troubleshooting](#troubleshooting)

---

## Requirements

- Host and client on the **same Wi‑Fi / LAN**
- A normal browser on the client (Chrome, Safari, Firefox, etc.)
- Default port: **8080** (must be free)

Do not expose the ftpsimp port to the internet (port forwarding / VPS) — this is a LAN tool.

---

## Windows — install and run

### Install

1. Download **`ftpsimp-setup-1.1.2.exe`** from [Releases](https://github.com/ynderboi/ftpsimp/releases).
2. Run the installer (Russian / English).
3. Optionally enable the desktop shortcut.
4. After install, launch **ftpsimp** from the Start menu.

### First run

1. A console window opens with a large **FTPSIMP** banner and a status panel.
2. Note the **PIN** line (usually 6 digits) and an address like `http://192.168.x.x:8080`.
3. On a phone / another PC on the same network, open that address in a browser.
4. Enter the PIN → the file explorer opens.

### Launch flags (optional)

If you run `ftpsimp.exe` manually:

| Flag | Purpose |
|------|---------|
| `-port 8080` | HTTP port |
| `-dir "C:\path"` | Shared folder |
| `-pin 123456` | Fixed PIN |
| `-readonly` | Browse and download only |
| `-open` | No PIN (trusted networks only!) |
| `-plain` | Plain log without the interactive panel |

Settings are stored in `%AppData%\ftpsimp\config.json`.

---

## Console control panel (TUI)

After startup, these hotkeys are available in the terminal:

| Key | Action |
|-----|--------|
| **S** / Space | **Start / Stop** the HTTP server (without quitting) |
| **1** | Change the shared root folder |
| **2** | Toggle read‑only |
| **3** | Toggle PIN authentication |
| **4** | Generate a new PIN (existing sessions are cleared) |
| **5** | Set PIN manually |
| **6** | Refresh LAN address list |
| **Q** | Quit the app (asks for Y/N confirmation) |

The panel shows: STATUS (RUNNING/STOPPED), ROOT, PORT, MODE, AUTH, PIN, SESSIONS, URL.

---

## Web explorer

Windows 98–style UI.

### Sign-in

- If auth is on, a **Log On** window asks for the PIN.
- Read the PIN on the host (Windows console or Android app screen).
- On phones, the login window stays a compact dialog; the PIN field uses a numeric keyboard.

### Features

- Browse folders and download files
- Upload (button / drag‑and‑drop), create folders, delete
- **Refresh** (menu or ↻) — reload the current folder
- **Address** field — real filesystem path on the host
- Language: **Рус / Eng** (menu and taskbar tray)
- Windows can be moved, resized, minimized / maximized
- **Settings** → change root folder: **host only** (phones cannot change the path)

### Read‑only

If the host is in read‑only mode:

- Downloads still work
- Upload or mkdir attempts show a warning

---

## Android

### Install the APK

1. Download **`ftpsimp-1.1.1.apk`** from [Releases](https://github.com/ynderboi/ftpsimp/releases).
2. Allow installs from unknown sources (for your browser / file manager).
3. Install the APK and open **ftpsimp**.

### Usage

1. Pick a folder to share (or use the default).
2. Tap **Start server**.
3. The screen shows:
   - addresses like `http://…:8080`
   - the **PIN** for other devices
4. On a second device, open the address in a browser and enter the PIN.
5. To stop — tap **Stop** (or use the notification).

Changing the root folder on Android is **only in the app**, not via web Settings.

---

## Security

| Mechanism | Behavior |
|-----------|----------|
| PIN + session | Without a PIN, the API is closed (except login/status) |
| Change root | Host only |
| Read‑only | Blocks upload / mkdir / delete |
| `-open` / Auth OFF | Anyone on the LAN has full access — trusted networks only |
| HTTP | Traffic is not encrypted — intended for home LAN |

Recommendations:

- Keep **Auth ON**
- Do not use `-open` on public Wi‑Fi
- Do not forward the port to the internet

---

## Troubleshooting

**Address does not open from the phone**  
Check that you share one Wi‑Fi network, the server is RUNNING, Windows Firewall is not blocking port 8080, and the address is current (press **6** in the TUI).

**Wrong PIN**  
Use the current PIN on the host. After **Rotate PIN (4)**, old sessions are invalid.

**PIN field looks broken on a phone**  
Update the host to **1.1.2+** and hard‑refresh the browser page. The login window should stay a compact dialog, not full‑screen.

**Cannot change the root folder from the browser**  
Open the UI on the host itself (localhost / that machine’s LAN IP), or change the path in the TUI (**1**) / in the Android app.

**Port already in use**  
Start with another port: `ftpsimp.exe -port 8081`.

**Android says “choose a folder first”**  
Pick an accessible folder with the folder picker, then start the server again.
