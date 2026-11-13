package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gigabytegrove/zero1local-desktop/internal/appliance"
)

// NAS sessions are ephemeral, per-appliance and private to the local Go
// service. Neither passwords, session cookies nor API keys enter settings.json,
// the browser-facing API, application logs, or the Wails webview.
type nasSession struct {
	origin   string
	username string
	role     string
	since    time.Time
	client   *http.Client
}

type nasLoginRequest struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	TOTP      string `json:"totp"`
	AllowHTTP bool   `json:"allowHTTP"`
}

type nasReadRequest struct {
	ID       string `json:"id"`
	Resource string `json:"resource"`
	Share    string `json:"share"`
	Rel      string `json:"rel"`
}

var nasReadPaths = map[string]string{
	"system":        "/api/system",
	"storage":       "/api/storage",
	"storageHealth": "/api/storage/health",
	"users":         "/api/users",
	"groups":        "/api/groups",
	"shares":        "/api/shares",
	"nfs":           "/api/nfs/exports",
	"files":         "/api/files",
	"tasks":         "/api/tasks",
	"services":      "/api/services",
	"backups":       "/api/backups",
	"notifications": "/api/notifications",
	"activity":      "/api/action-center",
	"protection":    "/api/protection/plans",
}

// A separate cookie jar prevents one NAS session from being sent to another.
// Automatic HTTP redirects and environment-provided proxies are disabled.
func newNASClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.MaxIdleConnsPerHost = 2
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: transport,
		Jar:       jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func (s *Server) nasStatuses() map[string]any {
	s.nasMu.RLock()
	defer s.nasMu.RUnlock()
	out := make(map[string]any, len(s.nasSessions))
	for id, session := range s.nasSessions {
		out[id] = map[string]any{
			"connected": true,
			"username":  session.username,
			"role":      session.role,
			"since":     session.since,
		}
	}
	return out
}

func (s *Server) getNASSession(id string) (*nasSession, bool) {
	if _, ok := s.store.Device(id); !ok {
		return nil, false
	}
	s.nasMu.RLock()
	defer s.nasMu.RUnlock()
	session, ok := s.nasSessions[id]
	return session, ok
}

func (s *Server) dropNASSession(id string) *nasSession {
	s.nasMu.Lock()
	defer s.nasMu.Unlock()
	session := s.nasSessions[id]
	delete(s.nasSessions, id)
	return session
}

func (s *Server) clearNASSessions() {
	s.nasMu.Lock()
	defer s.nasMu.Unlock()
	for id, session := range s.nasSessions {
		if session.client != nil {
			session.client.CloseIdleConnections()
		}
		delete(s.nasSessions, id)
	}
}

