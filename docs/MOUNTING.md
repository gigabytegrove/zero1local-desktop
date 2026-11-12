# Platform mounting contract

Zero1Local currently supports SMB and userspace NFSv3. Source v1.4.3 prefers NFS, but older installed appliances may not include the new NFS-only share workflow. **Do not infer export names from UI labels or treat an NFS export as an SMB share.**

## Compatibility matrix

| Client | NFSv3 | SMB | Target |
| --- | --- | --- | --- |
| Windows 10/11 | Only when the optional Windows Client for NFS is available; actual edition/features matter | Native Windows SMB redirector | Free drive letter (e.g. Z:) or UNC only |
| Linux desktop | mount.nfs client plus polkit/elevation when needed | GIO/GVfs desktop mount or CIFS client as appropriate | User-selected mount folder or desktop file-manager volume |
| macOS | NFSv3 mounting requires appropriate system privileges/configuration | Finder / system SMB support | `/Volumes` mount or Finder-mounted share |

## Mount wizard steps
1. Authenticate on chosen NAS; fetch `/api/nfs/exports` and `/api/shares` using the existing user's rights, never with an invented global token.
2. Enumerate connection capabilities, export path, allowed client CIDRs, user access and server version. Display unavailable modes and requirements honestly.
3. Select target; validate Windows drive letter is available, or on Unix that the mount point is user-owned and safe.
4. Preview protocol/host/path/identity consequences and privilege dialog **before** executing. Explicit consent required for user-account autoremount and credential persistence.
5. Perform native mount. Confirm the OS mount table or redirector reports precisely that share at precisely that target; test access only within the user's granted permissions.
6. Record only non-secret mapping metadata for reconnection; password and tokens must stay inside OS credential vault/Keychain/Secret Service or OS-native credential dialogue.
7. On failure show actionable diagnostics without exposing credentials or mounting as a different identity. On unmount, verify the selected mount rather than bulk deleting mappings.

## Windows
SMB uses native Windows credentials and can map a selected available drive letter. Reuse Windows Credential Manager rather than supplying passwords in a `net use` argument, command-line switches, process environment or logs. Conflict: an existing mapping to the same server with another username may be rejected by Windows; detect and report rather than disconnect unrelated sessions.

NFS requires a supported Windows edition and Client for NFS feature. Show the original server-provided NFS export path. Windows NFS uses Unix UID/GID mapping, not the SMB username/password. Never auto-enable Windows features, change `AnonymousUid/Gid`, or weaken NFS export `root_squash`.

## Linux
Prefer the desktop's native volume mounting flow for SMB; do not assume that `gio`, `udisks`, or `mount.cifs` exists on every distribution. For NFS, use a narrow, auditable elevated helper/polkit action. Never shell-interpolate hostnames, paths or credentials; use explicit argv and deny traversal and symlink escape in mount-target validation.

## macOS
Prefer Finder/system SMB mount and Keychain. NFS needs real NFS client privileges and mount-policy handling; avoid assuming `open nfs://` works. Validate actual mounts under the requested target; show macOS privacy/security prompts. Never run privileged `osascript` with embedded passwords.

## Reconnect and errors
Store mount definitions by stable appliance identity, protocol, server endpoint, remote export/share and target; key them so two NAS devices cannot hijack one saved letter/folder. Retry with bounded exponential backoff, stop on auth error, and offer cleanup after NAS migration. Never silently overwrite a local directory or reassign a letter.
