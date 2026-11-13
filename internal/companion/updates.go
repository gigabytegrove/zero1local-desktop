package companion

import (
 "context"
 "errors"
 "fmt"
 "net/http"
 "runtime"
 "strings"
 "sync"
 "time"
)

const publicReleasesURL = "https://github.com/gigabytegrove/zero1local-desktop/releases"

// UpdateStatus is safe for the loopback frontend: it never contains a download
// credential, elevated command, or local package path.
type UpdateStatus struct {
 Version string `json:"version"`
 Channel string `json:"channel"`
 Supported bool `json:"supported"`
 CanInstall bool `json:"canInstall"`
 InstallMode string `json:"installMode"`
 Checking bool `json:"checking"`
 Installing bool `json:"installing"`
 Available bool `json:"available"`
 LatestVersion string `json:"latestVersion,omitempty"`
 Phase string `json:"phase"`
 Message string `json:"message"`
 LastError string `json:"lastError,omitempty"`
 LastCheck time.Time `json:"lastCheck,omitempty"`
 ReleaseURL string `json:"releaseUrl"`
 RestartReady bool `json:"restartReady"`
}
type releaseSelection struct {
 release publicRelease
 asset releaseAsset
 checksums releaseAsset
}
type updateControl struct {
 mu sync.Mutex
 state UpdateStatus
 portableInstall func(context.Context)error
 debianInstall func(context.Context,string,string)error
 restart func()error
 selectRelease func(context.Context,string,string)(releaseSelection,bool,error)
 cancel context.CancelFunc
 selection releaseSelection
}
func (s *Server) ConfigureUpdater(version,channel string,portableInstall func(context.Context)error,reason string){
 s.updates.mu.Lock()
 defer s.updates.mu.Unlock()
 s.updates.portableInstall=portableInstall
 mode:="unavailable"
 if portableInstall!=nil{mode="portable-"+s.goos+"-"+runtime.GOARCH}
 if s.updates.debianInstall!=nil{mode="debian"}
 supported:=version!=""&&version!="development preview"&&mode!="unavailable"
 s.updates.state=UpdateStatus{
  Version:version,Channel:channel,Supported:supported,CanInstall:supported,
  InstallMode:mode,Phase:"idle",Message:"Check GitHub Releases for a newer version.",
  ReleaseURL:publicReleasesURL,
 }
 if !supported && reason!=""{s.updates.state.Message=reason}
}
func (s *Server) ConfigureDebianInstaller(installer func(context.Context,string,string)error){
 s.updates.mu.Lock()
 defer s.updates.mu.Unlock()
 s.updates.debianInstall=installer
 if installer!=nil{
  s.updates.state.InstallMode="debian"
  s.updates.state.Supported=s.updates.state.Version!=""&&s.updates.state.Version!="development preview"
  s.updates.state.CanInstall=s.updates.state.Supported
  s.updates.state.Message="Ready to check for updates. Installation uses the system package manager."
 }
}
func (s *Server) ConfigureUpdateRestart(restart func()error){
 s.updates.mu.Lock()
 defer s.updates.mu.Unlock()
 s.updates.restart=restart
}
func (s *Server) UpdateStatus()UpdateStatus{
 s.updates.mu.Lock();defer s.updates.mu.Unlock()
 status:=s.updates.state
 if status.Version==""{
  status.Version="development preview";status.Channel="beta"
  status.ReleaseURL=publicReleasesURL
  status.Message="Updates are available in the native desktop application."
 }
 return status
}
func (s *Server) setUpdatePhase(phase,message string){
 s.updates.mu.Lock();defer s.updates.mu.Unlock()
 s.updates.state.Phase=phase;s.updates.state.Message=message
}
// StartUpdateCheck checks for new releases only. It never installs or opens an
// independent updater window, and it works for package-managed Linux installs.
func (s *Server) StartUpdateCheck()error{
 s.updates.mu.Lock()
 if !s.updates.state.Supported{s.updates.mu.Unlock();return errors.New("update checks require the native desktop application")}
 if s.updates.state.Checking||s.updates.state.Installing{s.updates.mu.Unlock();return errors.New("an update operation is already running")}
 ctx,cancel:=context.WithTimeout(context.Background(),40*time.Second)
 s.updates.cancel=cancel
 s.updates.state.Checking=true;s.updates.state.LastError=""
 s.updates.state.Available=false;s.updates.state.LatestVersion=""
 s.updates.state.Phase="checking";s.updates.state.Message="Checking GitHub Releases…"
 s.updates.selection=releaseSelection{}
 version,channel:=s.updates.state.Version,s.updates.state.Channel
 mode:=s.updates.state.InstallMode
 lookup:=s.updates.selectRelease
 if lookup==nil{lookup=findUpdateRelease}
 s.updates.mu.Unlock()
 go func(){
  selected,found,err:=lookup(ctx,version,mode)
  s.updates.mu.Lock()
  defer s.updates.mu.Unlock()
  s.updates.state.Checking=false;s.updates.state.LastCheck=time.Now().UTC()
  if err!=nil{
   s.updates.state.Phase="error"
   s.updates.state.LastError=err.Error()
   s.updates.state.Message="Could not check GitHub Releases."
  }else if found{
   s.updates.selection=selected
   s.updates.state.Available=true
   s.updates.state.LatestVersion=strings.TrimPrefix(selected.release.TagName,"v")
   s.updates.state.ReleaseURL=publicReleasesURL+"/tag/"+selected.release.TagName
   s.updates.state.Phase="available"
   s.updates.state.Message=fmt.Sprintf("Version %s is ready to install.",s.updates.state.LatestVersion)
  }else{
   s.updates.state.Phase="current"
   s.updates.state.Message="You're up to date. No newer compatible "+channel+" release was found."
   s.updates.state.ReleaseURL=publicReleasesURL
  }
  s.updates.cancel=nil
  cancel()
 }()
 return nil
}
func (s *Server) StartUpdateInstall()error{
 s.updates.mu.Lock()
 if !s.updates.state.Supported||!s.updates.state.CanInstall{s.updates.mu.Unlock();return errors.New("installation is not supported on this desktop")}
 if s.updates.state.Installing||s.updates.state.Checking{s.updates.mu.Unlock();return errors.New("an update operation is already running")}
 if !s.updates.state.Available||s.updates.selection.release.TagName==""{s.updates.mu.Unlock();return errors.New("check for a newer release before installing")}
 mode:=s.updates.state.InstallMode
 portable,debian:=s.updates.portableInstall,s.updates.debianInstall
 if (mode=="debian"&&debian==nil)||(strings.HasPrefix(mode,"portable-")&&portable==nil){s.updates.mu.Unlock();return errors.New("no installer is configured for this installation")}
 selected:=s.updates.selection
 ctx,cancel:=context.WithTimeout(context.Background(),18*time.Minute)
 s.updates.cancel=cancel
 s.updates.state.Installing=true;s.updates.state.LastError=""
 s.updates.state.Phase="preparing";s.updates.state.Message="Preparing verified update…"
 s.updates.mu.Unlock()
 go func(){
  var err error
  if mode=="debian"{
   var path string
   path,cleanup,downloadErr:=downloadVerifiedDebian(ctx,selected,func(phase,message string){s.setUpdatePhase(phase,message)})
   if downloadErr!=nil{err=downloadErr}else{
    defer cleanup()
    s.setUpdatePhase("authorizing","Requesting administrator permission to install the verified update…")
    err=debian(ctx,path,strings.TrimPrefix(selected.release.TagName,"v"))
   }
  }else{
   s.setUpdatePhase("installing","Opening the built-in updater. Confirm installation there.")
   err=portable(ctx)
  }
  s.updates.mu.Lock()
  defer s.updates.mu.Unlock()
  s.updates.state.Installing=false
  if err!=nil{
   s.updates.state.Phase="error";s.updates.state.LastError=err.Error()
   s.updates.state.Message="Update installation failed; your existing installation was not intentionally removed."
  }else if mode=="debian"{
   s.updates.state.Available=false
   s.updates.state.RestartReady=true;s.updates.state.Phase="restart"
   s.updates.state.Message="Update installed. Restart Zero1Local Desktop to use the new version."
  }else{
   s.updates.state.Phase="complete"
   s.updates.state.Message="The native updater completed. Restart if requested."
   s.updates.state.Available=false
  }
  s.updates.cancel=nil
  cancel()
 }()
 return nil
}
func(s *Server) stopUpdater(){
 s.updates.mu.Lock();cancel:=s.updates.cancel;s.updates.mu.Unlock()
 if cancel!=nil{cancel()}
}
func(s *Server) updateCheckAPI(w http.ResponseWriter,r *http.Request){
 if err:=s.StartUpdateCheck();err!=nil{s.bad(w,err);return}
 s.writeJSON(w,http.StatusAccepted,map[string]any{"started":true})
}
func(s *Server) updateInstallAPI(w http.ResponseWriter,r *http.Request){
 if err:=s.StartUpdateInstall();err!=nil{s.bad(w,err);return}
 s.writeJSON(w,http.StatusAccepted,map[string]any{"started":true})
}
func(s *Server) updateRestartAPI(w http.ResponseWriter,r *http.Request){
 s.updates.mu.Lock()
 ready:=s.updates.state.RestartReady
 restart:=s.updates.restart
 s.updates.mu.Unlock()
 if !ready||restart==nil{s.bad(w,errors.New("no installed update is waiting for restart"));return}
 if err:=restart();err!=nil{s.bad(w,err);return}
 s.writeJSON(w,http.StatusAccepted,map[string]any{"restarting":true})
}
func(s *Server) updateReleasesAPI(w http.ResponseWriter,r *http.Request){
 if s.launcher==nil{s.bad(w,errors.New("Open "+publicReleasesURL+" in your browser"));return}
 if err:=s.launcher(publicReleasesURL);err!=nil{s.bad(w,err);return}
 s.writeJSON(w,http.StatusOK,map[string]any{"opened":true})
}
