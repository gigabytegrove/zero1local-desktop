# Zero1Local Desktop implementation status (2026-11-12)

## Completed in source (portable Go verified)
- Existing Zero1Local management API endpoints reviewed against private NAS source.
- First-party brand parity tokens and two original PNG logo assets in the private desktop repository.
- Go device scanning and manual health identification; bounded current private /24s, ports 80 and 8088.
- Persistent user-only installation list and theme settings; no passwords saved.
- Desktop workspace JS/CSS with overview, appliances, share access, mapping list and settings views.
- Local Go HTTP desktop service and browser protections. Original NAS routes are not proxied or altered.
- Windows SMB drive-letter command adapter, Linux GVfs SMB and NFSv3 (with pkexec) and macOS SMB Finder launcher.
- Read-only Zero1Local shares and NFS export API requests using per-request bearer token **only** over HTTPS.
- Go tests for portable core, store, unauthorized API calls, mount validation and unknown appliance.

## Native host present, platform validation still blocked
- Wails v3.0.0 native window and system tray source is committed in desktop/main.go.
- Go 1.26.10 and OS WebKit/GTK/WebView2 dependencies are needed for native compilation.
- This environment does not have that Go toolchain or all target OS runtimes; native compile/test is not claimed.
- Production-quality app icon crops, code signing, notarization, Windows MSI/EXE installers, Linux packages and macOS bundles are pending.

## Essential outstanding acceptance work
- [ ] Secure NAS-origin embedded WebView so browser scripts cannot invoke desktop privileged APIs. Current implementation opens existing NAS UI externally.
- [ ] Windows native SMB credential manager prompts and mount verification; detect conflicts with active mappings.
- [ ] Native Windows NFSv3 and macOS NFSv3 mount adapters with identity/ACL prerequisites and real test evidence.
- [ ] Verified Linux mount status, reconnection and sleep/wake handling; no blind source-host remapping.
- [ ] Appliance identity pinning, TLS-fingerprint changes, mDNS/DNS-SD, explicitly authorized cross-VLAN CIDRs, and current reachability.
- [ ] Confirmed system tray icons, startup at login (opt-in), per-OS notifications and host-specific packagers.
- [ ] End-to-end tests on a nonproduction NAS including OS credentials, read/write ACLs, offline recovery and all management routes.
- [ ] Signed release artifacts and SHA-256 manifest uploaded directly to Releases by native builders (no GitHub Actions).

Do not claim this first substantive code pass is an installable stable application. Do not automatically update or alter in-service Zero1Local appliances.
