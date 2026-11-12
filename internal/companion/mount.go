package companion

import(
 "context"
 "errors"
 "fmt"
 "net"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "runtime"
 "strings"
 "time"
)

var sharePattern=regexp.MustCompile(`^[\pL\pN][\pL\pN _.$()-]{0,79}$`)
var targetPattern=regexp.MustCompile(`^[D-Z]:$`)
var nfsExportPattern=regexp.MustCompile(`^/[\pL\pN _./-]{1,254}$`)
var hostnamePattern=regexp.MustCompile(`^[\pL\pN.-]{1,253}$`)

type MountRequest struct {
 DeviceID string `json:"deviceId"`
 Protocol string `json:"protocol"`
 Server string `json:"server"`
 Remote string `json:"remote"`
 Target string `json:"target"`
}
type MountResult struct {
 Status string `json:"status"`
 Message string `json:"message"`
 Mapping *Mapping `json:"mapping,omitempty"`
}
type Runner interface {Run(context.Context,string,...string)(string,error)}
type OSRunner struct {}
func(OSRunner)Run(ctx context.Context,name string,args ...string)(string,error){
 cmd:=exec.CommandContext(ctx,name,args...)
 data,err:=cmd.CombinedOutput()
 if len(data)>4096{data=data[:4096]}
 return strings.TrimSpace(string(data)),err
}
func validateHost(host string)error{
 if strings.ContainsAny(host,"@\/?#%[] 	
") || !hostnamePattern.MatchString(host) || strings.HasPrefix(host,"-") || strings.HasSuffix(host,"."){
  return errors.New("invalid share host")
 }
 if ip:=net.ParseIP(host);ip!=nil&&ip.To4()==nil{return errors.New("IPv6 mapping unavailable in this release")}
 if strings.Contains(host,"..") || strings.Contains(host,"--"){return errors.New("invalid share hostname")}
 return nil
}
func validateRequest(req MountRequest,goos string)(MountRequest,error){
 req.Server=strings.TrimSpace(req.Server)
 req.Remote=strings.TrimSpace(req.Remote)
 req.Target=strings.TrimSpace(req.Target)
 if err:=validateHost(req.Server);err!=nil{return req,err}
 switch req.Protocol{
 case "smb":
  if !sharePattern.MatchString(req.Remote)||req.Remote=="."||req.Remote==".."{return req,errors.New("invalid SMB share name")}
 case "nfs":
  if !nfsExportPattern.MatchString(req.Remote)||strings.Contains(req.Remote,"//")||strings.Contains(req.Remote,"..")||strings.ContainsAny(req.Remote,":\
	"){
   return req,errors.New("invalid NFSv3 export path")
  }
 default:
  return req,errors.New("protocol must be smb or nfs")
 }
 switch goos{
 case "windows":
  if req.Protocol!="smb"{return req,errors.New("Windows NFSv3 requires a later verified mount adapter; use SMB")}
  req.Target=strings.ToUpper(req.Target)
  if !targetPattern.MatchString(req.Target){return req,errors.New("choose a free Windows drive letter D: through Z:")}
 case "linux":
  if req.Protocol=="smb"{
   if req.Target!=""{return req,errors.New("Linux GVfs SMB target must be blank")}
  }else if err:=validateUnixMountPath(req.Target);err!=nil{return req,err}
 case "darwin":
  if req.Protocol!="smb"{return req,errors.New("macOS NFSv3 mount adapter is not implemented; use SMB")}
  if req.Target!=""{return req,errors.New("Finder selects the SMB mount target; leave this blank")}
 default:
  return req,errors.New("operating system not supported for mounts")
 }
 return req,nil
}
func validateUnixMountPath(target string)error{
 if target==""||!filepath.IsAbs(target)||filepath.Clean(target)!=target||target=="/"||strings.ContainsAny(target,"
"){
  return errors.New("target must be a clean absolute non-root path")
 }
 info,err:=os.Lstat(target)
 if err!=nil{return fmt.Errorf("mount directory must already exist: %w",err)}
 if !info.IsDir()||info.Mode()&os.ModeSymlink!=0{return errors.New("mount directory may not be a symlink")}
 entries,err:=os.ReadDir(target)
 if err!=nil{return err}
 if len(entries)!=0{return errors.New("mount target must be empty")}
 return nil
}
func smbURI(server,share string)string{return "smb://"+server+"/"+url.PathEscape(share)}
func(s *Server)mount(ctx context.Context,req MountRequest)(MountResult,error){
 if _,ok:=s.store.Device(req.DeviceID);!ok{return MountResult{},errors.New("unknown appliance")}
 req,err:=validateRequest(req,s.goos)
 if err!=nil{return MountResult{},err}
 for _,m:=range s.store.Snapshot().Mappings{
  if m.Target!=""&&m.Target==req.Target&&m.Protocol==req.Protocol{
   return MountResult{},errors.New("target already used by another saved mapping")
  }
 }
 ctx,cancel:=context.WithTimeout(ctx,35*time.Second)
 defer cancel()
 var output string
 switch s.goos{
 case "windows":
  if _,err:=os.Stat(req.Target+string(rune(92)));err==nil{return MountResult{},errors.New("Windows drive letter already exists")}
  unc:=string([]rune{92,92})+req.Server+string(rune(92))+req.Remote
  output,err=s.runner.Run(ctx,"net","use",req.Target,unc,"/persistent:no")
 case "linux":
  if req.Protocol=="smb"{
   output,err=s.runner.Run(ctx,"gio","mount",smbURI(req.Server,req.Remote))
  }else{
   output,err=s.runner.Run(ctx,"pkexec","/usr/bin/mount","-t","nfs","-o","vers=3,proto=tcp,nosuid,nodev",req.Server+":"+req.Remote,req.Target)
  }
 case "darwin":
  output,err=s.runner.Run(ctx,"open",smbURI(req.Server,req.Remote))
 }
 if err!=nil{
  if errors.Is(err,exec.ErrNotFound){return MountResult{},errors.New("required system mount utility is not installed")}
  if ctx.Err()!=nil{return MountResult{},ctx.Err()}
  if len(output)>500{output=output[:500]}
  return MountResult{},fmt.Errorf("operating system mapping failed: %s: %w",output,err)
 }
 if s.goos=="darwin"{
  return MountResult{Status:"requested",Message:"Finder received the SMB URL. Confirm the volume is mounted in Finder; mount status is not verified."},nil
 }
 m:=Mapping{
  ID:deviceID(req.DeviceID+"|"+req.Protocol+"|"+req.Server+"|"+req.Remote+"|"+req.Target),
  DeviceID:req.DeviceID,Protocol:req.Protocol,Server:req.Server,Remote:req.Remote,Target:req.Target,AddedAt:time.Now().UTC(),
 }
 if err:=s.store.Update(func(c *Config)error{c.Mappings=append(c.Mappings,m);return nil});err!=nil{
  return MountResult{},fmt.Errorf("OS mapping returned successfully but its metadata could not be saved: %w",err)
 }
 return MountResult{Status:"requested",Message:"The OS accepted the mapping request. Verify the actual mount and access through your file manager; the NAS controls permissions.",Mapping:&m},nil
}
func(s *Server)unmount(ctx context.Context,id string)(MountResult,error){
 var m Mapping
 found:=false
 for _,saved:=range s.store.Snapshot().Mappings{if saved.ID==id{m=saved;found=true;break}}
 if !found{return MountResult{},errors.New("unknown saved mapping")}
 ctx,cancel:=context.WithTimeout(ctx,30*time.Second);defer cancel()
 var output string
 var err error
 switch s.goos{
 case "windows":
  output,err=s.runner.Run(ctx,"net","use",m.Target,"/delete","/y")
 case "linux":
  if m.Protocol=="smb"{output,err=s.runner.Run(ctx,"gio","mount","-u",smbURI(m.Server,m.Remote))
  }else{output,err=s.runner.Run(ctx,"pkexec","/usr/bin/umount",m.Target)}
 default:
  return MountResult{},errors.New("Use Finder to eject this volume; native unmount not implemented")
 }
 if err!=nil{return MountResult{},fmt.Errorf("unmount failed: %.350s: %w",output,err)}
 if err:=s.store.Update(func(c *Config)error{
  for i,x:=range c.Mappings{
   if x.ID==id{c.Mappings=append(c.Mappings[:i],c.Mappings[i+1:]...);return nil}
  }
  return errors.New("mapping no longer exists")
 });err!=nil{return MountResult{},err}
 return MountResult{Status:"requested",Message:"The OS accepted the disconnect; ensure open files have closed."},nil
}
func(s *Server)platform()string{if s.goos!=""{return s.goos};return runtime.GOOS}
