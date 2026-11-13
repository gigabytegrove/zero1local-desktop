package companion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryCredentialVault struct {
	mu     sync.Mutex
	values map[string][]byte
}

func (v *memoryCredentialVault) Put(id string, secret []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.values == nil {
		v.values = make(map[string][]byte)
	}
	v.values[id] = slices.Clone(secret)
	return nil
}
func (v *memoryCredentialVault) Get(id string) ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	value, ok := v.values[id]
	if !ok {
		return nil, errors.New("credential missing")
	}
	return slices.Clone(value), nil
}
func (v *memoryCredentialVault) Delete(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.values, id)
	return nil
}

func TestRememberedNASRestoresRenewsAndSignsOut(t *testing.T) {
	var logins, activeToken atomic.Int32
	nas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/login":
			var in struct{ Username, Password, TOTP string }
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				http.Error(w, "bad request", 400)
				return
			}
			if in.Username != "root" || in.Password != "correct" {
				http.Error(w, "unauthorized", 401)
				return
			}
			token := strconv.Itoa(int(logins.Add(1)))
			activeToken.Store(logins.Load())
			http.SetCookie(w, &http.Cookie{Name: "zero1_session", Value: token, Path: "/", HttpOnly: true})
			_, _ = w.Write([]byte(`{"ok":true,"username":"root","role":"administrator"}`))
		case "/api/system":
			cookie, err := r.Cookie("zero1_session")
			if err != nil || cookie.Value != strconv.Itoa(int(activeToken.Load())) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"session expired"}`))
				return
			}
			_, _ = w.Write([]byte(`{"hostname":"saved-nas"}`))
		case "/api/logout":
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer nas.Close()

	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	vault := &memoryCredentialVault{}
	svc.ConfigureCredentialVault(vault)
	svc.origin = "http://127.0.0.1:8124"
	id := deviceID(nas.URL)
	if err := svc.store.Update(func(c *Config) error {
		c.Devices = append(c.Devices, InstalledDevice{ID: id, URL: nas.URL, Name: "Saved NAS"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result := nasTestRequest(t, svc, "POST", "/api/nas/login", map[string]any{
		"id": id, "username": "root", "password": "correct", "allowHTTP": true, "remember": true,
	})
	if result.Code != http.StatusOK {
		t.Fatalf("saved login failed: HTTP %d %s", result.Code, result.Body.String())
	}
	if !svc.store.Snapshot().Devices[0].RememberSignIn {
		t.Fatal("NAS remember choice not persisted")
	}
	raw, err := os.ReadFile(svc.store.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "correct") || strings.Contains(string(raw), "zero1_session") {
		t.Fatal("sensitive NAS credentials persisted in desktop JSON")
	}
	data, err := vault.Get(id)
	if err != nil {
		t.Fatal("credential missing from OS vault substitute")
	}
	if strings.Contains(string(data), "totp") {
		t.Fatal("MFA code must never be stored for automatic login")
	}

	svc.clearNASSessions()
	svc.restoreRememberedSessions()
	if _, exists := svc.getNASSession(id); !exists || logins.Load() != 2 {
		t.Fatalf("saved NAS session was not restored, total logins=%d", logins.Load())
	}
	activeToken.Add(1) // NAS reboot or session invalidation.
	read := nasTestRequest(t, svc, "POST", "/api/nas/read", map[string]string{"id": id, "resource": "system"})
	if read.Code != http.StatusOK || !strings.Contains(read.Body.String(), "saved-nas") || logins.Load() != 3 {
		t.Fatalf("NAS expiry was not recovered automatically: HTTP %d %s; logins=%d",
			read.Code, read.Body.String(), logins.Load())
	}
	logout := nasTestRequest(t, svc, "POST", "/api/nas/logout", map[string]string{"id": id})
	if logout.Code != http.StatusOK {
		t.Fatalf("logout failed: %s", logout.Body.String())
	}
	if svc.store.Snapshot().Devices[0].RememberSignIn {
		t.Fatal("sign out left persistent sign-in enabled")
	}
	if _, err := vault.Get(id); err == nil {
		t.Fatal("sign out failed to remove saved credentials from vault")
	}
	if _, exists := svc.getNASSession(id); exists {
		t.Fatal("sign out left a live NAS session")
	}
}

func TestRememberingRequiresNativeCredentialVault(t *testing.T) {
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.saveNASCredential("nas", "http://127.0.0.1", nasLoginRequest{
		Username: "root", Password: "pass", AllowHTTP: true,
	}); err == nil {
		t.Fatal("remembered sign-in accepted without a native credential vault")
	}
}

func TestNASAutomaticLoginDoesNotBypassMFA(t *testing.T) {
	var requireTOTP atomic.Bool
	nas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/login" {
			http.NotFound(w, r)
			return
		}
		var in struct{ Username, Password, TOTP string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		w.Header().Set("Content-Type", "application/json")
		if requireTOTP.Load() && in.TOTP != "246810" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"totp_required":true,"error":"two-factor authentication code required"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "zero1_session", Value: "verified", Path: "/", HttpOnly: true})
		_, _ = w.Write([]byte(`{"ok":true,"username":"root","role":"administrator"}`))
	}))
	defer nas.Close()
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	vault := &memoryCredentialVault{}
	svc.ConfigureCredentialVault(vault)
	id := deviceID(nas.URL)
	if err := svc.store.Update(func(c *Config) error {
		c.Devices = append(c.Devices, InstalledDevice{ID: id, Name: "NAS", URL: nas.URL, RememberSignIn: true})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.saveNASCredential(id, nas.URL, nasLoginRequest{
		Username: "root", Password: "pass", TOTP: "246810", AllowHTTP: true,
	}); err != nil {
		t.Fatal(err)
	}
	requireTOTP.Store(true)
	svc.restoreRememberedSessions()
	if _, ok := svc.getNASSession(id); ok {
		t.Fatal("saved sign-in bypassed NAS two-factor authentication")
	}
	if !strings.Contains(svc.nasRecoveryStatuses()[id], "verification code") {
		t.Fatal("MFA failure should display a sign-in prompt")
	}
}

type persistenceRunner struct {
	calls [][]string
}
func (r *persistenceRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return "", nil
}

func TestWindowsShareReconnectIsPerMapping(t *testing.T) {
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.goos = "windows"
	id := deviceID("http://127.0.0.1")
	if err := svc.store.Update(func(c *Config) error {
		c.Devices = append(c.Devices, InstalledDevice{ID: id, URL: "http://127.0.0.1", Name: "NAS"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	runner := &persistenceRunner{}
	svc.runner = runner
	connected, err := svc.mount(context.Background(), MountRequest{
		DeviceID: id, Protocol: "smb", Server: "nas", Remote: "Docs", Target: "X:", Reconnect: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if connected.Mapping == nil || !connected.Mapping.Reconnect {
		t.Fatal("share reconnect flag not saved")
	}
	if len(runner.calls) != 1 || !slices.Contains(runner.calls[0], "/persistent:yes") {
		t.Fatalf("Windows persistent drive flag missing: %v", runner.calls)
	}
	if _, err := validateRequest(MountRequest{
		Protocol: "smb", Server: "nas", Remote: "Docs", Reconnect: true,
	}, "linux"); err == nil {
		t.Fatal("unsupported Linux auto-reconnect was accepted")
	}
}

func TestNASRememberedSessionRenewalThrottledAfterFailure(t *testing.T) {
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	vault := &memoryCredentialVault{}
	svc.ConfigureCredentialVault(vault)
	id := deviceID("http://127.0.0.1")
	if err := svc.store.Update(func(c *Config) error {
		c.Devices = append(c.Devices, InstalledDevice{ID: id, URL: "http://127.0.0.1", RememberSignIn: true})
		return nil
	}); err != nil { t.Fatal(err) }
	if err := svc.saveNASCredential(id, "http://127.0.0.1", nasLoginRequest{
		Username: "root", Password: "password", AllowHTTP: true,
	}); err != nil { t.Fatal(err) }
	svc.nasMu.Lock()
	svc.retryAfter[id] = time.Now().Add(time.Minute)
	svc.nasMu.Unlock()
	if _, err := svc.renewRememberedNAS(context.Background(), id, nil); err == nil ||
		!strings.Contains(err.Error(), "try again later") {
		t.Fatalf("renewal cooldown not enforced: %v", err)
	}
}
