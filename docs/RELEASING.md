# Zero1Local Desktop release and update contract

## Artifact provenance

The existing dockeradm build from `d18ab27f459b9151622d8031badf08263e3102b0` produced three verified files: Linux amd64 .deb, Linux amd64 .tar.gz and Windows amd64 .zip. Tag **v0.1.0-beta.1** at precisely that commit. Never pretend this initial binary includes code subsequently committed to `main`.

Build updater-enabled **v0.1.0-beta.2** only after committing all Go source, `desktop/go.mod` changes and generated `desktop/go.sum`. Embed the version `0.1.0-beta.2` in both native binaries using `-ldflags "-X main.currentVersion=0.1.0-beta.2"`. Tag the exact final build commit.

## Release asset naming

For each release publish **both** human installation packages and dedicated machine-updater payloads:

| Asset | Purpose |
| --- | --- |
| `Zero1Local-Desktop-<commit12>-linux-amd64.deb` | Debian package install/update |
| `Zero1Local-Desktop-<commit12>-linux-amd64.tar.gz` | Human portable Linux archive |
| `Zero1Local-Desktop-<commit12>-windows-amd64.zip` | Human portable Windows archive including README |
| `Zero1Local-Desktop-Updater-linux-amd64.tar.gz` | **Exactly one** top-level Linux binary, `zero1local-desktop` |
| `Zero1Local-Desktop-Updater-windows-amd64.zip` | **Exactly one** top-level Windows binary, `Zero1Local-Desktop.exe` |
| `SHA256SUMS` | SHA-256 for every binary and package asset above |

The Wails updater explicitly selects only the exact `-Updater-` names for the running platform; never rename them, introduce a second top-level file, or select `.deb` for Wails binary replacement. Archives with two entries are rejected by Wails.

The GitHub Releases provider uses `Prerelease: true` for the beta channel. Version tags must be semver: `v0.1.0-beta.1`, `v0.1.0-beta.2` and onward. Update version code at every release.

For macOS, add `Zero1Local-Desktop-Updater-darwin-amd64.zip` and `Zero1Local-Desktop-Updater-darwin-arm64.zip` **only after** packaging actual single-top-level signed/notarized `.app` bundles on an appropriate platform. Do not manufacture fake macOS binaries.

## Release gate

1. Validate portable Go tests, native Wails Go tests, `go vet`, target architecture and clean Linux/Windows builds.
2. Validate `SHA256SUMS` offline for all five packages, and archive single-entry rules.
3. Confirm tag SHA exactly equals source used to build the assets; do not retag a published release.
4. Create a GitHub **prerelease** with all assets and notes. Keep stable/latest promotion disabled.
5. Confirm public GitHub API reports the exact tag and all six required assets, and redownload/verify at least one.
6. Test the native app on both Windows and Linux; test that current beta reports **Up to date**, then a later release stages and applies atomically to a writable portable build.
7. For Debian-installed Linux, verify the app downloads the newer .deb, rejects checksum failures, preserves existing files and requires an operator-managed installation. Do not direct-swap dpkg-owned binaries.

## Distribution and security

No GitHub Actions or credentials on end-user machines. The GitHub provider uses public GitHub Releases and SHA256SUMS over HTTPS. This is a **digest-only beta trust model**, not a cryptographic signature anchored in a publisher public key. A production promotion must add independent Ed25519 verification with an embedded public key and an owner-controlled signing process; never commit a private signing key.

No GitHub release asset should be replaced or overwritten after publication. If a packaged binary is defective, make a newly versioned release. The public repository contains only desktop companion source, not private NAS appliance source.

## Existing bootstrap build

The owner-reported successful builder log recorded `BUILD_AND_PACKAGING_SUCCESS` and verified three original archives, with files located under `/root/zero1local-desktop/dist` on dockeradm. This conversation does **not** have direct access to that server's filesystem. Publication requires uploading those exact original files from dockeradm using the GitHub CLI.

See `docs/releases/v0.1.0-beta.1.md` and `docs/releases/v0.1.0-beta.2.md` for release notes.

## Beta 4 source and validation gate

- Source tag must point to the exact commit built on dockeradm; embedded `main.currentVersion` must equal `0.1.0-beta.4`.
- Run `node --check` on the frontend scripts and `node --test tests/frontend-startup.test.mjs` before the native build.
- Run Go tests with the race detector, Go vet and the Wails GTK-tagged test/vet checks. Beta 4 includes NAS authentication and file-operation integration tests with a local mock NAS.
- Package Windows and Linux installers and two **single-entry** updater archives using the stable asset names above. Verify the manifest and remote release list.
- Tests on actual NAS devices must cover successful admin authentication, invalid credentials, TOTP, HTTP opt-in, HTTPS certificate handling, session expiry, separate NAS identities, share browsing, uploads/downloads, and NAS-level ACL enforcement before public stable promotion.
- File transfers over 256 MiB and desktop operations that require unverified management mutation contracts must continue to use the full Zero1Local console. Never imply all NAS features have been reproduced in the native wrapper.

## Beta 5 UI acceptance
- Exact source commit is tagged `v0.1.0-beta.5` and embedded in both native executables via `-X main.currentVersion=0.1.0-beta.5`.
- Existing Beta 4 assets are immutable; publish Beta 5 only after native Go tests and `node --test tests/frontend-startup.test.mjs tests/desktop-layout.test.mjs`.
- Validate keyboard disclosure behavior, the source-colored light/dark themes, NAS selector, full per-appliance sign-in, file list, share mapping, storage gauge and correct empty/loading/error states.
- Screenshots and 929-frame audit are source evidence, **not** live acceptance of the newly rendered Beta 5 binaries.

## Beta 6 source and release gate
- Source and updater fallback version: \`v0.1.0-beta.6\`.
- Native Windows/macOS/Linux credential vault integration depends on pinned \`github.com/zalando/go-keyring v0.2.8\`; both module and platform transitive checksums must be present and \`go mod verify\` must pass.
- Execute \`node --check\`, both Node test files, \`go test -race -count=1 ./...\`, \`go vet ./...\`, and Wails GTK-tagged test/vet before compiling packages.
- Verify secure vault/session tests including no plain-text credential leakage, resumed MFA-approved cookies, NAS session expiry recovery, manual sign-out, and Windows share-only reconnect.
- Keep NAS sessions scoped to the source appliance; no module or process should write credentials to settings JSON.
- Publish **only** if all build gates pass. Keep Beta 5 immutable and Beta 6 explicitly marked prerelease. Verify 6 release assets and exact source SHA.
- Native desktop and actual appliance integration tests remain acceptance requirements, including credential vault availability and locked-keyring errors.
