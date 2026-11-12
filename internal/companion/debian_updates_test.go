package companion

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestReleaseVersions(t *testing.T) {
	for _, test := range []struct {
		first, second string
		result int
	}{
		{"v0.1.0-beta.2", "v0.1.0-beta.1", 1},
		{"v0.1.0-beta.2", "0.1.0-beta.2", 0},
		{"v0.1.0", "v0.1.0-rc.99", 1},
		{"v0.1.1-alpha.1", "v0.1.0", 1},
		{"v0.1.0-beta.2", "v0.1.0-beta.3", -1},
	} {
		result, err := releaseVersionCompare(test.first, test.second)
		if err != nil || result != test.result {
			t.Errorf("%s vs %s: %d, %v", test.first, test.second, result, err)
		}
	}
	if _, err := releaseVersionCompare("v0.1.0-unknown.2", "v0.1.0"); err == nil {
		t.Fatal("unsupported version accepted")
	}
}

func TestReleaseChecksumParser(t *testing.T) {
	digest := strings.Repeat("a", 64)
	got, err := digestForAsset(digest+"  Zero1Local-Desktop-abcdef-linux-amd64.deb\n", "Zero1Local-Desktop-abcdef-linux-amd64.deb")
	if err != nil || hex.EncodeToString(got) != digest {
		t.Fatalf("valid checksum rejected: %v", err)
	}
	if _, err := digestForAsset(digest+"  other.deb", "zero1local.deb"); err == nil {
		t.Fatal("missing checksum accepted")
	}
}

func TestReleaseSelection(t *testing.T) {
	makeRelease := func(version string, assets bool) publicRelease {
		r := publicRelease{TagName: version}
		if assets {
			r.Assets = []releaseAsset{
				{Name: "Zero1Local-Desktop-abcdef-linux-amd64.deb", Size: 20, URL: "https://github.com/gigabytegrove/zero1local-desktop/releases/download/" + version + "/test.deb"},
				{Name: "SHA256SUMS", Size: 120, URL: "https://github.com/gigabytegrove/zero1local-desktop/releases/download/" + version + "/SHA256SUMS"},
			}
		}
		return r
	}
	latest, _, _, found := findLatestDeb([]publicRelease{
		makeRelease("v0.1.0-beta.1", true),
		makeRelease("v0.1.0-beta.3", false),
		makeRelease("v0.1.0-beta.2", true),
	}, "0.1.0-beta.1")
	if !found || latest.TagName != "v0.1.0-beta.2" {
		t.Fatalf("selected unexpected Debian release: %+v", latest)
	}
}

func TestTrustedReleaseDomains(t *testing.T) {
	for _, raw := range []string{
		"http://github.com/release",
		"https://github.com.evil.example/release",
		"https://evil.githubusercontent.com.example/release",
	} {
		if allowedReleaseURL(raw) == nil {
			t.Errorf("accepted unsafe release URL %q", raw)
		}
	}
	if allowedReleaseURL("https://release-assets.githubusercontent.com/resource") != nil {
		t.Fatal("legitimate GitHub asset redirect rejected")
	}
}
