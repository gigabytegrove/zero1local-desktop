package companion

import (
 "bytes"
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func tempStore(t *testing.T)*Store{
 t.Helper()
 s,err:=newStore(filepath.Join(t.TempDir(),"settings.json"))
 if err!=nil{t.Fatal(err)}
 return s
}
func TestDesktopServiceSecurity(t *testing.T){
 svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
 address,err:=svc.Start();if err!=nil{t.Fatal(err)}
 defer func(){ctx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel();_=svc.Shutdown(ctx)}()
 client:=&http.Client{Timeout:2*time.Second}
 resp,err:=client.Get(address+"/");if err!=nil{t.Fatal(err)}
 if resp.StatusCode!=200{t.Fatalf("UI HTTP status %d",resp.StatusCode)}
 if resp.Header.Get("Content-Security-Policy")==""{t.Fatal("missing browser CSP")}
 resp.Body.Close()
 invalid,err:=http.NewRequest("POST",address+"/api/theme",bytes.NewBufferString(`{"theme":"dark"}`))
 if err!=nil{t.Fatal(err)}
 invalid.Header.Set("Origin","http://attacker.invalid")
 resp,err=client.Do(invalid);if err!=nil{t.Fatal(err)}
 if resp.StatusCode!=403{t.Fatalf("cross-origin mutation accepted: %d",resp.StatusCode)}
 resp.Body.Close()
 valid,err:=http.NewRequest("POST",address+"/api/theme",bytes.NewBufferString(`{"theme":"dark"}`))
 if err!=nil{t.Fatal(err)}
 valid.Header.Set("Origin",address);valid.Header.Set("X-Z1L-Token",svc.token)
 resp,err=client.Do(valid);if err!=nil{t.Fatal(err)}
 if resp.StatusCode!=200{t.Fatalf("theme mutation rejected: %d",resp.StatusCode)}
 resp.Body.Close()
 if svc.store.Snapshot().Theme!="dark"{t.Fatal("theme did not persist")}
}
func TestStorePermissions(t *testing.T){
 path:=filepath.Join(t.TempDir(),"settings.json")
 store,err:=newStore(path);if err!=nil{t.Fatal(err)}
 err=store.Update(func(c *Config)error{c.Theme="light";return nil})
 if err!=nil{t.Fatal(err)}
 info,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
 if info.Mode().Perm()!=0600{t.Fatalf("unexpected mode %o",info.Mode().Perm())}
 b,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
 if !json.Valid(b)||strings.Contains(string(b),"token"){t.Fatal("unexpected settings content")}
 reloaded,err:=newStore(path);if err!=nil{t.Fatal(err)}
 if reloaded.Snapshot().Theme!="light"{t.Fatal("theme lost after restart")}
}
func TestHostHeader(t *testing.T){
 svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
 svc.origin="http://127.0.0.1:8124"
 r:=httptest.NewRequest("GET","http://untrusted.invalid/api/state",nil)
 w:=httptest.NewRecorder();svc.Handler().ServeHTTP(w,r)
 if w.Code!=403{t.Fatalf("wrong Host accepted %d",w.Code)}
}
func TestRejectUnsafeMounts(t *testing.T){
 for _,req:=range []MountRequest{
  {Protocol:"smb",Server:"nas;rm",Remote:"Documents",Target:"Z:"},
  {Protocol:"smb",Server:"nas",Remote:"..",Target:"Z:"},
  {Protocol:"smb",Server:"nas",Remote:"Documents",Target:"C:"},
  {Protocol:"nfs",Server:"nas",Remote:"/disk0/../etc",Target:"Z:"},
 }{
  if _,err:=validateRequest(req,"windows");err==nil{t.Fatalf("unsafe mapping accepted: %+v",req)}
 }
}
type runnerFake struct{calls int}
func(f *runnerFake)Run(_ context.Context,name string,_ ...string)(string,error){f.calls++;return "",nil}
func TestMountRequiresSavedAppliance(t *testing.T){
 svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
 f:=&runnerFake{};svc.runner=f;svc.goos="linux"
 _,err=svc.mount(context.Background(),MountRequest{DeviceID:"missing",Protocol:"smb",Server:"nas",Remote:"Documents"})
 if err==nil||f.calls!=0{t.Fatal("unknown appliance dispatched OS command")}
}
