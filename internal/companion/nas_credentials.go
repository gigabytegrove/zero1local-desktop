package companion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// CredentialVault is provided by the native desktop runtime. No passwords,
// NAS cookies, TOTP codes or operating-system vault contents are stored in the
// JSON preferences file or returned to the frontend.
type CredentialVault interface {
	Put(id string, secret []byte) error
	Get(id string) ([]byte, error)
	Delete(id string) error
}

type savedNASCredential struct {
	URL              string    `json:"url"`
	Username         string    `json:"username"`
	Password         string    `json:"password"`
	AllowHTTP        bool      `json:"allowHTTP"`
	SessionToken     string    `json:"sessionToken,omitempty"`
	SessionExpiresAt time.Time `json:"sessionExpiresAt,omitempty"`
	Role             string    `json:"role,omitempty"`
}

// Only renewal operations for the same NAS are serialized. One offline NAS
// cannot delay a completely independent NAS workspace.
func (s *Server) renewalLock(id string) *sync.Mutex {
	s.reauthMu.Lock()
	defer s.reauthMu.Unlock()
	if s.renewLocks == nil {
		s.renewLocks = map[string]*sync.Mutex{}
	}
	if lock := s.renewLocks[id]; lock != nil {
		return lock
	}
	lock := &sync.Mutex{}
	s.renewLocks[id] = lock
	return lock
}

func (s *Server) ConfigureCredentialVault(v CredentialVault) {
	// Configure once before Start. Preview mode intentionally has no vault.
	s.vault = v
}

func (s *Server) saveNASCredential(id, address string, in nasLoginRequest, sessions ...*nasSession) error {
	if s.vault == nil {
		return errors.New("system credential storage is unavailable; choose session-only sign-in")
	}
	if id != deviceID(address) || in.Username == "" || in.Password == "" {
		return errors.New("invalid appliance credential identity")
	}
	record := savedNASCredential{URL: address, Username: strings.TrimSpace(in.Username),
		Password: in.Password, AllowHTTP: in.AllowHTTP}
	if len(sessions) > 0 && sessions[0] != nil {
		captureNASSessionCookie(&record, sessions[0])
	}
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.vault.Put(id, body)
}

func captureNASSessionCookie(record *savedNASCredential, session *nasSession) {
	if record == nil || session == nil || session.client == nil || session.client.Jar == nil {
		return
	}
	u, err := url.Parse(session.origin)
	if err != nil {
		return
	}
	for _, cookie := range session.client.Jar.Cookies(u) {
		if cookie.Name == "zero1_session" && cookie.Value != "" {
			record.SessionToken = cookie.Value
			record.SessionExpiresAt = session.expiresAt
			if record.SessionExpiresAt.IsZero() {
				record.SessionExpiresAt = session.since.Add(8 * time.Hour)
			}
			record.Role = session.role
			return
		}
	}
}

func resumeNASSavedCookie(ctx context.Context, address string, record savedNASCredential) (*nasSession, error) {
	if record.SessionToken == "" || record.SessionExpiresAt.IsZero() ||
		!time.Now().Before(record.SessionExpiresAt.Add(-time.Minute)) {
		return nil, errors.New("saved NAS session has expired")
	}
	target, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	client, err := newNASClient()
	if err != nil {
		return nil, err
	}
	client.Jar.SetCookies(target, []*http.Cookie{{Name: "zero1_session", Value: record.SessionToken,
		Path: "/", HttpOnly: true, Secure: target.Scheme == "https", Expires: record.SessionExpiresAt}})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address+"/api/session", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Zero1Local-Desktop")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NAS session cannot be resumed (HTTP %d)", resp.StatusCode)
	}
	return &nasSession{origin: address, username: record.Username, role: record.Role,
		since: record.SessionExpiresAt.Add(-8 * time.Hour), expiresAt: record.SessionExpiresAt, client: client}, nil
}

func (s *Server) removeNASCredential(id string) error {
	if s.vault == nil {
		return errors.New("the system credential vault is unavailable; saved sign-in cannot be removed")
	}
	return s.vault.Delete(id)
}

func (s *Server) setNASRecoveryStatus(id, state string) {
	s.nasMu.Lock()
	defer s.nasMu.Unlock()
	if state == "" {
		delete(s.restoreStatus, id)
	} else {
		s.restoreStatus[id] = state
	}
}

func (s *Server) nasRecoveryStatuses() map[string]string {
	s.nasMu.RLock()
	defer s.nasMu.RUnlock()
	out := make(map[string]string, len(s.restoreStatus))
	for id, state := range s.restoreStatus {
		out[id] = state
	}
	return out
}

func (s *Server) rememberExistingNAS(id, address string, in nasLoginRequest, session *nasSession) error {
	lock := s.renewalLock(id)
	lock.Lock()
	defer lock.Unlock()
	device, ok := s.store.Device(id)
	if !ok || device.URL != address {
		return errors.New("unknown appliance")
	}
	if in.Remember {
		if err := s.saveNASCredential(id, address, in, session); err != nil {
			return fmt.Errorf("could not store saved sign-in in the system credential vault: %w", err)
		}
	} else if device.RememberSignIn {
		if err := s.removeNASCredential(id); err != nil {
			return err
		}
	}
	if in.Remember != device.RememberSignIn {
		if err := s.store.Update(func(c *Config) error {
			for i := range c.Devices {
				if c.Devices[i].ID == id {
					c.Devices[i].RememberSignIn = in.Remember
					return nil
				}
			}
			return errors.New("appliance was removed")
		}); err != nil {
			if in.Remember && s.vault != nil {
				_ = s.vault.Delete(id)
			}
			return err
		}
	}
	s.nasMu.Lock()
	if previous := s.nasSessions[id]; previous != nil && previous != session {
		previous.client.CloseIdleConnections()
	}
	s.nasSessions[id] = session
	s.nasMu.Unlock()
	s.setNASRecoveryStatus(id, "")
	return nil
}

