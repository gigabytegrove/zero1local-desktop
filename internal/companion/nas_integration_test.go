package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNASSessionsNeverCrossAppliances(t *testing.T) {
	fake := func(secret string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/login":
				http.SetCookie(w, &http.Cookie{Name: "zero1_session", Value: secret, Path: "/", HttpOnly: true})
				_, _ = io.WriteString(w, `{"ok":true,"username":"root","role":"administrator"}`)
			case "/api/system":
				cookie, err := r.Cookie("zero1_session")
				if err != nil || cookie.Value != secret {
					http.Error(w, "wrong device session", http.StatusUnauthorized)
					return
				}
				_, _ = io.WriteString(w, `{"hostname":"`+secret+`"}`)
			default:
				http.NotFound(w, r)
			}
		}))
	}
	a, b := fake("nas-alpha"), fake("nas-bravo")
	defer a.Close()
	defer b.Close()

	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.origin = "http://127.0.0.1:8124"
	for _, pair := range []struct {
		ID, URL string
	}{
		{"alpha", a.URL}, {"bravo", b.URL},
	} {
		id, origin := pair.ID, pair.URL
		if err := svc.store.Update(func(c *Config) error {
			c.Devices = append(c.Devices, InstalledDevice{ID: id, URL: origin, Name: id})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		sess, err := svc.loginNAS(context.Background(), origin, nasLoginRequest{
			Username: "root", Password: "secret", AllowHTTP: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		svc.nasMu.Lock()
		svc.nasSessions[id] = sess
		svc.nasMu.Unlock()
	}
	for _, pair := range []struct{ ID, Expected string }{
		{"alpha", "nas-alpha"}, {"bravo", "nas-bravo"},
	} {
		rec := nasTestRequest(t, svc, "POST", "/api/nas/read", map[string]string{
			"id": pair.ID, "resource": "system",
		})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), pair.Expected) {
			t.Fatalf("per-NAS session isolation failed for %s: %d %s", pair.ID, rec.Code, rec.Body.String())
		}
	}
}

func TestNASUploadWithLocalCSRFAndNASMutationGuard(t *testing.T) {
	const name = "sample.txt"
	payload := []byte("uploaded through Zero1Local Desktop")
	nas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/login":
			http.SetCookie(w, &http.Cookie{Name: "zero1_session", Value: "verified", Path: "/", HttpOnly: true})
			_, _ = io.WriteString(w, `{"ok":true,"username":"root","role":"administrator"}`)
		case "/api/files/upload":
			if r.Header.Get("X-Zero1Local-Request") != "1" || r.URL.Query().Get("path") != "/disk0/Docs" {
				http.Error(w, `{"error":"invalid upload"}`, http.StatusForbidden)
				return
			}
			if c, err := r.Cookie("zero1_session"); err != nil || c.Value != "verified" {
				http.Error(w, `{"error":"missing session"}`, http.StatusUnauthorized)
				return
			}
			reader, err := r.MultipartReader()
			if err != nil {
				http.Error(w, "bad upload", 400)
				return
			}
			part, err := reader.NextPart()
			if err != nil || part.FileName() != name {
				http.Error(w, "bad file", 400)
				return
			}
			raw, err := io.ReadAll(part)
			if err != nil || !bytes.Equal(raw, payload) {
				http.Error(w, "wrong file data", 400)
				return
			}
			_, _ = io.WriteString(w, `{"ok":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer nas.Close()
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.origin = "http://127.0.0.1:8124"
	if err := svc.store.Update(func(c *Config) error {
		c.Devices = append(c.Devices, InstalledDevice{ID: "nas", URL: nas.URL, Name: "NAS"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sess, err := svc.loginNAS(context.Background(), nas.URL, nasLoginRequest{
		Username: "root", Password: "secret", AllowHTTP: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.nasMu.Lock()
	svc.nasSessions["nas"] = sess
	svc.nasMu.Unlock()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	const requestPath = "/api/nas/upload?id=nas&path=%2Fdisk0%2FDocs"
	makeRequest := func(origin, csrf string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, svc.origin+requestPath, bytes.NewReader(body.Bytes()))
		req.Header.Set("Origin", origin)
		req.Header.Set("X-Z1L-Token", csrf)
		req.Header.Set("Content-Type", form.FormDataContentType())
		rec := httptest.NewRecorder()
		svc.Handler().ServeHTTP(rec, req)
		return rec
	}
	if status := makeRequest("https://evil.invalid", svc.token).Code; status != 403 {
		t.Fatalf("cross-site file upload accepted: %d", status)
	}
	if status := makeRequest(svc.origin, "invalid-token").Code; status != 403 {
		t.Fatalf("invalid-CSRF upload accepted: %d", status)
	}
	if result := makeRequest(svc.origin, svc.token); result.Code != http.StatusOK {
		var response any
		_ = json.Unmarshal(result.Body.Bytes(), &response)
		t.Fatalf("authorized upload failed: %d %v", result.Code, response)
	}
}
