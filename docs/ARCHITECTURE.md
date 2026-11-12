# Architecture and Zero1Local integration

## Runtime layout
```text
zero1local-desktop (Go + Wails v3.0.0)
  desktop shell / tray / menu bar
  device registry (untrusted discoveries -> verified/pinned appliances)
  Go connection manager (timeouts, checks, capability negotiation)
  share catalog (authenticated NFS and SMB inventory)
  mount coordinator (OS adapters; consent and permission boundary)
  native management webview per appliance (separate session isolation)
  optional login-start helper (per OS)
                  |
   existing authorized LAN/VLAN/private route
                  |
  Zero1Local zero1d + original Zero1Local admin web UI
```

## Verified Zero1Local source interfaces
Source examined: private `gigabytegrove/zero1local-source` as of 2026-11-12; code path `cmd/zero1d/main.go`. **Verify against the actual installed NAS version.**

| Endpoint | Auth | Purpose |
| --- | --- | --- |
| GET `/healthz` | public | Returns `ok`, `service: "zero1-local"`, `version`, `setup`; safe candidate identification, not proof of trust |
| GET `/api/status` | public | Returns `device_name`, `version`, `setup_required`, `boot_id`; name and basic availability |
| GET `/api/setup/discovery` | public | Existing-configuration adoption inventory; **not** intended for scanning; avoid it during normal discovery |
| GET `/api/shares` | authenticated | SMB `active` inventory with share names and permission-related attributes |
| GET `/api/nfs/exports` | authenticated | NFS export inventory (current source candidate) |
| GET `/api/session` | authenticated | Current admin session; no credentials should be captured from webview |
| `/api/connect/v1` | paired LAN mobile clients | Existing mobile share/file operations, **not** a desktop management superuser API |

`/healthz` explicitly reveals service name and version. A lookalike response can be forged on a local network, so discovery is untrusted until appliance authentication and certificate/identity verification. HTTP is supported for existing trusted LAN deployments, but must carry a prominent unencrypted-session warning; never quietly downgrade a saved HTTPS endpoint to HTTP.

## Discovery
Priority: (1) known/pinned entries; (2) mDNS/DNS-SD when advertised by a compatible server; (3) bounded current-LAN IPv4 probes of management port 80 and legacy 8088, with concurrency/time limits; (4) explicitly provided CIDRs and manually added hostnames/URLs. Do **not** enumerate an entire routed /16 by default. Report service and hostname separately; many devices behind reverse proxies must not collapse into one appliance. Do not accept an HTTP redirect as proof of product identity.

## First-party visual parity
The local desktop shell inherits the canonical Zero1Local visual language from `web/app.css` and approved logo assets in the private NAS source repository. Desktop-specific pages such as the appliance picker, mount editor and tray menu must use those same design tokens, icons, type scale and accessible interaction patterns. The management webview renders the installed NAS UI directly; never inject a competing theme or overwrite its existing UI components. See [BRAND.md](BRAND.md).

## Management webview
The desktop shell and remote appliance webview must be separate windows. Webview contents loaded from a NAS are **remote untrusted content**; do not expose privileged desktop Go services or OS mount APIs to that origin. Verify Wails bridge isolation/origin allowlisting at the pinned v3 release **before enabling in-app remote navigation**. If the platform cannot enforce this security boundary, open the authenticated console in the system browser and report the embedded-console requirement as blocked instead of weakening the boundary. Existing app navigation and cookies remain on the appliance origin.

## Server compatibility contract
The desktop client should not require changes to NAS runtime. Future optional Zero1Local read-only API improvements belong in a separately reviewed source change; the three in-service units are not test targets for automatic updates. Source v1.4.3 is NFS-first but unpublished/unqualified on hardware; detect installed capabilities rather than assuming it is present.

## Go build
- Go 1.26+ for Wails v3.0.0; ideally use the pinned project toolchain when the desktop app module lands.
- Pure-Go core packages must remain independently race-testable without WebKit/GTK or Windows/macOS build dependencies.
- No GitHub Actions. Native builds in temporary builder machines/containers; sign/checksum and publish manually after verification.
- Avoid global OS settings changes. No listener on a public network interface.
