package main

import (
 "context"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "runtime"
 "strings"
 "time"

 "github.com/wailsapp/wails/v3/pkg/application"
)

// A Debian install is recognized by the actual running executable and dpkg's
// installation records, never by an arbitrary file-system path chosen by UI.
func nativeDebianInstall()bool{
 if runtime.GOOS!="linux"||runtime.GOARCH!="amd64"{return false}
 executable,err:=os.Executable()
 if err!=nil{return false}
 resolved,err:=filepath.EvalSymlinks(executable)
 if err!=nil{return false}
 if resolved!="/usr/bin/zero1local-desktop"{return false}
 out,err:=exec.Command("/usr/bin/dpkg-query","-W","-f=${Status}","zero1local-desktop").Output()
 return err==nil && strings.TrimSpace(string(out))=="install ok installed"
}
var sourceInDebAsset=regexp.MustCompile(`^Zero1Local-Desktop-([0-9a-f]{12})-linux-amd64\.deb$`)
func expectedDebianPackageVersion(releaseVersion,assetName string)(string,error){
 name:=sourceInDebAsset.FindStringSubmatch(assetName)
 if name==nil{return "",errors.New("unexpected Debian asset filename")}
 m:=regexp.MustCompile(`^(\d+\.\d+\.\d+)-(alpha|beta|rc)\.(\d+)$`).FindStringSubmatch(releaseVersion)
 if m==nil{return "",errors.New("invalid Debian prerelease version")}
 return m[1]+"~"+m[2]+m[3]+"+git"+name[1],nil
}
func validateDebianUpgrade(ctx context.Context,filename,releaseVersion string)error{
 if !filepath.IsAbs(filename){return errors.New("updater package path must be absolute")}
 info,err:=os.Lstat(filename)
 if err!=nil{return err}
 if !info.Mode().IsRegular()||info.Size()<=0{return errors.New("update is not a regular Debian package")}
 expected,err:=expectedDebianPackageVersion(releaseVersion,filepath.Base(filename))
 if err!=nil{return err}
 cmd:=exec.CommandContext(ctx,"/usr/bin/dpkg-deb","--field",filename,"Package","Architecture","Version")
 result,err:=cmd.Output()
 if err!=nil{return fmt.Errorf("cannot inspect downloaded Debian package: %w",err)}
 values:=strings.Fields(string(result))
 if len(values)!=3||values[0]!="zero1local-desktop"||values[1]!="amd64"||values[2]!=expected{
  return fmt.Errorf("verified release package metadata does not match the requested version %s",releaseVersion)
 }
 return nil
}

// Caller already verified the exact release SHA-256 manifest and its size.
// PolicyKit opens the familiar GUI authentication prompt; APT retains dpkg's
// package database, dependencies and rollback/recovery semantics.
// No sudo password is ever handled by this application.
func installNativeDebianPackage(ctx context.Context,filename,releaseVersion string)error{
 if !nativeDebianInstall(){return errors.New("this Zero1Local Desktop installation is not managed by Debian")}
 if err:=validateDebianUpgrade(ctx,filename,releaseVersion);err!=nil{return err}
 if _,err:=os.Stat("/usr/bin/pkexec");err!=nil{
  return errors.New("graphical administrator authorization (pkexec) is not installed; enable PolicyKit to use in-app Debian updates")
 }
 command:=exec.CommandContext(ctx,"/usr/bin/pkexec","/usr/bin/apt-get","install",
  "-y","--no-remove","--no-install-recommends",filename)
 output,err:=command.CombinedOutput()
 if err!=nil{
  detail:=strings.TrimSpace(string(output))
  if len(detail)>550{detail=detail[len(detail)-550:]}
  return fmt.Errorf("administrator authorization or package installation failed: %v: %s",err,detail)
 }
 // Verify apt/dpkg agree that the exact requested release is installed.
 installed,err:=exec.CommandContext(ctx,"/usr/bin/dpkg-query","-W",
  "-f=${Version}","zero1local-desktop").Output()
 if err!=nil||strings.TrimSpace(string(installed))!=mustExpectedVersion(releaseVersion,filepath.Base(filename)){
  return errors.New("package manager finished but the requested version was not confirmed as installed")
 }
 return nil
}
func mustExpectedVersion(version,asset string)string{
 value,err:=expectedDebianPackageVersion(version,asset)
 if err!=nil{return ""}
 return value
}
// Schedule restart after the local HTTP response is sent and after the old
// single-instance lock has been released; never launch a second copy first.
func scheduleDesktopRestart(app *application.App)error{
 executable,err:=os.Executable()
 if err!=nil{return err}
 if executable!="/usr/bin/zero1local-desktop"{
  return errors.New("automatic restart requires the installed Debian desktop executable")
 }
 command:=exec.Command("/bin/sh","-c",`sleep 2; exec "$1"`,"zero1local-restart",executable)
 command.Stdin=nil
 command.Stdout=nil
 command.Stderr=nil
 if err:=command.Start();err!=nil{return fmt.Errorf("schedule desktop restart: %w",err)}
 if err:=command.Process.Release();err!=nil{return err}
 go func(){time.Sleep(250*time.Millisecond);app.Quit()}()
 return nil
}
