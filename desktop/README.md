# Native Windows, Linux and macOS desktop host

The native application uses **Go 1.26.10** and **Wails v3.0.0** with a Zero1Local-themed native window, a background system tray and a pure-Go localhost API in the parent module.

Build on an actual graphical desktop running Windows, macOS or Linux:

```bash
cd desktop
go mod download
go build -trimpath -o zero1local-desktop .
./zero1local-desktop
```

Go 1.26.10 and native WebKit/GTK/WebView2 dependencies are required. There is no GitHub Actions workflow. This source file has been inspected against the Wails v3 official APIs, but native cross-platform builds and runtime tests remain pending.

The native window closes to the tray; tray actions restore, hide and quit. It does not install a system daemon, force login autostart, or alter a NAS. The management UI currently opens through the operating-system browser as a deliberate origin-isolation safeguard; in-app remote-origin webview support requires further security tests.