func (s *Server) loginNAS(ctx context.Context, origin string, in nasLoginRequest) (*nasSession, error) {
	if strings.TrimSpace(in.Username) == "" || len(in.Username) > 128 ||
		in.Password == "" || len(in.Password) > 4096 || len(in.TOTP) > 32 {
		return nil, errors.New("provide the NAS administrator username and password")
	}
	normalized, err := appliance.NormalizeEndpoint(origin)
	if err != nil {
		return nil, err
	}
	target, err := url.Parse(normalized)
	if err != nil {
		return nil, errors.New("invalid appliance address")
	}
	if target.Scheme == "http" && !in.AllowHTTP {
		return nil, errors.New("this NAS uses unencrypted HTTP; explicitly acknowledge that the administrator password would travel over the network without encryption, or configure HTTPS on the NAS")
	}
	client, err := newNASClient()
	if err != nil {
		return nil, err
	}
	// Zero1Local's actual /api/login contract uses Username, Password and
	// optional TOTP, and requires the X-Zero1Local-Request mutation guard.
	body, err := json.Marshal(map[string]string{
		"Username": strings.TrimSpace(in.Username),
		"Password": in.Password,
		"TOTP":     strings.TrimSpace(in.TOTP),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, normalized+"/api/login", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Zero1Local-Request", "1")
	req.Header.Set("User-Agent", "Zero1Local-Desktop")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("NAS sign-in failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
	if err != nil {
		return nil, err
	}
	var result struct {
		OK           bool   `json:"ok"`
		Username     string `json:"username"`
		Role         string `json:"role"`
		Error        string `json:"error"`
		TOTPRequired bool   `json:"totp_required"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, errors.New("NAS sign-in returned an invalid response")
	}
	if resp.StatusCode != http.StatusOK || !result.OK {
		if result.TOTPRequired {
			return nil, errors.New("the NAS requires a valid two-factor authentication code")
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, errors.New("the NAS temporarily limited sign-in attempts; try again later")
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, errors.New("invalid NAS administrator credentials")
		}
		return nil, fmt.Errorf("NAS sign-in rejected (HTTP %d): %s", resp.StatusCode, limitedNASMessage(result.Error))
	}
	if result.Username == "" {
		return nil, errors.New("NAS sign-in did not identify the authenticated account")
	}
	// The server sets an HttpOnly zero1_session cookie. Verify the cookie is
	// present before advertising this appliance as authenticated.
	cookies := client.Jar.Cookies(target)
	sessionCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "zero1_session" && cookie.Value != "" {
			sessionCookie = true
			break
		}
	}
	if !sessionCookie {
		return nil, errors.New("NAS did not provide an authenticated session cookie")
	}
	return &nasSession{
		origin: normalized, username: result.Username, role: result.Role,
		since: time.Now().UTC(), client: client,
	}, nil
}

func limitedNASMessage(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "the appliance rejected the request"
	}
	if len(s) > 180 {
		return s[:180]
	}
	return s
}

func (s *Server) connectNAS(w http.ResponseWriter, r *http.Request) {
	var in nasLoginRequest
	if err := s.decode(r, &in); err != nil {
		s.bad(w, err)
		return
	}
	device, ok := s.store.Device(in.ID)
	if !ok {
		s.bad(w, errors.New("choose a saved Zero1Local appliance"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	session, err := s.loginNAS(ctx, device.URL, in)
	if err != nil {
		s.bad(w, err)
		return
	}
	s.nasMu.Lock()
	if old := s.nasSessions[device.ID]; old != nil {
		old.client.CloseIdleConnections()
	}
	s.nasSessions[device.ID] = session
	s.nasMu.Unlock()
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "id": device.ID,
		"username": session.username, "role": session.role,
	})
}

func (s *Server) disconnectNAS(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID string `json:"id"` }
	if err := s.decode(r, &in); err != nil {
		s.bad(w, err)
		return
	}
	session := s.dropNASSession(in.ID)
	if session != nil {
		// Best-effort remote sign-out. Local session is discarded even if the
		// NAS is offline. Never print cookies or session details to a log.
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, session.origin+"/api/logout", bytes.NewReader([]byte("{}")))
		if err == nil {
			req.Header.Set("X-Zero1Local-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			if resp, err := session.client.Do(req); err == nil {
				resp.Body.Close()
			}
		}
		cancel()
		session.client.CloseIdleConnections()
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) readNAS(w http.ResponseWriter, r *http.Request) {
	var in nasReadRequest
	if err := s.decode(r, &in); err != nil {
		s.bad(w, err)
		return
	}
	path, ok := nasReadPaths[in.Resource]
	if !ok {
		s.bad(w, errors.New("unsupported NAS resource"))
		return
	}
	session, ok := s.getNASSession(in.ID)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "sign in to this NAS to view its management data"})
		return
	}
	target := session.origin + path
	if in.Resource == "files" {
		query := url.Values{}
		if in.Share != "" {
			if len(in.Share) > 255 {
				s.bad(w, errors.New("invalid share name"))
				return
			}
			query.Set("share", in.Share)
			if in.Rel != "" {
				if len(in.Rel) > 4096 || strings.HasPrefix(in.Rel, "/") || strings.Contains(in.Rel, "\\") {
					s.bad(w, errors.New("invalid relative folder"))
					return
				}
				for _, part := range strings.Split(in.Rel, "/") {
					if part == "" || part == "." || part == ".." {
						s.bad(w, errors.New("invalid relative folder"))
						return
					}
				}
				query.Set("rel", in.Rel)
			}
		} else if in.Rel != "" {
			s.bad(w, errors.New("select a share before opening a folder"))
			return
		}
		if len(query) != 0 {
			target += "?" + query.Encode()
		}
	} else if in.Share != "" || in.Rel != "" {
		s.bad(w, errors.New("unexpected resource arguments"))
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		s.bad(w, err)
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Zero1Local-Desktop")
	resp, err := session.client.Do(req)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Unable to reach the NAS: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if resp.StatusCode == http.StatusUnauthorized {
			s.dropNASSession(in.ID)
		}
		s.writeJSON(w, resp.StatusCode, map[string]string{"error": "the NAS session expired or this administrator lacks permission"})
		return
	}
	if resp.StatusCode != http.StatusOK {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("NAS %s returned HTTP %d", in.Resource, resp.StatusCode),
		})
		return
	}
	// Only pass bounded JSON from an exact, approved endpoint. Never proxy
	// arbitrary URLs, HTML, cross-origin scripts or privileged mutations.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil || len(raw) > (2<<20) || !json.Valid(raw) {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS returned an invalid or oversized response"})
		return
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS returned malformed JSON"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"resource": in.Resource, "data": payload})
}
