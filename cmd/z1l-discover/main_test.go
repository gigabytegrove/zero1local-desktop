package main

import (
    "bytes"
    "context"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestRunProbe(t *testing.T) {
    s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if r.URL.Path == "/healthz" { _, _ = w.Write([]byte(`{"ok":true,"service":"zero1-local","version":"test","setup":true}`)); return }
        if r.URL.Path == "/api/status" { _, _ = w.Write([]byte(`{"device_name":"Test NAS"}`)); return }
        http.NotFound(w,r)
    }))
    defer s.Close()
    var out bytes.Buffer
    if err:=run(context.Background(),[]string{"-probe",s.URL},&out);err!=nil{t.Fatal(err)}
    if !strings.Contains(out.String(),`"Test NAS"`){t.Fatal(out.String())}
}

func TestRunBadFlags(t *testing.T) {
    for _, args:=range [][]string{{}, {"-scan","-probe","http://nas"}, {"-probe", "bad", "extra"}, {"-scan", "-timeout", "1ms"}} {
        if err:=run(context.Background(),args,&bytes.Buffer{});err==nil{t.Fatalf("accepted %v",args)}
    }
}
