package companion

import(
 "embed"
 "fmt"
 "net/http"
 "strings"
)
//go:embed ui/*
var assets embed.FS
func(s *Server)handleUI(w http.ResponseWriter,r *http.Request){
 if r.URL.Path!="/"{http.NotFound(w,r);return}
 b,err:=assets.ReadFile("ui/index.html")
 if err!=nil{http.Error(w,"desktop UI missing",500);return}
 w.Header().Set("Content-Type","text/html; charset=utf-8")
 _,_=w.Write([]byte(strings.ReplaceAll(string(b),"__LOCAL_TOKEN__",s.token)))
}
func serveAsset(w http.ResponseWriter,path,kind string){
 b,err:=assets.ReadFile(path)
 if err!=nil{http.Error(w,fmt.Sprintf("missing asset: %s",path),500);return}
 w.Header().Set("Content-Type",kind)
 _,_=w.Write(b)
}
func(s *Server)handleCSS(w http.ResponseWriter,r *http.Request){serveAsset(w,"ui/app.css","text/css; charset=utf-8")}
func(s *Server)handleJS(w http.ResponseWriter,r *http.Request){serveAsset(w,"ui/app.js","text/javascript; charset=utf-8")}
func(s *Server)handleBrandLight(w http.ResponseWriter,r *http.Request){serveAsset(w,"ui/brand-light.png","image/png")}
func(s *Server)handleBrandDark(w http.ResponseWriter,r *http.Request){serveAsset(w,"ui/brand-dark.png","image/png")}
