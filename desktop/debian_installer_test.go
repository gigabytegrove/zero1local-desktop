package main

import(
 "context"
 "os"
 "path/filepath"
 "testing"
)

func TestDebianMetadataRejectsWrongIdentity(t *testing.T){
 raw:="Package: zero1local-desktop\nArchitecture: amd64\nVersion: 0.1.0~beta7+git123456789abc\n"
 fields:=parseDebianMetadata(raw)
 if fields["Package"]!="zero1local-desktop"||fields["Architecture"]!="amd64"||fields["Version"]!="0.1.0~beta7+git123456789abc"{
  t.Fatalf("unexpected dpkg-deb field parse: %#v",fields)
 }
 if len(parseDebianMetadata(raw+"Version: 0.1.0~beta6+git000000000000\n"))!=0{
  t.Fatal("duplicate control fields were accepted")
 }
 if len(parseDebianMetadata("Package: malicious\n"+raw))!=0{
  t.Fatal("overlapping package identities were accepted")
 }
}

func TestExpectedDebianVersionAndAssetIdentity(t *testing.T){
 asset:="Zero1Local-Desktop-123456789abc-linux-amd64.deb"
 version,err:=expectedDebianPackageVersion("0.1.0-beta.7",asset)
 if err!=nil||version!="0.1.0~beta7+git123456789abc"{
  t.Fatalf("wrong Debian version: %q, %v",version,err)
 }
 for _,name:=range []string{
  "../Zero1Local-Desktop-123456789abc-linux-amd64.deb",
  "Zero1Local-Desktop-notasha-linux-amd64.deb",
  "Zero1Local-Desktop-123456789abc-windows-amd64.zip",
 }{
  if _,err:=expectedDebianPackageVersion("0.1.0-beta.7",name);err==nil{
   t.Fatalf("unexpected Debian asset accepted: %s",name)
  }
 }
 stable,err:=expectedDebianPackageVersion("0.1.0","Zero1Local-Desktop-123456789abc-linux-amd64.deb")
 if err!=nil||stable!="0.1.0+git123456789abc"{
  t.Fatalf("stable upgrade version mismatch: %s, %v",stable,err)
 }
 if _,err:=expectedDebianPackageVersion("0.1.0-beta.7","Zero1Local-Desktop-badsha-linux-amd64.deb");err==nil{
  t.Fatal("unrecognized Debian source revision was accepted")
 }
}

func TestUnverifiedPackageIsNeverInstalled(t *testing.T){
 file:=filepath.Join(t.TempDir(),"Zero1Local-Desktop-123456789abc-linux-amd64.deb")
 if err:=os.WriteFile(file,[]byte("untrusted data"),0600);err!=nil{t.Fatal(err)}
 if err:=validateDebianUpgrade(context.Background(),file,"0.1.0-beta.7");err==nil{
  t.Fatal("unverified package content was accepted")
 }
}
