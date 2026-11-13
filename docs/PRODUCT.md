# Product specification · Zero1Local Desktop

## Purpose
A native Windows, Linux, and macOS companion that makes several Zero1Local NAS installations feel like local drives and one management workspace. It remains useful when closed to its tray/menu bar, with no always-running privileged daemon required.

## Required user journeys
1. **First run:** desktop opens a short introduction; detects recognizable Zero1Local services on local interfaces; offers Add by IP/hostname/URL, including routed/VLAN-connected networks. Every discovered device is explicitly confirmed and requires a verified NAS administrator login (username, password, optional TOTP) before it is saved. First-run never installs components or changes NAS settings.
2. **Multiple appliances:** a persistent switcher lists name, endpoint, reachability, discovered version, and last verified identity; users can rename, pin, reorder, forget, and select a favorite. Appliances can be on different trusted VLANs; network access is never silently assumed.
3. **Manage:** selecting a NAS exposes authenticated, NAS-ACL-controlled live management pages in the desktop, preserving separate per-appliance in-memory sessions. Complete advanced management is available in the actual original Zero1Local UI through an external browser until NAS-origin WebView isolation has been proven secure. Passwords and cookies are never persisted or sent to browser-side JavaScript.
4. **Files:** choose NAS → Shares → NFS or SMB → folder/share → map target. Show protocol, server-reported export path or share name, read/write expectation, identity mapping/ACL prerequisites and permission elevation. Distinguish mapping success from mere command launch.
5. **Reconnect:** resume saved mounts when the desktop user logs in **if and only if** the operator enabled that mount. Pause retries on authentication failures; offer repair; never globally alter networking, firewall, or filesystem permissions.
6. **Background:** tray/menu bar opens app, lists recent devices/shares, shows disconnected/offline status, and quits fully. Native startup toggle is opt-in. Quitting does not unmount user data without explicit request.
7. **Accessibility:** keyboard traversal, 4.5:1 body contrast, screen-reader labels, reduced motion, clear empty/loading/error/success/disabled states, minimum 360px responsive layout.

## Visual identity — locked to Zero1Local
This product is a **first-party Zero1Local companion**, not a separately branded disk utility. It must reproduce the existing NAS admin UI's design language while adding multi-appliance switching and native desktop operations. The source of truth is `gigabytegrove/zero1local-source/web/app.css` and its approved `web/assets/zero1-local-logo-*.png` graphics. Do not replace it with an unrelated green/cream theme.

- **Personality:** familiar, functional, understated; reference: the existing Zero1Local admin application, not a new product concept.
- **Light theme:** actual source-derived tokens: background `#F2F5F8`, surface `#FFFFFF`, ink `#182230`, muted `#68768A`, primary/navy `#132840`, secondary/navy `#1D3D60`, blue accent `#1768B7`, soft blue `#E9F2FB`, line `#DFE5EB`.
- **Dark theme:** actual source-derived tokens: background `#111820`, surface `#17212B`, raised `#202C38`, ink `#E6EDF4`, muted `#9AACBD`, accent `#62A8E8`, line `#2D3B49`.
- **Typography:** `Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`; monospace for hostnames, export paths and drive letters.
- **Layout:** existing Zero1Local application rail (`232px`) and `60px` top bar, source-like navigation, typography, icon treatments, tables and settings; keep the NAS picker visually integrated with the same navigation system.
- **Shape/depth:** existing `8px` and `6px` base radii, 1px dividers and restrained surface shadows. Avoid decorative gradients, unrelated icon sets and newly invented controls.
- **Imagery:** reuse the original Zero1Local light and dark logo assets byte-for-byte from its private source; retain original proportions and never recolor or re-create them.
- **Theme behavior:** explicit light/dark choice with persistent user setting and operating-system-follow option; maintain equivalent contrast, hover/focus and disabled states. Do not introduce a new design system.
- **Motion:** 150–250ms lightweight transitions, disabled under `prefers-reduced-motion`. Familiar status indications and share mapping rows are native desktop additions within the established brand.

See [BRAND.md](BRAND.md) for source references, exact full token contract, parity acceptance, and implementation guidance.

## Exclusions
No full file-sync engine, remote registration service, remote VPN provisioning, cloud identity, arbitrary root filesystem browsing, credentials embedded in commands, iSCSI invention, or automatic NAS upgrades.

## Release gate
Actual Windows 10/11, current macOS Intel/Apple Silicon and supported Linux desktops must pass UI, tray, network discovery, share mount/unmount, security and sleep/resume testing, not just cross-compilation. No features are called complete merely because buttons render.
