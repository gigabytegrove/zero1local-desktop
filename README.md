# Zero1Local Desktop

A genuine Go/Wails native desktop companion for the existing Zero1Local NAS platform on Windows, Linux and macOS. The **original Zero1Local light/dark design**, icons and artwork are the design baseline—not a new or separately branded NAS dashboard.

## Implemented in source

- Native Wails v3 desktop window with system tray (Open, Hide, Quit); closing the window hides the workspace, leaving the companion running.
- Exact original Zero1Local light/dark PNG logo assets copied from the Zero1Local source repository.
- Desktop application shell with 232px left navigation rail, 60px topbar, familiar Zero1Local blue palette, light/dark/system theme setting and responsive tables.
- Current local LAN /24 discovery and manual appliance URLs; verifies the existing Zero1Local /healthz and /api/status service responses.
- Multiple saved NAS endpoints, version information and explicit removal; permissions and original NAS software are unchanged.
- Go localhost HTTP service with random local port, origin/host/token checks, restrictive CSP, small JSON limits and user-only preferences.
- OS share mapping requests: Windows native SMB drive letter, Linux GVfs SMB, Linux NFSv3 using pkexec, and macOS Finder SMB URLs.
- Administrator credential prompt when enrolling a NAS (username, password and optional TOTP). The device is saved **only after Zero1Local's real /api/login verifies the credentials**.
- Independent, in-memory NAS sessions backed by HttpOnly cookies in the local Go service. Passwords and session cookies never go to settings.json or the webview.
- Normal HTTPS certificate verification; explicit acknowledgement required before transmitting administrator credentials over HTTP.
- Live NAS views: System, Storage, Network, Hardware, Docker, Tasks, Users & Groups, Services, Backups, Protection Plans, Notifications, and unified SMB/NFS Shared Folders.
- Top-level File Manager restricted to real NAS shared folders, with authenticated navigation, create folder, upload, download, rename and Move to Recycle Bin. Large transfers are limited to 256 MiB.
- Original authenticated NAS management console remains available in the system browser for administrative features not yet reproduced securely in the desktop.


## Beta 5 desktop layout rework
The November 12, 2026 user screencast was decoded and visually reviewed in full: **929 consecutive frames**, without sampling. Details and before/after acceptance criteria are recorded in [UI-VIDEO-AUDIT.md](docs/UI-VIDEO-AUDIT.md).

- Fixed the **[object PointerEvent]** text appearing in Add Appliance's address field by removing the browser-event/default-address coupling.
- Added an always-visible appliance selector and sign-in status in the command header; NAS list moved ahead of secondary navigation.
- Replaced the 17-item always-open side navigation with a grouped management disclosure.
- Rebuilt Overview as an appliance-first command workspace with primary actions, connection identity, quick tasks and mapping summary.
- Rebuilt My Appliances as an actionable device directory with status and direct sign-in/management controls.
- Simplified File Manager's share list, removed literal **null** and eliminated duplicate initial Load/Loading UI.
- Reordered shared-folder connection around verified SMB/NFS inventories before advanced manual mapping.
- Added human-readable disk capacity meters, RAID health tables and opt-in raw diagnostics rather than dense JSON/mdstat blocks.
- Kept existing Zero1Local light/dark source branding, session boundaries, update APIs, and NAS administration abilities unchanged.

**Status:** Beta 5 has new source and regression tests; the native Windows/Linux release must still pass the build/test/package/release gate. End-to-end device UI acceptance is not implied by JavaScript parsing or Linux cross-compilation.


## Beta 6 — NAS-scoped workspaces and remembered sign-in

**My appliances is the landing screen.** Click an individual NAS to open its own Overview, File Manager, Shared Folders, Mapped Drives, and management sections. The NAS selector and full management navigation only appear inside a selected appliance workspace. Switching NAS devices clears the currently shown share selection. Mapped Drives lists only mappings belonging to the selected NAS.

**Sign-in is a choice on each NAS, not a global password setting.** By default passwords remain memory-only and signing in lasts only for the current application session. Select **Keep me signed in to this NAS** to explicitly store that NAS's credentials and current (up to eight-hour) session cookie in the native operating-system vault. On startup the desktop first tries to resume the original NAS session using \`/api/session\` and then attempts password reauthentication if the cookie was revoked or expired. TOTP codes are never stored. If the NAS demands two-factor verification again, the user is asked to sign in; the desktop will never silently bypass MFA. Signing out erases the remembered vault entry; **Stop remembering** removes the credential while leaving the active session connected. Passwords and cookies do not enter \`settings.json\`, frontend state, logs, or GitHub.

**Each Windows SMB share has its own independent Reconnect option.** Windows handles persistent SMB mappings and any Windows credentials; these are unrelated to the NAS administrator account. Linux and macOS share auto-reconnect are not advertised as implemented: the respective desktop/system file manager controls their credentials and reconnection.

Native OS vaults: Windows Credential Manager, macOS Keychain, Linux Secret Service. If a vault is unavailable/locked, opt-in persistent sign-in fails clearly rather than silently saving plaintext. Linux Secret Service requires an available unlocked keyring. The original NAS server and its eight-hour session policy are unchanged.

