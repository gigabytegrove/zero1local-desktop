package appliance

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"
)

func TestNormalizeEndpoint(t *testing.T) {
    for _, tc := range []struct{ input, want string; valid bool }{
        {"192.168.1.2", "http://192.168.1.2", true},
        {"https://nas.example:8443/", "https://nas.example:8443", true},
        {"http://user:secret@nas", "", false},
        {"ftp://nas", "", false},
        {"http://nas/api/login", "", false},
        {"http://nas?token=bad", "", false},
        {"http://nas:99999", "", false},
    } {
        got, err := NormalizeEndpoint(tc.input)
        if (err == nil) != tc.valid || (tc.valid && got != tc.want) { t.Errorf("NormalizeEndpoint(%q) = %q, %v", tc.input, got, err) }
    }
}

func TestProbePositiveIdentity(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        switch r.URL.Path {
        case "/healthz": fmt.Fprint(w, `{"ok":true,"service":"zero1-local","version":"1.4.3","setup":true}`)
        case "/api/status": fmt.Fprint(w, `{"device_name":"Office NAS"}`)
        default: http.NotFound(w, r)
        }
    }))
    defer srv.Close()
    got, err := Probe(context.Background(), srv.URL, time.Second)
    if err != nil || got.Name != "Office NAS" || got.Version != "1.4.3" || !got.Setup { t.Fatalf("probe = %+v: %v", got, err) }
}

func TestProbeRejectsImpostorAndRedirect(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.RawQuery, "redirect") { http.Redirect(w, r, "https://example.org", http.StatusFound); return }
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, `{"ok":true,"service":"not-zero1-local"}`)
    }))
    defer srv.Close()
    if _, err := Probe(context.Background(), srv.URL, time.Second); err == nil { t.Fatal("impostor accepted") }
    if !privateIPv4(net.ParseIP("172.20.1.5")) || privateIPv4(net.ParseIP("172.32.1.5")) { t.Fatal("incorrect RFC1918 classification") }
}

func TestDiscoverContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    started := time.Now()
    _ = Discover(ctx, []string{"198.18.0.1", "198.18.0.2"}, 250*time.Millisecond)
    if time.Since(started) > 500*time.Millisecond { t.Fatal("cancellation was not bounded") }
}


func TestProbeDoesNotFollowRedirect(t *testing.T) {
    destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        t.Error("redirect destination should not have been contacted")
    }))
    defer destination.Close()
    source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, destination.URL, http.StatusFound)
    }))
    defer source.Close()
    if _, err := Probe(context.Background(), source.URL, time.Second); err == nil { t.Fatal("redirected endpoint accepted") }
}

func TestProbeRejectsHTML(t *testing.T) {
    source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        _,_ = w.Write([]byte(`{"ok":true,"service":"zero1-local"}`))
    }))
    defer source.Close()
    if _, err := Probe(context.Background(), source.URL, time.Second); err == nil { t.Fatal("HTML endpoint accepted") }
}
