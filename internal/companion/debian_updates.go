package companion

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const releaseAPI = "https://api.github.com/repos/gigabytegrove/zero1local-desktop/releases?per_page=30"
const debianAssetLimit int64 = 256 << 20

var debianAssetPattern = regexp.MustCompile(`^Zero1Local-Desktop-[A-Za-z0-9._+~-]+-linux-amd64\.deb$`)
var versionTagPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-(alpha|beta|rc)\.(\d+))?$`)

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}
type publicRelease struct {
	TagName string         `json:"tag_name"`
	Draft   bool           `json:"draft"`
	Assets  []releaseAsset `json:"assets"`
}
type packageDownload struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

func releaseVersionParts(tag string) ([5]int, error) {
	m := versionTagPattern.FindStringSubmatch(tag)
	var out [5]int
	if m == nil {
		return out, fmt.Errorf("unsupported release version %q", tag)
	}
	for i := 1; i <= 3; i++ {
		n, err := strconv.Atoi(m[i])
		if err != nil || n > 999999 {
			return out, errors.New("invalid version number")
		}
		out[i-1] = n
	}
	switch m[4] {
	case "alpha":
		out[3] = 0
	case "beta":
		out[3] = 1
	case "rc":
		out[3] = 2
	case "":
		out[3] = 3
	default:
		return out, errors.New("unknown release channel")
	}
	if m[5] != "" {
		n, err := strconv.Atoi(m[5])
		if err != nil || n > 999999 {
			return out, errors.New("invalid prerelease sequence")
		}
		out[4] = n
	}
	return out, nil
}
func releaseVersionCompare(a, b string) (int, error) {
	x, err := releaseVersionParts(a)
	if err != nil {
		return 0, err
	}
	y, err := releaseVersionParts(b)
	if err != nil {
		return 0, err
	}
	for i := range x {
		if x[i] < y[i] {
			return -1, nil
		}
		if x[i] > y[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func allowedReleaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" {
		return errors.New("untrusted release download URL")
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && !strings.HasSuffix(host, ".githubusercontent.com") {
		return errors.New("release download redirected outside GitHub")
	}
	return nil
}

func publicReleaseClient() *http.Client {
	return &http.Client{
		Timeout: 8 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 6 {
				return errors.New("too many release redirects")
			}
			return allowedReleaseURL(req.URL.String())
		},
	}
}

func downloadReleaseBody(ctx context.Context, client *http.Client, urlString string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlString, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Zero1Local-Desktop-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub release request returned HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, errors.New("release response exceeded its limit")
	}
	return content, nil
}

func digestForAsset(manifest, asset string) ([]byte, error) {
	scanner := bufio.NewScanner(strings.NewReader(manifest))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != asset {
			continue
		}
		checksum := strings.ToLower(fields[0])
		if len(checksum) != 64 {
			return nil, errors.New("malformed release SHA256SUMS entry")
		}
		expected, err := hex.DecodeString(checksum)
		if err != nil {
			return nil, errors.New("malformed SHA-256 digest")
		}
		return expected, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("release checksum manifest does not include the Debian package")
}

func findLatestDeb(releases []publicRelease, currentVersion string) (publicRelease, releaseAsset, releaseAsset, bool) {
	var chosen publicRelease
	var deb releaseAsset
	var checksum releaseAsset
	found := false
	for _, release := range releases {
		if release.Draft {
			continue
		}
		cmp, err := releaseVersionCompare(release.TagName, currentVersion)
		if err != nil || cmp <= 0 {
			continue
		}
		if found {
			order, err := releaseVersionCompare(release.TagName, chosen.TagName)
			if err != nil || order <= 0 {
				continue
			}
		}
		var newDeb, newChecksum releaseAsset
		for _, asset := range release.Assets {
			if asset.Name == "SHA256SUMS" {
				newChecksum = asset
			}
			if debianAssetPattern.MatchString(asset.Name) {
				newDeb = asset
			}
		}
		if newDeb.Name == "" || newChecksum.Name == "" || newDeb.Size <= 0 || newDeb.Size > debianAssetLimit {
			continue
		}
		if allowedReleaseURL(newDeb.URL) != nil || allowedReleaseURL(newChecksum.URL) != nil {
			continue
		}
		chosen, deb, checksum, found = release, newDeb, newChecksum, true
	}
	return chosen, deb, checksum, found
}

// downloadDebianPackage checks a public GitHub release, downloads the exact
// Linux amd64 Debian package and validates it against the release SHA256SUMS.
// No package installation or privilege escalation occurs in this function.
func downloadDebianPackage(ctx context.Context, version string) (packageDownload, error) {
	client := publicReleaseClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPI, nil)
	if err != nil {
		return packageDownload{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Zero1Local-Desktop-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return packageDownload{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return packageDownload{}, fmt.Errorf("GitHub Releases returned HTTP %d", resp.StatusCode)
	}
	var releases []publicRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&releases); err != nil {
		return packageDownload{}, fmt.Errorf("decode releases: %w", err)
	}
	release, deb, checksumAsset, found := findLatestDeb(releases, version)
	if !found {
		return packageDownload{Status: "current", Message: "No newer Linux Debian package is published."}, nil
	}
	checksums, err := downloadReleaseBody(ctx, client, checksumAsset.URL, 128<<10)
	if err != nil {
		return packageDownload{}, fmt.Errorf("fetch release checksums: %w", err)
	}
	expected, err := digestForAsset(string(checksums), deb.Name)
	if err != nil {
		return packageDownload{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return packageDownload{}, err
	}
	downloadDir := filepath.Join(home, "Downloads")
	info, err := os.Stat(downloadDir)
	if err != nil || !info.IsDir() {
		return packageDownload{}, errors.New("Downloads folder does not exist; create ~/Downloads or use the GitHub Releases page")
	}
	part, err := os.CreateTemp(downloadDir, ".zero1local-release-*.part")
	if err != nil {
		return packageDownload{}, err
	}
	partName := part.Name()
	defer os.Remove(partName)
	defer part.Close()
	if err := part.Chmod(0600); err != nil {
		return packageDownload{}, err
	}
	downloadReq, err := http.NewRequestWithContext(ctx, http.MethodGet, deb.URL, nil)
	if err != nil {
		return packageDownload{}, err
	}
	downloadReq.Header.Set("User-Agent", "Zero1Local-Desktop-Updater")
	packageResp, err := client.Do(downloadReq)
	if err != nil {
		return packageDownload{}, err
	}
	defer packageResp.Body.Close()
	if packageResp.StatusCode != http.StatusOK {
		return packageDownload{}, fmt.Errorf("Debian asset returned HTTP %d", packageResp.StatusCode)
	}
	sum := sha256.New()
	written, err := io.Copy(io.MultiWriter(part, sum), io.LimitReader(packageResp.Body, debianAssetLimit+1))
	if err != nil {
		return packageDownload{}, err
	}
	if written != deb.Size || written > debianAssetLimit {
		return packageDownload{}, errors.New("Debian package size did not match the release asset")
	}
	if !strings.EqualFold(hex.EncodeToString(sum.Sum(nil)), hex.EncodeToString(expected)) {
		return packageDownload{}, errors.New("Debian package failed SHA-256 verification")
	}
	if err := part.Sync(); err != nil {
		return packageDownload{}, err
	}
	if err := part.Close(); err != nil {
		return packageDownload{}, err
	}
	target := filepath.Join(downloadDir, deb.Name)
	// Link rather than rename: it fails safely if a user file already exists,
	// instead of overwriting a previous download or a symlink.
	if err := os.Link(partName, target); err != nil {
		return packageDownload{}, fmt.Errorf("save verified package without overwriting an existing file: %w", err)
	}
	return packageDownload{
		Status: "downloaded",
		Version: strings.TrimPrefix(release.TagName, "v"),
		Path: target,
		Message: "SHA-256 verified package saved. Install it with your distribution's package manager; the application does not request root access.",
	}, nil
}

func (s *Server) downloadDebianAPI(w http.ResponseWriter, r *http.Request) {
	s.updates.mu.Lock()
	if !s.updates.state.PackageDownload || s.updates.state.Checking {
		s.updates.mu.Unlock()
		s.bad(w, errors.New("Debian package download is unavailable or an update operation is already running"))
		return
	}
	s.updates.state.Checking = true
	s.updates.state.LastError = ""
	version := s.updates.state.Version
	s.updates.mu.Unlock()

	result, err := downloadDebianPackage(r.Context(), version)
	s.updates.mu.Lock()
	s.updates.state.Checking = false
	s.updates.state.LastCheck = time.Now().UTC()
	if err != nil {
		s.updates.state.LastError = err.Error()
	}
	s.updates.mu.Unlock()

	if err != nil {
		s.bad(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}