**Verification gate:** Node frontend regression suites, Go race and integration tests, Wails GTK tests/vet, cross-platform compilation, architecture checks, checksum verification and exact GitHub release assets. End-to-end native desktop tests and real NAS session/mount acceptance remain required before declaring stable.

## Connecting an appliance

1. Open **Add appliance**, enter its IP address or management hostname, then enter the NAS administrator username, password and optional six-digit TOTP code. Discovery candidates use the same sign-in process.
2. A NAS using plain HTTP requires an explicit risk acknowledgement before a password is sent. Prefer the NAS's valid HTTPS address whenever available.
3. The desktop verifies the account with Zero1Local, saves the appliance address and display name, and keeps the resulting NAS session **in memory only**. Restarting the app requires signing in again; passwords are not retained.
4. Select your NAS to use File Manager and the management sections. Use **Sign out** to discard the session. **NAS console** opens the original browser administration UI, which maintains a separate login session.
5. Share mapping still uses the operating system's SMB/NFS identity, not the NAS's web administrator password. A web login does not bypass share ACLs.

## GitHub Releases and built-in updates

- **v0.1.0-beta.1 (bootstrap)** is built from commit `d18ab27f459b9151622d8031badf08263e3102b0`: Linux x64 .deb and portable .tar.gz, plus Windows x64 portable .zip. The owner has confirmed a successful Docker build and SHA-256 validation, but not native GUI/runtime acceptance. **These original binaries do not contain the updater.**
- **v0.1.0-beta.2** contains the GitHub Releases updater. In native **Settings → Software updates**, choose **Check for updates**; a matching tray menu action is also present. Wails downloads only the platform's dedicated updater bundle and checks `SHA256SUMS` before presenting the restart/install action. It never installs updates on startup.
- **Linux Debian-managed installs** must remain package-manager managed. Settings provides **Download verified .deb** to fetch a newer Debian package, verify its SHA-256 digest and save it in the user's Downloads folder. The user installs it using their distribution's package manager; the companion does not silently gain root privileges.
- **v0.1.0-beta.3** corrects the broken desktop workspace loading screen and adds frontend smoke tests.
- **v0.1.0-beta.4 (source ready for qualification)** adds credentialed onboarding and broader management/File Manager capabilities. Build and release require the full test gate below.
- **macOS** updater support is implemented for future signed/notarized single-entry .app bundles, but no macOS release has been built.
- **No native GUI update has yet been confirmed on actual target desktops.** SHA-256 supplied with a GitHub Release validates asset integrity against that release, not publisher signature authenticity.
- Release asset names, bootstrap provenance and exact manual publishing procedure are documented in [RELEASING.md](docs/RELEASING.md).

## Current limitations — not a production release

- Linux amd64 and Windows amd64 binaries were built and published for Betas 1–3, but native desktop GUI, tray, share mapping, and credentialed NAS lifecycle acceptance on target computers remain incomplete. Beta 4 must pass its Go, frontend, native build and checksum checks before publication.
- Zero1Local management opens in the **system browser**, not an embedded NAS-origin webview, until remote-origin isolation is verified. The original interface and all its authorized management features are available there.
- Windows NFSv3 and macOS NFSv3 mount adapters remain unfinished. Existing NFS management must not be confused with SMB mapping.
- macOS Finder receives SMB URL mount requests but the app does not yet prove that the OS completed the mount.
- Administrator passwords and NAS session cookies are intentionally **not persisted**; per-appliance sign-in must be repeated after restart. Verified share mount auto-reconnection is not implemented. The app never weakens NAS permissions or filesystem access.
- Hardware verification, shared credential-manager integration, signed macOS release, Windows installer and cryptographically signed update manifests remain open engineering milestones.

## Local tests and preview

Go 1.23+ for portable core:

```bash
node --check internal/companion/ui/app.js
node --check internal/companion/ui/startup.js
node --test tests/frontend-startup.test.mjs
go test -race ./...
go vet ./...
go run ./cmd/z1l-preview
go run ./cmd/z1l-discover -scan
```

The localhost preview prints its URL and can be opened in a browser on the **same desktop**. It shares the same secure Go service and frontend as the native Wails application. The preview has no system tray and cannot launch the console natively.

## Native desktop application (requires a connected OS builder)

Go **1.26.10** + Wails **v3.0.0**:

```bash
cd desktop
go mod download
go build -trimpath -o zero1local-desktop .
./zero1local-desktop
```

Build each platform with its required Wails WebView dependencies in a separate builder. No GitHub Actions are required. Compiling packages is not equivalent to completing target-OS runtime and on-NAS acceptance. Keep beta releases labeled as prereleases until those gates pass.

## Engineering references

- [First-party Zero1Local design](docs/BRAND.md)
- [Product](docs/PRODUCT.md)
- [Architecture and existing NAS API](docs/ARCHITECTURE.md)
- [OS SMB/NFS mount policies](docs/MOUNTING.md)
- [Security contract](docs/SECURITY.md)
- [Roadmap and release gates](docs/ROADMAP.md)

This is a Gigabyte Grove project. No changes have been made to running Zero1Local NAS appliances.
