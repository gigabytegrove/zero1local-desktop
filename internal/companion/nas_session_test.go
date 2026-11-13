package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func nasTestRequest(t *testing.T, svc *Server, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, "http://"+strings.TrimPrefix(svc.origin, "http://")+path, bytes.NewReader(body))
	req.Header.Set("Origin", svc.origin)
	req.Header.Set("X-Z1L-Token", svc.token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, req)
	return rec
}

func TestNASAuthenticatedOnboardingAndScopedRead(t *testing.T) {
	var loginCount, logoutCount atomic.Int32
	nas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"service":"zero1-local","version":"1.4.6","setup":true}`))
		case "/api/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"device_name":"Test NAS"}`))
		case "/api/login":
			loginCount.Add(1)
			if r.Header.Get("X-Zero1Local-Request") != "1" {
				http.Error(w, "bad mutation header", 403)
				return
			}
			var credentials struct{ Username, Password, TOTP string }
			if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if credentials.Username != "root" || credentials.Password != "correct" || credentials.TOTP != "123456" {
				w.WriteHeader(401)
				_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "zero1_session", Value: "secret", Path: "/", HttpOnly: true})
			_, _ = w.Write([]byte(`{"ok":true,"username":"root","role":"administrator"}`))
		case "/api/system":
			c, err := r.Cookie("zero1_session")
			if err != nil || c.Value != "secret" {
				http.Error(w, "unauthorized", 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"hostname":"zero1-test","kernel":"test-kernel"}`))
		case "/api/logout":
			logoutCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer nas.Close()
	store := tempStore(t)
	svc, err := New(store, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.origin = "http://127.0.0.1:8124"
	creds := map[string]any{"url": nas.URL, "username": "root", "password": "correct", "totp": "123456"}
	// No password may cross an unencrypted NAS connection without explicit consent.
	refused := nasTestRequest(t, svc, "POST", "/api/add", creds)
	if refused.Code != 400 || len(store.Snapshot().Devices) != 0 || loginCount.Load() != 0 {
		t.Fatalf("HTTP credentials were not blocked: %d %s", refused.Code, refused.Body.String())
	}
	creds["allowHTTP"] = true
	creds["password"] = "wrong"
	wrong := nasTestRequest(t, svc, "POST", "/api/add", creds)
	if wrong.Code != 400 || len(store.Snapshot().Devices) != 0 {
		t.Fatal("invalid credentials saved a device")
	}
	creds["password"] = "correct"
	ok := nasTestRequest(t, svc, "POST", "/api/add", creds)
	if ok.Code != 201 {
		t.Fatalf("valid login failed: %d %s", ok.Code, ok.Body.String())
	}
	id := store.Snapshot().Devices[0].ID
	raw, _ := json.Marshal(store.Snapshot())
	if bytes.Contains(raw, []byte("correct")) || bytes.Contains(raw, []byte("secret")) {
		t.Fatal("credential material leaked into settings")
	}
	read := nasTestRequest(t, svc, "POST", "/api/nas/read", map[string]any{"id": id, "resource": "system"})
	if read.Code != 200 || !strings.Contains(read.Body.String(), "zero1-test") {
		t.Fatalf("NAS data not shown: %d %s", read.Code, read.Body.String())
	}
	badPath := nasTestRequest(t, svc, "POST", "/api/nas/read", map[string]any{"id": id, "resource": "files", "share": "Docs", "rel": "../../etc"})
	if badPath.Code != 400 {
		t.Fatalf("path traversal accepted: %d", badPath.Code)
	}
	badResource := nasTestRequest(t, svc, "POST", "/api/nas/read", map[string]any{"id": id, "resource": "apiKey"})
	if badResource.Code != 400 {
		t.Fatalf("unexpected resource accepted: %d", badResource.Code)
	}
	logout := nasTestRequest(t, svc, "POST", "/api/nas/logout", map[string]string{"id": id})
	if logout.Code != 200 || logoutCount.Load() != 1 {
		t.Fatal("NAS logout failed")
	}
	disconnected := nasTestRequest(t, svc, "POST", "/api/nas/read", map[string]any{"id": id, "resource": "system"})
	if disconnected.Code != 401 {
		t.Fatal("logged-out session still authorized")
	}
	// Reconnect an already-saved NAS without saving the password.
	login := nasTestRequest(t, svc, "POST", "/api/nas/login", map[string]any{"id": id, "username": "root", "password": "correct", "totp": "123456", "allowHTTP": true})
	if login.Code != 200 {
		t.Fatalf("reconnect failed: %d %s", login.Code, login.Body.String())
	}
	forget := nasTestRequest(t, svc, "POST", "/api/remove", map[string]string{"id": id})
	if forget.Code != 200 {
		t.Fatalf("forget failed: %s", forget.Body.String())
	}
	if len(svc.nasStatuses()) != 0 {
		t.Fatal("forgotten NAS still has a session")
	}
}

func TestNASLoginRejectsRedirectAndMissingCookie(t *testing.T) {
	nas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/login" {
			http.Redirect(w, r, "http://evil.invalid/capture", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer nas.Close()
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = svc.loginNAS(ctx, nas.URL, nasLoginRequest{Username: "root", Password: "correct", AllowHTTP: true})
	if err == nil {
		t.Fatal("redirecting login was accepted")
	}
}

func TestNASSameOriginGuardOnCredentialLogin(t *testing.T) {
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.origin = "http://127.0.0.1:8124"
	req := httptest.NewRequest("POST", svc.origin+"/api/add", strings.NewReader(`{"url":"http://127.0.0.1","username":"root","password":"p"}`))
	req.Header.Set("Origin", "https://evil.invalid")
	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("cross-site credential operation accepted: %d", rec.Code)
	}
}
