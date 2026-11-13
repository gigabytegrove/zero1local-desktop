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


## Beta 6 — NAS-scoped workspaces and remembered sign-in

**My appliances is the landing screen.** Click an individual NAS to open its own Overview, File Manager, Shared Folders, Mapped Drives, and management sections. The NAS selector and full management navigation only appear inside a selected appliance workspace. Switching NAS devices clears the currently shown share selection. Mapped Drives lists only mappings belonging to the selected NAS.

**Sign-in is a choice on each NAS, not a global password setting.** By default passwords remain memory-only and signing in lasts only for the current application session. Select **Keep me signed in to this NAS** to explicitly store that NAS's credentials and current (up to eight-hour) session cookie in the native operating-system vault. On startup the desktop first tries to resume the original NAS session using \`/api/session\` and then attempts password reauthentication if the cookie was revoked or expired. TOTP codes are never stored. If the NAS demands two-factor verification again, the user is asked to sign in; the desktop will never silently bypass MFA. Signing out erases the remembered vault entry; **Stop remembering** removes the credential while leaving the active session connected. Passwords and cookies do not enter \`settings.json\`, frontend state, logs, or GitHub.

**Each Windows SMB share has its own independent Reconnect option.** Windows handles persistent SMB mappings and any Windows credentials; these are unrelated to the NAS administrator account. Linux and macOS share auto-reconnect are not advertised as implemented: the respective desktop/system file manager controls their credentials and reconnection.

Native OS vaults: Windows Credential Manager, macOS Keychain, Linux Secret Service. If a vault is unavailable/locked, opt-in persistent sign-in fails clearly rather than silently saving plaintext. Linux Secret Service requires an available unlocked keyring. The original NAS server and its eight-hour session policy are unchanged.

**Verification gate:** Node frontend regression suites, Go race and integration tests, Wails GTK tests/vet, cross-platform compilation, architecture checks, checksum verification and exact GitHub release assets. End-to-end native desktop tests and real NAS session/mount acceptance remain required before declaring stable.

## Beta 7: built-in update lifecycle

- **Settings → Software updates → Check for updates** queries public GitHub Releases and shows either the latest compatible release, an up-to-date result, or a useful error. Checking does **not** install.
- **Install update** appears only for a newer compatible release. The user confirms the update; Debian installs ask the OS for administrator authorization through PolicyKit. The app never handles the administrator password.
- Linux `.deb` installs remain managed by Debian's APT/dpkg. Verified downloads go to a private temporary location, **not** `~/Downloads`. The installation stops if SHA-256, package metadata, architecture or installed source is inconsistent.
- **Restart Desktop now** appears only after Debian's package manager has installed and verified the requested version. The restart helper waits for the old single-instance lock to clear.
- The native Windows/Linux portable update path still uses Wails' official GitHub provider and dedicated checksum-verified single-entry updater archives.
- **Installer prerequisites:** desktop Linux packages should depend on `pkexec` (PolicyKit graphical authorization) as well as GTK and WebKit. On systems without a running polkit authentication agent, a user-facing error is reported instead of silently attempting root installation.
- Previous Beta 6 Debian binaries intentionally only downloaded a `.deb`. They require **one** traditional package upgrade to Beta 7; subsequent updates use the integrated installer.
- No NAS firmware or server updates occur from the desktop updater.

## Boundaries

The Go companion binds to 127.0.0.1 with a randomized port and enforces origin, host, token, and CSP checks.
Management opens the original NAS UI through the OS browser until Wails remote-origin bridge isolation is verified.
OS mappings and credential prompts remain subject to platform support, NAS ACLs and identity mapping.
Official Zero1Local logos are embedded unchanged and a source-artwork crop is used for the native tray icon.
No GitHub Actions, background root agents, or NAS modifications.

Platform prerequisites: https://v3.wails.io/quick-start/installation/
