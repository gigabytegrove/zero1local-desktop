# Implementation and acceptance roadmap

Status legend: **DONE** (implemented and verified), **NEXT** (not yet implemented), **GATED** (requires OS/hardware or security acceptance).

## Milestone 0 — product and source contract
- **DONE:** Private desktop repository initialized with product, architecture, security and mount contracts.
- **DONE:** Verified actual Zero1Local `/healthz`, `/api/status`, `/api/shares`, `/api/nfs/exports` and session routes against current private source.
- **DONE:** Chosen Wails v3.0.0 stable desktop framework, confirmed official Windows/macOS/Linux and tray support.
- **DONE:** Audited the current NAS source CSS and locked desktop light/dark tokens, navigation dimensions and original approved logo filenames to Zero1Local's existing design system.
- **NEXT:** Embed original logo bytes from private Zero1Local source without alteration; confirm app bundle ID, signing identities and app-update release owner.

## Milestone 1 — functional desktop foundation
- **NEXT:** Wails app shell with the **existing Zero1Local** light/dark themes, 232px navigation rail, 60px topbar, exact source logos, familiar settings/tables/icons, appliance shelf and accessible empty/error states.
- **NEXT:** Visually compare desktop shell and built-in NAS UI on Windows, Linux and macOS in both themes; do not sign off based only on matching colors.
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
