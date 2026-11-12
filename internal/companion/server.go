package companion

import(
 "context"
 "github.com/gigabytegrove/zero1local-desktop/internal/appliance"
 "crypto/rand"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "runtime"
 "strings"
 "time"
)

type Server struct {
 store *Store
 runner Runner
 goos string
 token string
 launcher func(string)error
 listener net.Listener
 server *http.Server
 origin string
}
func New(store *Store,launcher func(string)error)(*Server,error){
 if store==nil{return nil,errors.New("settings store is required")}
 b:=make([]byte,32)
 if _,err:=rand.Read(b);err!=nil{return nil,err}
 return &Server{store:store,runner:OSRunner{},goos:runtime.GOOS,token:hex.EncodeToString(b),launcher:launcher},nil
}
func(s *Server)URL()string{return s.origin}
func(s *Server)Start()(string,error){
 if s.listener!=nil{return "",errors.New("companion already listening")}
 l,err:=net.Listen("tcp4","127.0.0.1:0")
 if err!=nil{return "",err}
 s.listener=l
 s.origin="http://"+l.Addr().String()
 s.server=&http.Server{Handler:s.Handler(),ReadHeaderTimeout:5*time.Second,IdleTimeout:35*time.Second,MaxHeaderBytes:16*1024}
 go func(){_ = s.server.Serve(l)}()
 return s.origin,nil
}
func(s *Server)Shutdown(ctx context.Context)error{
 if s.server==nil{return nil}
 return s.server.Shutdown(ctx)
}
func(s *Server)Handler()http.Handler{
 m:=http.NewServeMux()
 m.HandleFunc("GET /",s.handleUI)
 m.HandleFunc("GET /app.css",s.handleCSS)
 m.HandleFunc("GET /app.js",s.handleJS)
 m.HandleFunc("GET /brand-light.png",s.handleBrandLight)
 m.HandleFunc("GET /brand-dark.png",s.handleBrandDark)
 m.HandleFunc("GET /api/state",s.state)
 m.HandleFunc("POST /api/discover",s.discover)
 m.HandleFunc("POST /api/add",s.add)
 m.HandleFunc("POST /api/remove",s.remove)
 m.HandleFunc("POST /api/theme",s.theme)
 m.HandleFunc("POST /api/open",s.open)
 m.HandleFunc("POST /api/mount",s.mountAPI)
 m.HandleFunc("POST /api/unmount",s.unmountAPI)
 m.HandleFunc("POST /api/shares",s.sharesAPI)
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Host=="" || r.Host!=strings.TrimPrefix(s.origin,"http://"){
   http.Error(w,"invalid localhost host",http.StatusForbidden);return
  }
  w.Header().Set("X-Content-Type-Options","nosniff")
  w.Header().Set("X-Frame-Options","DENY")
  w.Header().Set("Referrer-Policy","no-referrer")
  w.Header().Set("Cache-Control","no-store")
  w.Header().Set("Content-Security-Policy","default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
  if r.Method==http.MethodPost{
   if r.Header.Get("Origin")!=s.origin || r.Header.Get("X-Z1L-Token")!=s.token{
    http.Error(w,"request not authorized",http.StatusForbidden);return
   }
   r.Body=http.MaxBytesReader(w,r.Body,32*1024)
  }
  m.ServeHTTP(w,r)
 })
}
func(s *Server)decode(r *http.Request,out any)error{
 dec:=json.NewDecoder(r.Body);dec.DisallowUnknownFields()
 if err:=dec.Decode(out);err!=nil{return err}
 var another any
 if err:=dec.Decode(&another);err!=io.EOF{return errors.New("unexpected trailing JSON")}
 return nil
}
func(s *Server)writeJSON(w http.ResponseWriter,status int,value any){
 w.Header().Set("Content-Type","application/json; charset=utf-8")
 w.WriteHeader(status)
 _=json.NewEncoder(w).Encode(value)
}
func(s *Server)bad(w http.ResponseWriter,err error){
 s.writeJSON(w,400,map[string]any{"error":err.Error()})
}
func(s *Server)state(w http.ResponseWriter,r *http.Request){
 s.writeJSON(w,200,map[string]any{"settings":s.store.Snapshot(),"platform":s.platform()})
}
func(s *Server)discover(w http.ResponseWriter,r *http.Request){
 ctx,cancel:=context.WithTimeout(r.Context(),28*time.Second)
 defer cancel()
 ds:=s.scan(ctx)
 if ds==nil{ds=[]appliance.Device{}}
 s.writeJSON(w,200,map[string]any{"devices":ds})
}
func(s *Server)add(w http.ResponseWriter,r *http.Request){
 var in struct{URL string `json:"url"`}
 if err:=s.decode(r,&in);err!=nil{s.bad(w,err);return}
 d,err:=s.probe(r.Context(),in.URL)
 if err!=nil{s.bad(w,err);return}
 var result InstalledDevice
 err=s.store.Update(func(c *Config)error{v,e:=addDevice(c,d);result=v;return e})
 if err!=nil{s.bad(w,err);return}
 s.writeJSON(w,201,result)
}
func(s *Server)remove(w http.ResponseWriter,r *http.Request){
 var in struct{ID string `json:"id"`}
 if err:=s.decode(r,&in);err!=nil{s.bad(w,err);return}
 err:=s.store.Update(func(c *Config)error{
  for i,v:=range c.Devices{
   if v.ID!=in.ID{continue}
   for _,m:=range c.Mappings{if m.DeviceID==in.ID{return errors.New("disconnect saved shares before forgetting appliance")}}
   c.Devices=append(c.Devices[:i],c.Devices[i+1:]...)
   return nil
  }
  return errors.New("unknown appliance")
 })
 if err!=nil{s.bad(w,err);return}
 s.writeJSON(w,200,map[string]any{"ok":true})
}
func(s *Server)theme(w http.ResponseWriter,r *http.Request){
 var in struct{Theme string `json:"theme"`}
 if err:=s.decode(r,&in);err!=nil{s.bad(w,err);return}
 if in.Theme!="system" && in.Theme!="light" && in.Theme!="dark"{s.bad(w,errors.New("invalid theme"));return}
 if err:=s.store.Update(func(c *Config)error{c.Theme=in.Theme;return nil});err!=nil{s.bad(w,err);return}
 s.writeJSON(w,200,map[string]any{"theme":in.Theme})
}
func(s *Server)open(w http.ResponseWriter,r *http.Request){
 var in struct{ID string `json:"id"`}
 if err:=s.decode(r,&in);err!=nil{s.bad(w,err);return}
 d,exists:=s.store.Device(in.ID)
 if !exists{s.bad(w,errors.New("unknown appliance"));return}
 if s.launcher==nil{s.bad(w,errors.New("native browser launcher unavailable in preview mode"));return}
 if err:=s.launcher(d.URL);err!=nil{s.bad(w,err);return}
 s.writeJSON(w,200,map[string]any{"opened":true})
}
func(s *Server)mountAPI(w http.ResponseWriter,r *http.Request){
 var in MountRequest
 if err:=s.decode(r,&in);err!=nil{s.bad(w,err);return}
 result,err:=s.mount(r.Context(),in)
 if err!=nil{s.bad(w,err);return}
 s.writeJSON(w,200,result)
}
func(s *Server)unmountAPI(w http.ResponseWriter,r *http.Request){
 var in struct{ID string `json:"id"`}
 if err:=s.decode(r,&in);err!=nil{s.bad(w,err);return}
 result,err:=s.unmount(r.Context(),in.ID)
 if err!=nil{s.bad(w,err);return}
 s.writeJSON(w,200,result)
}
func(s *Server)sharesAPI(w http.ResponseWriter,r *http.Request){
 var in struct{
  ID string `json:"id"`
  Token string `json:"token"`
 }
 if err:=s.decode(r,&in);err!=nil{s.bad(w,err);return}
 d,exists:=s.store.Device(in.ID)
 if !exists{s.bad(w,errors.New("unknown appliance"));return}
 if len(in.Token)<8 || len(in.Token)>1024{s.bad(w,errors.New("provide a Zero1Local read-only API key"));return}
 // Never transmit an API key over an unencrypted HTTP appliance origin.
 if !strings.HasPrefix(d.URL,"https://"){s.bad(w,errors.New("authenticated inventory requires an HTTPS appliance origin"));return}
 transport:=http.DefaultTransport.(*http.Transport).Clone()
 transport.Proxy=nil
 client:=&http.Client{Timeout:7*time.Second,Transport:transport,CheckRedirect:func(_ *http.Request,_ []*http.Request)error{return http.ErrUseLastResponse}}
 defer client.CloseIdleConnections()
 output:=map[string]any{}
 for _,path:=range []string{"/api/shares","/api/nfs/exports"}{
  req,err:=http.NewRequestWithContext(r.Context(),http.MethodGet,d.URL+path,nil)
  if err!=nil{s.bad(w,err);return}
  req.Header.Set("Authorization","Bearer "+in.Token)
  resp,err:=client.Do(req)
  if err!=nil{s.bad(w,fmt.Errorf("inventory %s: %w",path,err));return}
  body,err:=io.ReadAll(io.LimitReader(resp.Body,1024*1024))
  resp.Body.Close()
  if err!=nil{s.bad(w,err);return}
  if resp.StatusCode!=200{
   output[path]=map[string]any{"available":false,"status":resp.StatusCode}
   continue
  }
  var payload any
  if err=json.Unmarshal(body,&payload);err!=nil{s.bad(w,errors.New("invalid appliance inventory response"));return}
  output[path]=payload
 }
 s.writeJSON(w,200,output)
}
