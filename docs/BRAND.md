# Zero1Local Desktop — binding visual parity contract

## Product identity

Zero1Local Desktop is the first-party companion to Zero1Local. It must look like **the same application** as the appliance admin interface. The desktop adds operating-system interactions (tray, connected appliances, share mapping) but must not invent a second brand.

**Source audited (2026-11-12):** private `gigabytegrove/zero1local-source/web/app.css`, specifically the later, operative application theme block beginning with `/* Zero1Local v0.1.22 Professional Interface & Experience */`, with original first-party logos under `web/assets/`. This is the canonical styling source; the earliest CSS `:root` block is overridden later. Verify parity against actual installed Zero1Local UI during acceptance.

## Canonical CSS variable values

| Token | Light | Dark |
| --- | --- | --- |
| `--bg` | `#f2f5f8` | `#111820` |
| `--surface` | `#ffffff` | `#17212b` |
| `--surface-soft` | `#f7f9fb` | `#1b2732` |
| `--surface-raised` | `#ffffff` | `#202c38` |
| `--ink` | `#182230` | `#e6edf4` |
| `--navy` | `#132840` | `#f0f5fa` |
| `--navy-2` | `#1d3d60` | `#c8d7e6` |
| `--muted` | `#68768a` | `#9aacbd` |
| `--line` | `#dfe5eb` | `#2d3b49` |
| `--line-strong` | `#c8d2dc` | `#405162` |
| `--blue` | `#1768b7` | `#62a8e8` |
| `--blue-2` | `#2678c7` | `#82b9ec` |
| `--blue-soft` | `#e9f2fb` | `#17344d` |
| `--good` | `#19734b` | `#61c895` |
| `--good-soft` | `#eaf6ef` | `#15372a` |
| `--warn` | `#8a5a08` | `#e3b55c` |
| `--warn-soft` | `#fff5df` | `#3b2e13` |
| `--bad` | `#a6323d` | `#ef838c` |
| `--bad-soft` | `#fff0f2` | `#421f25` |

Reusable CSS token definitions are in [ui/zero1local-theme.css](../ui/zero1local-theme.css); this token file is **not wired to a desktop UI yet**.

## Styling and layout invariants

- Primary typeface stack: `Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`; monospace for technical values.
- Existing 232px application rail and 60px topbar. Light sidebar `#f8fafc`, dark sidebar `#131d27`. Original Zero1Local top-left logo sizing `176px` wide, max-height `42px`.
- Existing active rail item: light background `#e4eef8`, foreground `#145e9f`; dark background `#1b3851`, foreground `#8dc3ef`.
- Spacing tokens `4px`, `8px`, `12px`, `16px`, `20px`, `24px`, `32px`. Base radii `8px` and `6px`.
- New desktop UI includes an appliance switcher and mapped drive table **within** Zero1Local's layout grammar. Avoid an unrelated cream/green theme, giant marketing hero, pill-heavy controls or a separate wallpaper-like desktop design.
- Honor the active theme, system-follow option, 360px layout, visible focus, WCAG 2.2 AA contrast and reduced-motion settings.
- Source paths of original artwork: `web/assets/zero1-local-logo-light.png`, `web/assets/zero1-local-logo-dark.png`, `web/assets/zero1-local-logo.png`. Preserve real asset bytes and proportions. Light/dark artwork pairing must be confirmed against the running UI; file suffix alone is not proof of which rendering variant to select.

## Implementation acceptance

1. Screenshot-parity review against the current appliance's Home, Files & Sharing, and Settings views in both themes; compare navigation rhythm, icons, typography and interaction, not just colors.
2. Native app features (NAS switcher, mount wizard, tray menu) remain recognizable as Zero1Local components.
3. Embedded NAS management UI is **not** styled by the desktop or rewritten; it retains its original appearance and authentication.
4. Real logo assets embedded and verified; no generic/generated substitute.
5. Do not call brand parity implemented until the Wails GUI actually renders and passes visual review on all three desktop operating systems.
