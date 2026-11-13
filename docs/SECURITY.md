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

## Administrator onboarding and local session boundary

The desktop sends username/password/TOTP directly from its localhost UI to its own Go service over loopback, then to the selected NAS's documented `POST /api/login` with Zero1Local's mutation guard. It verifies login success and the issued `zero1_session` cookie before saving a new device. Each NAS has its own in-memory `http.CookieJar`. Local APIs never return NAS session cookies, API keys or administrator passwords, and neither startup nor logout saves them.

TLS uses the operating system trust store with normal certificate verification and no insecure skip. Cleartext HTTP administrator login requires the user to explicitly acknowledge unencrypted password transmission each time. The NAS client does not accept redirects or inherited HTTP proxies. NAS responses are bounded JSON from an exact endpoint allowlist, not a general-purpose remote-content proxy.

File mutations call only `/api/files/mkdir`, `/api/files/rename` and `/api/files/trash` with the existing NAS request guard; the NAS is solely responsible for share-root validation and ACL enforcement. Downloads and uploads have 256 MiB bounds and a per-action local CSRF token. Permanent deletion, NAS shell access, arbitrary forwarding and untrusted NAS-origin script loading are not implemented.

## Trust boundary
Zero1Local's existing server authorizes all administration. Desktop code must not implement an unprivileged workaround for API ACL, login, CSRF, TOTP or NFS root squash. A host being reachable does not prove the authenticated user's access. Management sessions remain specific to each appliance and are not copied between webviews.

## Private/public repositories
This repository (`gigabytegrove/zero1local-desktop`) is public desktop source. The separate NAS implementation source (`gigabytegrove/zero1local-source`) is private; public `gigabytegrove/zero1local` distributes NAS releases. The desktop contains no copied NAS server implementation, administrator passwords, authentication cookies, encryption keys, or private deployment addresses. No desktop release automatically modifies either NAS repository.

## Security acceptance
Tests for malicious hostnames, credential-bearing URLs, redirects, bad TLS, huge JSON bodies, partial cancellation, OS mapping collisions, interrupted mounts, malicious export paths, and management webview origin isolation are mandatory. Windows Defender, notarization/quarantine on macOS and Linux packaging/signature checks must be exercised on real hosts.
