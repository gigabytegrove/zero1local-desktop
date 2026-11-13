package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	updateRepository = "gigabytegrove/zero1local-desktop"
	updateChannel = "beta"
)

// currentVersion is set to the release's version (without v) by -ldflags.
// Always update this fallback with each source release; a fresh dev build
// otherwise risks reporting a version different from its actual code.
var currentVersion = "0.1.0-beta.6"

// updateArtifactName is intentionally strict. The GitHub release also carries
// .deb installers and user-facing ZIPs, neither of which should be passed to
// the Wails binary replacement engine. Only single-entry updater bundles pass.
func updateArtifactName(goos, goarch string) string {
	switch {
	case goos == "windows" && goarch == "amd64":
		return "Zero1Local-Desktop-Updater-windows-amd64.zip"
	case goos == "linux" && goarch == "amd64":
		return "Zero1Local-Desktop-Updater-linux-amd64.tar.gz"
	case goos == "darwin" && goarch == "amd64":
		return "Zero1Local-Desktop-Updater-darwin-amd64.zip"
	case goos == "darwin" && goarch == "arm64":
		return "Zero1Local-Desktop-Updater-darwin-arm64.zip"
	}
	return ""
}

// writableInstallDirectory is a necessary (not sufficient) precondition for
// native binary replacement. Debian-managed installs must always go through
// the package manager so dpkg's ownership and version records stay correct.
func writableInstallDirectory(goos, executable string) error {
	if executable == "" {
		return errors.New("cannot locate the running desktop executable")
	}
	executable, err := filepath.Abs(executable)
	if err != nil {
		return err
	}
	if goos == "linux" && (strings.HasPrefix(executable, "/usr/bin/") ||
		strings.HasPrefix(executable, "/usr/local/bin/") ||
		strings.HasPrefix(executable, "/opt/")) {
		return errors.New("system-installed Linux application: install the updated .deb from GitHub Releases using your package manager")
	}
	f, err := os.CreateTemp(filepath.Dir(executable), ".zero1local-updater-permission-*")
	if err != nil {
		return fmt.Errorf("application directory is not writable: %w", err)
	}
	name := f.Name()
	closeErr := f.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil {
		return removeErr
	}
	return nil
}

// configureUpdater uses Wails v3's verified GitHub Releases provider and
// framework-owned update modal. No auto-install occurs during application
// startup; an update is initiated explicitly by the desktop user.
func configureUpdater(app *application.App) (func(context.Context) error, string, error) {
	assetName := updateArtifactName(runtime.GOOS, runtime.GOARCH)
	if assetName == "" {
		return nil, "No updater bundle is published for this platform and architecture.", nil
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, "Unable to identify the running executable.", nil
	}
	if err := writableInstallDirectory(runtime.GOOS, executable); err != nil {
		return nil, err.Error(), nil
	}
	gh, err := github.New(github.Config{
		Repository:    updateRepository,
		Prerelease:    true,
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher: func(_ updater.CheckRequest, assets []github.ReleaseAsset) int {
			for i, asset := range assets {
				if asset.Name == assetName {
					return i
				}
			}
			return -1
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("create GitHub update provider: %w", err)
	}
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: currentVersion,
		Providers:      []updater.Provider{gh},
		Window: &updater.BuiltinWindow{
			CSS: `:root {
				--bg: #f2f5f8;
				--surface: #ffffff;
				--surface-2: #f7f9fb;
				--fg: #182230;
				--fg-dim: #68768a;
				--border: #dfe5eb;
				--accent: #1768b7;
				--accent-fg: #ffffff;
				--success: #19734b;
				--error: #a6323d;
				--radius: 8px;
				--font: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
			}
			@media (prefers-color-scheme: dark) {
				:root {
					--bg: #111820;
					--surface: #17212b;
					--surface-2: #1b2732;
					--fg: #e6edf4;
					--fg-dim: #9aacbd;
					--border: #2d3b49;
					--accent: #62a8e8;
					--accent-fg: #111820;
				}
			}`,
		},
	}); err != nil {
		return nil, "", fmt.Errorf("initialize Wails updater: %w", err)
	}
	return func(ctx context.Context) error {
		log.Printf("Zero1Local Desktop: checking GitHub Releases for updates (installed %s)", currentVersion)
		return app.Updater.CheckAndInstall(ctx)
	}, "", nil
}
