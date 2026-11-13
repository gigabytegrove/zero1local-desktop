# Zero1Local Desktop — native Wails v3.0.0 build

Go 1.26.10 or later and native WebView prerequisites are required. This is source code, not a verified signed release.
The desktop runs in a native window with a system tray. Closing hides the window; the menu restores it or quits.

## Linux: Ubuntu 24.04+ or Debian 13+

```bash
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
git clone https://github.com/gigabytegrove/zero1local-desktop.git
cd zero1local-desktop
go test -race ./...
go vet ./...
cd desktop
go mod download
go build -trimpath -o zero1local-desktop .
./zero1local-desktop
```

Run on an actual Linux graphical session. Wails supports an alternative GTK3 build tag on supported legacy distributions with WebKit2GTK 4.1.

## Windows: PowerShell

Install Go 1.26.10, Git for Windows and the WebView2 Runtime (usually present).

```powershell
git clone https://github.com/gigabytegrove/zero1local-desktop.git
Set-Location zero1local-desktop
go test ./...
go vet ./...
Set-Location desktop
go mod download
go build -trimpath -o zero1local-desktop.exe .
.\zero1local-desktop.exe
```

## macOS: Terminal

Install Go 1.26.10, Git, and the Xcode Command Line Tools.

```bash
xcode-select -p
git clone https://github.com/gigabytegrove/zero1local-desktop.git
cd zero1local-desktop
go test ./...
go vet ./...
cd desktop
go mod download
go build -trimpath -o zero1local-desktop .
./zero1local-desktop
```

Raw macOS executables are not notarized .app bundles; installer packaging and signing still need testing.

## Zero1Local Desktop Beta 4 workflows

The desktop prompts for the NAS administrator username and password when adding an appliance, with a separate TOTP field when two-factor authentication is enabled. A new appliance is saved only after Zero1Local verifies that login. Signed-in NAS sessions are independent for each appliance and are not preserved when the desktop quits.

When using a plain HTTP NAS address, the app requires an explicit warning acknowledgement because the password would travel over the network unencrypted. HTTPS uses normal certificate validation. No administrator password or NAS session cookie is written to the desktop settings file.

The left navigation includes File Manager, Shared Folders, Mapped Drives, and authenticated live sections for System, Storage, Network, Containers & Docker, Hardware, Tasks, Users & Groups, Services, Backups, Protection Plans and Notifications. The File Manager supports NAS share-rooted browsing, folder creation, file upload/download, rename and Move to Recycle Bin (transfer limit 256 MiB).

**Important:** the NAS browser console remains the complete source of advanced administration. The desktop's NAS login and the system browser's NAS login are isolated sessions, so opening the full console can prompt for login again. SMB and NFS mounts use the operating system's distinct share credentials and access permissions.

## Boundaries

The Go companion binds to 127.0.0.1 with a randomized port and enforces origin, host, token, and CSP checks.
Management opens the original NAS UI through the OS browser until Wails remote-origin bridge isolation is verified.
OS mappings and credential prompts remain subject to platform support, NAS ACLs and identity mapping.
Official Zero1Local logos are embedded unchanged and a source-artwork crop is used for the native tray icon.
No GitHub Actions, background root agents, or NAS modifications.

Platform prerequisites: https://v3.wails.io/quick-start/installation/
