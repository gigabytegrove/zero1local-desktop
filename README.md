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
- Optional authenticated read-only SMB/NFS share inventory against existing NAS endpoints over verified HTTPS (no passwords or API keys are persisted).

## Current limitations — not a production release

- Native Wails binaries are not yet built, code-signed, packaged, or tested on real Windows/macOS/Linux systems.
- Zero1Local management opens in the **system browser**, not an embedded NAS-origin webview, until remote-origin isolation is verified. The original interface and all its authorized management features are available there.
- Windows NFSv3 and macOS NFSv3 mount adapters remain unfinished. Existing NFS management must not be confused with SMB mapping.
- macOS Finder receives SMB URL mount requests but the app does not yet prove that the OS completed the mount.
- Credentials and mapped drive persistence/auto-reconnection are not implemented. The app never asks to disable NAS permissions or weaken filesystem access.
- Hardware verification, shared credential-manager integration, signed release, and native installer remain open engineering milestones.

## Go core (tested)

Go 1.23+ for portable core:

```bash
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

Build each platform natively with installed Wails WebView dependencies, in a temporary builder environment. No GitHub Actions. Do not publish an installer until build checks, local NAS authentication, mapping and sleep/resume have been tested.

## Engineering references

- [First-party Zero1Local design](docs/BRAND.md)
- [Product](docs/PRODUCT.md)
- [Architecture and existing NAS API](docs/ARCHITECTURE.md)
- [OS SMB/NFS mount policies](docs/MOUNTING.md)
- [Security contract](docs/SECURITY.md)
- [Roadmap and release gates](docs/ROADMAP.md)

This is a Gigabyte Grove project. No changes have been made to running Zero1Local NAS appliances.
