# Zero1Local Desktop

Go-native desktop companion for [Zero1Local](https://github.com/gigabytegrove/zero1local): Windows, Linux and macOS device discovery, background access, per-appliance management, and SMB/NFS share mapping.

**Project stage:** engineering foundation, not a production release. The Zero1Local appliances are not modified by this repository.

## Product contract

- **All installs in one place:** detect on trusted local networks, pin verified NAS identities, retain named and manually-added endpoints, and show reachability without confusing it with authentication.
- **Always within reach:** optional start-at-login, native system tray/menu bar, minimized/background operation, a single running instance, and quick per-NAS actions.
- **Manage the complete NAS:** open the installed Zero1Local management UI **within a native application window** using the appliance's existing session, permissions, routes and MFA. No cloned admin controls or credentials in URLs.
- **One-click share mounting:** discover actual SMB shares and NFS exports after authentication; provide Windows drive letters and filesystem mounting for Linux/macOS; always show credential/privilege prompts when needed. Never bypass server ACLs.
- **Local-first:** no cloud backend, remote bootstrap, automatic port forwarding, credential synchronization, or bundled VPN. Access via existing trusted LAN or a user-managed private network path.

## Engineering decisions

Go application with [Wails v3.0.0](https://v3.wails.io/) for native Windows/macOS/Linux webviews and system trays. Separate portable Go domain packages from platform-specific mounting and privilege workflows. The bundled desktop shell has its own identity and is deliberately **not** a restyle of Zero1Local's administrator UI.

The existing Zero1Local source is under the private `gigabytegrove/zero1local-source` repo, currently a v1.4.3 NFS-first **source candidate**, not necessarily what each installed NAS is running. Runtime capability checks, not version-string guesses, determine whether an appliance offers an operation.

## Repository documentation

- [Product and UX specification](docs/PRODUCT.md)
- [Architecture and source integration](docs/ARCHITECTURE.md)
- [Platform share-mount contract](docs/MOUNTING.md)
- [Security and threat model](docs/SECURITY.md)
- [Implementation and acceptance roadmap](docs/ROADMAP.md)

## Release policy

No GitHub Actions. Build in clean temporary containers or native platform builders, produce checksums and signed artifacts, and publish to GitHub Releases only after platform acceptance. No installation or change to a running Zero1Local NAS is part of desktop development.

© Gigabyte Grove. Product owner: Brad Trammell.
