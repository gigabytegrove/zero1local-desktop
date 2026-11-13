package companion

import (
	"bufio"
	"bytes"
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

var debianAssetPattern = regexp.MustCompile(`^Zero1Local-Desktop-[0-9a-f]{12}-linux-amd64\.deb$`)
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


// findUpdateRelease is independent of the installer. Package-managed installs
// therefore get the exact same working Check for updates control as portable
// Windows and Linux applications.
func findUpdateRelease(ctx context.Context,installed,mode string)(releaseSelection,bool,error){
 client:=publicReleaseClient()
 request,err:=http.NewRequestWithContext(ctx,http.MethodGet,releaseAPI,nil)
 if err!=nil{return releaseSelection{},false,err}
 request.Header.Set("Accept","application/vnd.github+json")
 request.Header.Set("User-Agent","Zero1Local-Desktop-Updater")
 response,err:=client.Do(request)
 if err!=nil{return releaseSelection{},false,err}
 defer response.Body.Close()
 if response.StatusCode!=200{return releaseSelection{},false,fmt.Errorf("GitHub Releases returned HTTP %d",response.StatusCode)}
 raw,err:=io.ReadAll(io.LimitReader(response.Body,(2<<20)+1))
 if err!=nil{return releaseSelection{},false,err}
 if len(raw)>2<<20{return releaseSelection{},false,errors.New("release list exceeds safety limit")}
 var releases []publicRelease
 if err:=json.Unmarshal(raw,&releases);err!=nil{return releaseSelection{},false,fmt.Errorf("invalid GitHub release list: %w",err)}
 return chooseUpdateRelease(releases,installed,mode)
}
func chooseUpdateRelease(releases []publicRelease,installed,mode string)(releaseSelection,bool,error){
 if _,err:=releaseVersionParts(installed);err!=nil{return releaseSelection{},false,err}
 var match releaseSelection
 found:=false
 for _,release:=range releases{
  if release.Draft || !versionTagPattern.MatchString(release.TagName){continue}
  // Match the beta channel but never unexpectedly move to alpha.
  if strings.Contains(release.TagName,"-alpha.") {continue}
  cmp,err:=releaseVersionCompare(release.TagName,installed)
  if err!=nil||cmp<=0{continue}
  if found{
   newer,err:=releaseVersionCompare(release.TagName,match.release.TagName)
   if err!=nil||newer<=0{continue}
  }
  want:=""
  if mode=="debian"{
   // The Debian package embeds the exact source revision in its asset name.
   // We accept only the single matching .deb in a release.
  }else if strings.HasPrefix(mode,"portable-"){
   want="Zero1Local-Desktop-Updater-"+strings.TrimPrefix(mode,"portable-")
   if strings.HasPrefix(mode,"portable-windows-")||strings.HasPrefix(mode,"portable-darwin-"){want+=".zip"}else{want+=".tar.gz"}
  }else{continue}
  var asset,checksums releaseAsset
  matches:=0
  for _,a:=range release.Assets{
   if a.Name=="SHA256SUMS"{checksums=a}
   if (mode=="debian"&&debianAssetPattern.MatchString(a.Name)) || (want!=""&&a.Name==want){asset=a;matches++}
  }
  if matches!=1||asset.Size<=0||asset.Size>debianAssetLimit||checksums.Name==""{continue}
  if allowedReleaseURL(asset.URL)!=nil||allowedReleaseURL(checksums.URL)!=nil{continue}
  match=releaseSelection{release:release,asset:asset,checksums:checksums}
  found=true
 }
 return match,found,nil
}

// downloadVerifiedDebian streams the pinned GitHub release package into a
// mode-0700 private working directory. It never uses ~/Downloads and never
// installs unless the release checksum and size both match.
func downloadVerifiedDebian(ctx context.Context,selected releaseSelection,progress func(string,string))(string,func(),error){
 return downloadVerifiedDebianWithClient(ctx,selected,publicReleaseClient(),progress)
}
func downloadVerifiedDebianWithClient(ctx context.Context,selected releaseSelection,client *http.Client,progress func(string,string))(string,func(),error){
 cleanup:=func(){}
 if !debianAssetPattern.MatchString(selected.asset.Name){return "",cleanup,errors.New("release does not contain a valid Linux amd64 Debian package")}
 if selected.asset.Size<=0||selected.asset.Size>debianAssetLimit{return "",cleanup,errors.New("Debian asset is too large or missing its size")}
 if allowedReleaseURL(selected.asset.URL)!=nil||allowedReleaseURL(selected.checksums.URL)!=nil{return "",cleanup,errors.New("release asset URL is untrusted")}
 if progress!=nil{progress("verifying","Verifying the official release checksum…")}
 checksums,err:=downloadReleaseBody(ctx,client,selected.checksums.URL,128<<10)
 if err!=nil{return "",cleanup,fmt.Errorf("download release SHA-256 checksums: %w",err)}
 expected,err:=digestForAsset(string(checksums),selected.asset.Name)
 if err!=nil{return "",cleanup,err}
 dir,err:=os.MkdirTemp("","zero1local-desktop-update-*")
 if err!=nil{return "",cleanup,err}
 cleanup=func(){_ = os.RemoveAll(dir)}
 if err:=os.Chmod(dir,0700);err!=nil{cleanup();return "",func(){},err}
 name:=filepath.Join(dir,selected.asset.Name)
 output,err:=os.OpenFile(name,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600)
 if err!=nil{cleanup();return "",func(){},err}
 if progress!=nil{progress("downloading","Downloading the signed-off, checksum-protected Debian release…")}
 request,err:=http.NewRequestWithContext(ctx,http.MethodGet,selected.asset.URL,nil)
 if err!=nil{output.Close();cleanup();return "",func(){},err}
 request.Header.Set("User-Agent","Zero1Local-Desktop-Updater")
 response,err:=client.Do(request)
 if err!=nil{output.Close();cleanup();return "",func(){},err}
 if response.StatusCode!=http.StatusOK{
  response.Body.Close();output.Close();cleanup()
  return "",func(){},fmt.Errorf("GitHub Debian package returned HTTP %d",response.StatusCode)
 }
 sum:=sha256.New()
 written,copyErr:=io.Copy(io.MultiWriter(output,sum),io.LimitReader(response.Body,debianAssetLimit+1))
 closeResponseErr:=response.Body.Close()
 syncErr:=output.Sync()
 closeErr:=output.Close()
 if copyErr!=nil||closeResponseErr!=nil||syncErr!=nil||closeErr!=nil{
  cleanup();return "",func(){},errors.Join(copyErr,closeResponseErr,syncErr,closeErr)
 }
 if written!=selected.asset.Size||written>debianAssetLimit{cleanup();return "",func(){},errors.New("downloaded Debian package size differs from GitHub's release metadata")}
 if !bytes.Equal(sum.Sum(nil),expected){cleanup();return "",func(){},errors.New("Debian package SHA-256 verification failed; nothing was installed")}
 if progress!=nil{progress("verified","Verified package checksum and size. Preparing the system installer…")}
 return name,cleanup,nil
}
