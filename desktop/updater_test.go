package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateAssetNames(t *testing.T) {
	tests := []struct{ goos, goarch, want string }{
		{"windows", "amd64", "Zero1Local-Desktop-Updater-windows-amd64.zip"},
		{"linux", "amd64", "Zero1Local-Desktop-Updater-linux-amd64.tar.gz"},
		{"darwin", "arm64", "Zero1Local-Desktop-Updater-darwin-arm64.zip"},
		{"darwin", "amd64", "Zero1Local-Desktop-Updater-darwin-amd64.zip"},
		{"windows", "386", ""},
		{"linux", "arm64", ""},
	}
	for _, test := range tests {
		if got := updateArtifactName(test.goos, test.goarch); got != test.want {
			t.Errorf("updateArtifactName(%q,%q) = %q, want %q", test.goos, test.goarch, got, test.want)
		}
	}
}

func TestWritableInstallationDirectory(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "zero1local-desktop")
	if err := os.WriteFile(executable, []byte("test"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := writableInstallDirectory("linux", executable); err != nil {
		t.Fatal(err)
	}
	list, err := os.ReadDir(root)
	if err != nil || len(list) != 1 {
		t.Fatalf("writable directory left temporary content: %v / %v", list, err)
	}
	for _, installed := range []string{"/usr/bin/zero1local-desktop", "/opt/zero1local-desktop/app"} {
		if err := writableInstallDirectory("linux", installed); err == nil ||
			!strings.Contains(err.Error(), "package manager") {
			t.Errorf("system-installed application %q is not protected: %v", installed, err)
		}
	}
}