func (s *Server) forgetNASCredentialAPI(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if err := s.decode(r, &in); err != nil {
		s.bad(w, err)
		return
	}
	lock := s.renewalLock(in.ID)
	lock.Lock()
	defer lock.Unlock()
	device, ok := s.store.Device(in.ID)
	if !ok {
		s.bad(w, errors.New("unknown appliance"))
		return
	}
	if device.RememberSignIn {
		if err := s.removeNASCredential(in.ID); err != nil {
			s.bad(w, err)
			return
		}
		if err := s.store.Update(func(c *Config) error {
			for i := range c.Devices {
				if c.Devices[i].ID == in.ID {
					c.Devices[i].RememberSignIn = false
					return nil
				}
			}
			return errors.New("unknown appliance")
		}); err != nil {
			s.bad(w, err)
			return
		}
	}
	s.setNASRecoveryStatus(in.ID, "")
	s.writeJSON(w, 200, map[string]any{"ok": true})
}

// A single guarded renewal prevents concurrent pages from producing a burst
// of NAS login attempts and triggering remote account lockout.
func (s *Server) renewRememberedNAS(ctx context.Context, id string, expected *nasSession) (*nasSession, error) {
	lock := s.renewalLock(id)
	lock.Lock()
	defer lock.Unlock()
	device, ok := s.store.Device(id)
	if !ok || !device.RememberSignIn || s.vault == nil {
		return nil, errors.New("sign in to this NAS again")
	}
	current, hasSession := s.getNASSession(id)
	if hasSession && current != expected && current != nil {
		return current, nil
	}
	s.nasMu.RLock()
	delayUntil := s.retryAfter[id]
	s.nasMu.RUnlock()
	if time.Now().Before(delayUntil) {
		return nil, errors.New("saved sign-in needs attention; try again later or enter a fresh verification code")
	}
	body, err := s.vault.Get(id)
	if err != nil {
		s.setNASRecoveryStatus(id, "Saved sign-in unavailable. Enter your NAS credentials to reconnect.")
		return nil, errors.New("the saved NAS sign-in is not available in the system credential vault")
	}
	var credentials savedNASCredential
	if err := json.Unmarshal(body, &credentials); err != nil ||
		credentials.URL != device.URL || credentials.Username == "" || credentials.Password == "" {
		s.setNASRecoveryStatus(id, "Saved sign-in needs to be set up again.")
		return nil, errors.New("saved NAS identity does not match this appliance")
	}
	var updated *nasSession
	// Resume an already-MFA-approved NAS session before reauthenticating.
	if credentials.SessionToken != "" && time.Now().Before(credentials.SessionExpiresAt.Add(-time.Minute)) {
		updated, err = resumeNASSavedCookie(ctx, device.URL, credentials)
	}
	if updated == nil {
		login := nasLoginRequest{Username: credentials.Username, Password: credentials.Password, AllowHTTP: credentials.AllowHTTP}
		updated, err = s.loginNAS(ctx, device.URL, login)
	}
	if err != nil {
		s.nasMu.Lock()
		s.retryAfter[id] = time.Now().Add(2 * time.Minute)
		s.restoreStatus[id] = "Sign in again. Your NAS may require a new verification code."
		s.nasMu.Unlock()
		return nil, err
	}
	// Refresh the vaulted cookie after a successful reauthentication.
	captureNASSessionCookie(&credentials, updated)
	if refreshed, marshalErr := json.Marshal(credentials); marshalErr == nil {
		_ = s.vault.Put(id, refreshed)
	}
	s.nasMu.Lock()
	if previous := s.nasSessions[id]; previous != nil {
		previous.client.CloseIdleConnections()
	}
	s.nasSessions[id] = updated
	delete(s.retryAfter, id)
	delete(s.restoreStatus, id)
	s.nasMu.Unlock()
	return updated, nil
}

// The NAS issues eight-hour sessions. Refresh before they expire, and recover
// from a NAS reboot that invalidates a cookie when a saved login is enabled.
func (s *Server) activeNASSession(ctx context.Context, id string) (*nasSession, bool) {
	current, ok := s.getNASSession(id)
	if ok && (current.expiresAt.IsZero() && time.Since(current.since) < 7*time.Hour+45*time.Minute || !current.expiresAt.IsZero() && time.Until(current.expiresAt) > 0) {
		return current, true
	}
	device, exists := s.store.Device(id)
	if !exists || !device.RememberSignIn || s.vault == nil {
		if ok {
			s.dropNASSession(id)
		}
		return nil, false
	}
	next, err := s.renewRememberedNAS(ctx, id, current)
	return next, err == nil
}

func (s *Server) restoreRememberedSessions() {
	var wg sync.WaitGroup
	limit := make(chan struct{}, 3)
	for _, device := range s.store.Snapshot().Devices {
		if !device.RememberSignIn {
			continue
		}
		wg.Add(1)
		s.setNASRecoveryStatus(device.ID, "Restoring saved sign-in…")
		go func(id string) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			_, _ = s.renewRememberedNAS(ctx, id, nil)
		}(device.ID)
	}
	wg.Wait()
}
