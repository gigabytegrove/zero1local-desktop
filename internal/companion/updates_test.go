package companion

import (
 "context"
 "errors"
 "strings"
 "testing"
 "time"
)

func awaitUpdate(t *testing.T,svc *Server,condition func(UpdateStatus)bool)UpdateStatus{
 t.Helper()
 deadline:=time.Now().Add(3*time.Second)
 for time.Now().Before(deadline){
  status:=svc.UpdateStatus()
  if condition(status){return status}
  time.Sleep(5*time.Millisecond)
 }
 t.Fatalf("update state did not converge: %+v",svc.UpdateStatus())
 return UpdateStatus{}
}
func testRelease(version string)releaseSelection{
 return releaseSelection{release:publicRelease{TagName:"v"+version},
  asset:releaseAsset{Name:"Zero1Local-Desktop-Updater-linux-amd64.tar.gz",URL:"https://github.com/test/release.tar.gz",Size:123},
  checksums:releaseAsset{Name:"SHA256SUMS",URL:"https://github.com/test/SHA256SUMS",Size:72}}
}
func TestCheckingDoesNotInstallAndPortableUpdateIsExplicit(t *testing.T){
 svc,err:=New(tempStore(t),nil)
 if err!=nil{t.Fatal(err)}
 calls:=0
 svc.ConfigureUpdater("0.1.0-beta.6","beta",func(context.Context)error{calls++;return nil},"")
 started:=make(chan struct{});finish:=make(chan struct{})
 svc.updates.selectRelease=func(ctx context.Context,installed,mode string)(releaseSelection,bool,error){
  if installed!="0.1.0-beta.6"||!strings.HasPrefix(mode,"portable-"){return releaseSelection{},false,errors.New("wrong release search")}
  close(started)
  select{case <-finish:return testRelease("0.1.0-beta.7"),true,nil;case <-ctx.Done():return releaseSelection{},false,ctx.Err()}
 }
 if err:=svc.StartUpdateCheck();err!=nil{t.Fatal(err)}
 <-started
 if err:=svc.StartUpdateCheck();err==nil{t.Fatal("parallel update check started")}
 if calls!=0{t.Fatal("checking unexpectedly installed a release")}
 close(finish)
 status:=awaitUpdate(t,svc,func(u UpdateStatus)bool{return !u.Checking&&u.Phase=="available"})
 if !status.Available||status.LatestVersion!="0.1.0-beta.7"{t.Fatalf("did not report available version: %+v",status)}
 if calls!=0{t.Fatal("check invoked installer")}
 if err:=svc.StartUpdateInstall();err!=nil{t.Fatal(err)}
 status=awaitUpdate(t,svc,func(u UpdateStatus)bool{return !u.Installing&&u.Phase=="complete"})
 if calls!=1||status.LastError!=""{t.Fatalf("unexpected installation outcome: calls=%d %+v",calls,status)}
}
func TestDebianInstallationIsAvailableWithoutBinaryReplacement(t *testing.T){
 svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
 svc.ConfigureUpdater("0.1.0-beta.6","beta",nil,"system-installed Linux application")
 if svc.UpdateStatus().Supported{t.Fatal("unconfigured installed package claims native updater is available")}
 svc.ConfigureDebianInstaller(func(context.Context,string,string)error{return nil})
 if status:=svc.UpdateStatus();!status.Supported||status.InstallMode!="debian"{
  t.Fatalf("Debian package installer not recognized: %+v",status)
 }
 svc.updates.selectRelease=func(context.Context,string,string)(releaseSelection,bool,error){
  return releaseSelection{},false,nil
 }
 if err:=svc.StartUpdateCheck();err!=nil{t.Fatal(err)}
 status:=awaitUpdate(t,svc,func(u UpdateStatus)bool{return !u.Checking&&u.Phase=="current"})
 if status.Available||status.LastError!=""{t.Fatalf("unexpected Debian state: %+v",status)}
 if err:=svc.StartUpdateInstall();err==nil{t.Fatal("installation without a newer selected release was allowed")}
}
func TestUpdaterFailureIsShownInSettings(t *testing.T){
 svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
 svc.ConfigureUpdater("0.1.0-beta.6","beta",func(context.Context)error{return nil},"")
 svc.updates.selectRelease=func(context.Context,string,string)(releaseSelection,bool,error){
  return releaseSelection{},false,errors.New("GitHub is unreachable")
 }
 if err:=svc.StartUpdateCheck();err!=nil{t.Fatal(err)}
 status:=awaitUpdate(t,svc,func(u UpdateStatus)bool{return !u.Checking&&u.Phase=="error"})
 if status.LastError!="GitHub is unreachable"||status.LastCheck.IsZero(){t.Fatalf("update check did not record error: %+v",status)}
}
func TestRestartRequiresSuccessfulDebianInstall(t *testing.T){
 svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
 svc.origin="http://127.0.0.1:8124"
 calls:=0
 svc.ConfigureUpdateRestart(func()error{calls++;return nil})
 forbidden:=nasTestRequest(t,svc,"POST","/api/updates/restart",map[string]any{})
 if forbidden.Code==202||calls!=0{t.Fatal("restart allowed without installed update")}
 svc.updates.mu.Lock()
 svc.updates.state.RestartReady=true
 svc.updates.mu.Unlock()
 allowed:=nasTestRequest(t,svc,"POST","/api/updates/restart",map[string]any{})
 if allowed.Code!=202||calls!=1{t.Fatalf("approved restart failed: %d %s",allowed.Code,allowed.Body.String())}
}
func TestUpdaterUnavailableForPreview(t *testing.T){
 svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
 if svc.UpdateStatus().Supported{t.Fatal("preview updater claimed native install support")}
 if err:=svc.StartUpdateCheck();err==nil{t.Fatal("preview allowed update check without a native runtime")}
}
