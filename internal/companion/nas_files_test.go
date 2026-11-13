package companion

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"context"
)

func TestNASFilesActionsAndDownload(t *testing.T) {
	var remoteAction string
	nas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		switch r.URL.Path {
		case "/api/login":
			http.SetCookie(w,&http.Cookie{Name:"zero1_session",Value:"verified",Path:"/",HttpOnly:true})
			w.Header().Set("Content-Type","application/json")
			_,_=io.WriteString(w,`{"ok":true,"username":"root","role":"administrator"}`)
		case "/api/files/mkdir","/api/files/rename","/api/files/trash":
			if r.Header.Get("X-Zero1Local-Request")!="1"{http.Error(w,"missing request guard",403);return}
			var b map[string]string
			if err:=json.NewDecoder(r.Body).Decode(&b);err!=nil{http.Error(w,"bad JSON",400);return}
			if b["Path"]!="/disk0/Docs"{http.Error(w,"wrong path",400);return}
			remoteAction=r.URL.Path
			w.Header().Set("Content-Type","application/json")
			_,_=io.WriteString(w,`{"ok":true}`)
		case "/api/files/download":
			if r.URL.Query().Get("path")!="/disk0/Docs/readme.txt"{http.Error(w,"bad path",400);return}
			http.ServeContent(w,r,"readme.txt",time.Now(),bytes.NewReader([]byte("hello from NAS")))
		default:http.NotFound(w,r)
		}
	}))
	defer nas.Close()
	svc,err:=New(tempStore(t),nil);if err!=nil{t.Fatal(err)}
	svc.origin="http://127.0.0.1:8124"
	id:="abc"
	svc.store.Update(func(c *Config)error{c.Devices=append(c.Devices,InstalledDevice{ID:id,URL:nas.URL,Name:"NAS"});return nil})
	session,err:=svc.loginNAS(context.Background(),nas.URL,nasLoginRequest{Username:"root",Password:"pass",AllowHTTP:true})
	if err!=nil{t.Fatal(err)}
	svc.nasMu.Lock();svc.nasSessions[id]=session;svc.nasMu.Unlock()
	action:=nasTestRequest(t,svc,"POST","/api/nas/file-action",map[string]string{"id":id,"action":"mkdir","path":"/disk0/Docs","name":"NewFolder"})
	if action.Code!=200||remoteAction!="/api/files/mkdir"{t.Fatalf("mkdir failed: %d %s",action.Code,action.Body.String())}
	bad:=nasTestRequest(t,svc,"POST","/api/nas/file-action",map[string]string{"id":id,"action":"shell","path":"/disk0/Docs"})
	if bad.Code!=400{t.Fatal("arbitrary file operation accepted")}
	rename:=nasTestRequest(t,svc,"POST","/api/nas/file-action",map[string]string{"id":id,"action":"rename","path":"/disk0/Docs","name":"Renamed"})
	if rename.Code!=200||remoteAction!="/api/files/rename"{t.Fatal("rename failed")}
	trash:=nasTestRequest(t,svc,"POST","/api/nas/file-action",map[string]string{"id":id,"action":"trash","path":"/disk0/Docs"})
	if trash.Code!=200||remoteAction!="/api/files/trash"{t.Fatal("recycle bin operation failed")}
	dl:=nasTestRequest(t,svc,"POST","/api/nas/download",map[string]string{"id":id,"path":"/disk0/Docs/readme.txt"})
	if dl.Code!=200||dl.Body.String()!="hello from NAS"{t.Fatalf("download failed: %d %s",dl.Code,dl.Body.String())}
}
