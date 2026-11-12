# Security and deployment boundary

## Threats and controls
| Threat | Control |
| --- | --- |
| Rogue LAN host impersonates Zero1Local | `/healthz` is a **candidate marker only**; users explicitly approve devices; prefer HTTPS and certificate validation/pinning, with warning for HTTP |
| Host or reverse-proxy redirect to malicious site | No redirects during discovery; validate URI schemes, destination, and final origin for embedded UI |
| NAS webview JavaScript requests local OS operations | Separate webview origin and deny bridge/service exposure to remote content; fail to system browser if isolation cannot be enforced |
| Credentials in process list, logs, config or telemetry | No password-bearing argv, URL, logs, or config. Use OS credential store or OS-owned credential prompts |
| Malicious share name/mount target | Treat server metadata as untrusted, check canonical paths and reserved names, use structured argv, never shell execute; require operator consent for mounts |
| Auto-discovery floods networks | Current /24 only, short request timeouts, bounded workers, explicit opt-in for additional routed CIDRs; no automatic WAN scanning |
| Background startup becomes persistence abuse | Opt-in native startup. UI indicator and disable control; never install a root daemon silently |
| Disk/data loss | Unmount only explicitly selected managed mounts; never delete mounted folders, user data, exports or NAS content |
| Dead appliance triggers repeated credential prompts | Bounded retries, pause on authentication error, notify once per event |
| Compromised NAS uses updater path | Signed desktop release bundles, checksums, immutable GitHub Releases, no code execution from NAS responses |

## Trust boundary
Zero1Local's existing server authorizes all administration. Desktop code must not implement an unprivileged workaround for API ACL, login, CSRF, TOTP or NFS root squash. A host being reachable does not prove the authenticated user's access. Management sessions remain specific to each appliance and are not copied between webviews.

## Private/public repositories
This repo is private desktop source. Zero1Local private NAS source is `gigabytegrove/zero1local-source`; public `gigabytegrove/zero1local` is package distribution. No automatic rewrite or release of any of those repositories.

## Security acceptance
Tests for malicious hostnames, credential-bearing URLs, redirects, bad TLS, huge JSON bodies, partial cancellation, OS mapping collisions, interrupted mounts, malicious export paths, and management webview origin isolation are mandatory. Windows Defender, notarization/quarantine on macOS and Linux packaging/signature checks must be exercised on real hosts.
