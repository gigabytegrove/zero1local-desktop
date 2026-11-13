package companion

import (
 "bytes"
 "context"
 "crypto/sha256"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"
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

type updaterTransport func(*http.Request)(*http.Response,error)
func (f updaterTransport)RoundTrip(r *http.Request)(*http.Response,error){return f(r)}
func TestCompatibleReleaseSelectionForDebianAndWindows(t *testing.T){
 makeRelease:=func(tag string,assets ...string)publicRelease{
  release:=publicRelease{TagName:tag}
  for _,name:=range assets{
   release.Assets=append(release.Assets,releaseAsset{
    Name:name,Size:32,URL:"https://github.com/gigabytegrove/zero1local-desktop/releases/download/"+tag+"/"+name,
   })
  }
  return release
 }
 latestDeb:="Zero1Local-Desktop-123456789abc-linux-amd64.deb"
 releases:=[]publicRelease{
  makeRelease("v0.1.0-beta.7","SHA256SUMS",latestDeb,
   "Zero1Local-Desktop-Updater-windows-amd64.zip"),
  makeRelease("v0.1.0-beta.8","SHA256SUMS"), // Ignore incomplete release.
  makeRelease("v0.1.0-beta.6","SHA256SUMS",latestDeb),
  makeRelease("v0.1.0-alpha.9","SHA256SUMS",latestDeb),
 }
 deb,found,err:=chooseUpdateRelease(releases,"0.1.0-beta.6","debian")
 if err!=nil||!found||deb.release.TagName!="v0.1.0-beta.7"||deb.asset.Name!=latestDeb{
  t.Fatalf("unexpected compatible Debian release: %+v, %v %v",deb,found,err)
 }
 win,found,err:=chooseUpdateRelease(releases,"0.1.0-beta.6","portable-windows-amd64")
 if err!=nil||!found||win.asset.Name!="Zero1Local-Desktop-Updater-windows-amd64.zip"{
  t.Fatalf("unexpected compatible Windows updater: %+v %v %v",win,found,err)
 }
 if _,found,err=chooseUpdateRelease(releases,"0.1.0-beta.7","portable-windows-amd64");err!=nil||found{
  t.Fatal("installed latest release was incorrectly reported as upgradable")
 }
}
func TestVerifiedDebianDownloadStaysPrivateAndDetectsTampering(t *testing.T){
 contents:=[]byte("a synthetic Debian test package; it is never executed")
 asset:="Zero1Local-Desktop-123456789abc-linux-amd64.deb"
 digest:=sha256.Sum256(contents)
 manifest:=fmt.Sprintf("%s  %s\n",hex.EncodeToString(digest[:]),asset)
 selected:=releaseSelection{release:publicRelease{TagName:"v0.1.0-beta.7"},
  asset:releaseAsset{Name:asset,URL:"https://github.com/release/"+asset,Size:int64(len(contents))},
  checksums:releaseAsset{Name:"SHA256SUMS",URL:"https://github.com/release/SHA256SUMS",Size:int64(len(manifest))}}
 client:=&http.Client{Transport:updaterTransport(func(req *http.Request)(*http.Response,error){
  var value []byte
  if strings.HasSuffix(req.URL.Path,"SHA256SUMS"){value=[]byte(manifest)}else{value=contents}
  return &http.Response{StatusCode:200,Body:io.NopCloser(bytes.NewReader(value)),Header:make(http.Header),ContentLength:int64(len(value)),Request:req},nil
 })}
 file,cleanup,err:=downloadVerifiedDebianWithClient(context.Background(),selected,client,nil)
 if err!=nil{t.Fatal(err)}
 info,err:=os.Lstat(file)
 if err!=nil{t.Fatal(err)}
 if info.Mode().Perm()!=0600{t.Fatalf("downloaded package has unsafe mode %v",info.Mode())}
 folder,err:=os.Stat(filepath.Dir(file))
 if err!=nil||folder.Mode().Perm()!=0700{t.Fatalf("staging directory must be private: %v %v",folder,err)}
 contentsOnDisk,err:=os.ReadFile(file)
 if err!=nil||!bytes.Equal(contentsOnDisk,contents){t.Fatal("download bytes changed")}
 cleanup()
 if _,err:=os.Stat(file);!os.IsNotExist(err){t.Fatal("staging file not deleted after install attempt")}
 selected.asset.Size++
 if _,cleanup,err:=downloadVerifiedDebianWithClient(context.Background(),selected,client,nil);err==nil{
  cleanup();t.Fatal("mismatched package size was accepted")
 }
 selected.asset.Size--
 wrongChecksum:=fmt.Sprintf("%s  %s\n",strings.Repeat("0",64),asset)
 client.Transport=updaterTransport(func(req *http.Request)(*http.Response,error){
  value:=contents
  if strings.HasSuffix(req.URL.Path,"SHA256SUMS"){value=[]byte(wrongChecksum)}
  return &http.Response{StatusCode:200,Body:io.NopCloser(bytes.NewReader(value)),Header:make(http.Header),ContentLength:int64(len(value)),Request:req},nil
 })
 if _,cleanup,err:=downloadVerifiedDebianWithClient(context.Background(),selected,client,nil);err==nil{
  cleanup();t.Fatal("tampered release manifest was accepted")
 }
}
