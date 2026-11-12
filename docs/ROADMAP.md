# Implementation and acceptance roadmap

Status legend: **DONE** (implemented and verified), **NEXT** (not yet implemented), **GATED** (requires OS/hardware or security acceptance).

## Milestone 0 — product and source contract
- **DONE:** Private desktop repository initialized with product, architecture, security and mount contracts.
- **DONE:** Verified actual Zero1Local `/healthz`, `/api/status`, `/api/shares`, `/api/nfs/exports` and session routes against current private source.
- **DONE:** Chosen Wails v3.0.0 stable desktop framework, confirmed official Windows/macOS/Linux and tray support.
- **NEXT:** Confirm official product logo asset and app bundle ID, signing identities, app-update release owner.

## Milestone 1 — functional desktop foundation
- **NEXT:** Wails app shell with approved branding, appliance shelf, status, helpful empty/error states and keyboard navigation.
- **NEXT:** Native tray on Windows/macOS/Linux; hide-to-tray, quit, window restore and single-instance behavior.
- **DONE:** Pure-Go bounded discovery of candidate LAN Zero1Local appliances with read-only health/status API validation, HTTP origin validation, cancellation and tests; diagnostic CLI.
- **NEXT:** Desktop integration, user-approved device registry, duplicate reconciliation, authenticated enrollment, optional CIDRs and mDNS/DNS-SD.
- **NEXT:** Secure trusted-appliance enrollment and TLS fingerprint change warning.
- **NEXT:** Embedded authenticated Zero1Local admin window with tested isolation from privileged desktop bindings.

## Milestone 2 — real share mounting
- **NEXT:** Authenticated catalog adapter for installed-version SMB and NFS response shapes.
- **NEXT:** Windows SMB drive-letter map/unmap, OS credential prompts, mount status and collision handling.
- **NEXT:** Linux SMB desktop mount and NFSv3 mount with narrow privilege consent.
- **NEXT:** macOS SMB Finder integration and system NFSv3 mounts.
- **NEXT:** Persistent opt-in reconnect and recover after sleep, Wi-Fi changes and multi-VLAN transitions.

## Milestone 3 — release quality
- **NEXT:** OS-native autostart toggles and selective notifications.
- **NEXT:** App settings, device rename/removal and mapping repair.
- **NEXT:** UI and service regression tests; `go test -race`, `go vet`, `wails3 doctor` and native build validation.
- **GATED:** Native signed Win x64/ARM64 package, Linux x64/ARM64 packaging, macOS Intel/Apple Silicon bundle/notarization.
- **GATED:** Manual smoke tests with non-production test NAS and real share/user ACLs; no unattended modification of operational NAS units.
- **GATED:** Alpha GitHub Release with reproducible build record, checksum manifest, provenance and known limitations.

## Decision gates
1. **Embedded webview origin isolation:** may not be claimed done without a reproducible malicious-page test on all three platforms.
2. **NFS Windows UX:** must inspect Windows feature availability, edition support, server-export ACL and UID/GID behavior; never assume a command success means write access.
3. **Cross-VLAN discovery:** mDNS not expected across routers unless reflected; allow user-authorized CIDRs and explicit endpoints without promoting invasive broad scans.
4. **Mounted share persistence:** must not reassign a drive letter/mount path already used by another filesystem.
