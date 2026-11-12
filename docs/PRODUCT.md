# Product specification · Zero1Local Desktop

## Purpose
A native Windows, Linux, and macOS companion that makes several Zero1Local NAS installations feel like local drives and one management workspace. It remains useful when closed to its tray/menu bar, with no always-running privileged daemon required.

## Required user journeys
1. **First run:** desktop opens a short introduction; detects recognizable Zero1Local services on local interfaces; offers Add by IP/hostname/URL, including routed/VLAN-connected networks. Every discovered device is explicitly confirmed before being trusted. First-run never installs components or changes NAS settings.
2. **Multiple appliances:** a persistent switcher lists name, endpoint, reachability, discovered version, and last verified identity; users can rename, pin, reorder, forget, and select a favorite. Appliances can be on different trusted VLANs; network access is never silently assumed.
3. **Manage:** selecting a NAS opens its **actual** authenticated Zero1Local UI inside an app window, with the same capabilities, CSRF protections, multi-factor login, file manager and direct URLs as the installed NAS. Separate per-appliance sessions; no password harvesting.
4. **Files:** choose NAS → Shares → NFS or SMB → folder/share → map target. Show protocol, server-reported export path or share name, read/write expectation, identity mapping/ACL prerequisites and permission elevation. Distinguish mapping success from mere command launch.
5. **Reconnect:** resume saved mounts when the desktop user logs in **if and only if** the operator enabled that mount. Pause retries on authentication failures; offer repair; never globally alter networking, firewall, or filesystem permissions.
6. **Background:** tray/menu bar opens app, lists recent devices/shares, shows disconnected/offline status, and quits fully. Native startup toggle is opt-in. Quitting does not unmount user data without explicit request.
7. **Accessibility:** keyboard traversal, 4.5:1 body contrast, screen-reader labels, reduced motion, clear empty/loading/error/success/disabled states, minimum 360px responsive layout.

## Visual identity
Personality: **calm, tactile, reliable**; reference: a well-organized physical workstation and modern operating-system disk utility, not an administrative analytics dashboard.
- Light palette: canvas #F7F6F1, surface #FFFFFF, ink #26302B, secondary #56645A, evergreen #24583E, sage #DCE9DE, warm action #AA5B23, critical #A4312D, rule #DBDFD7.
- Fonts: native UI system sans (Segoe UI on Windows, SF on macOS, system UI on Linux); monospace for paths, hostnames, drive letters.
- Layout: left installation rail (216px on desktop), center file/workspace overview, compact device utility strip, optional management workspace tabs. No repetitive boxed-card grid.
- Navigation: persistent appliance list; main navigation Overview / Shares / Mounts / Manage / Settings; tray shortcuts.
- Shape/depth: flat soft surfaces, 1px separators, 4–8px radii, very light shadows only for popovers.
- Imagery: existing approved Zero1Local logo; authentic drive, network and device glyphs; no stock photos.
- Motion: 180ms transitions, reduced-motion suppression. Signature: connected NAS nodes as an orderly left-edge appliance shelf, with each mounted share displayed as a row and OS drive/mount badge.

## Exclusions
No full file-sync engine, remote registration service, remote VPN provisioning, cloud identity, arbitrary root filesystem browsing, credentials embedded in commands, iSCSI invention, or automatic NAS upgrades.

## Release gate
Actual Windows 10/11, current macOS Intel/Apple Silicon and supported Linux desktops must pass UI, tray, network discovery, share mount/unmount, security and sleep/resume testing, not just cross-compilation. No features are called complete merely because buttons render.
