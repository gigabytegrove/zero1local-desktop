package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"strconv"
	"strings"
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
	Remember bool `json:"remember"`
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
	"network":       "/api/network",
	"docker":        "/api/docker",
	"hardware":      "/api/hardware",
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
		Timeout:   15 * time.Second,
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
	if err:=s.rememberExistingNAS(device.ID,device.URL,in);err!=nil {session.client.CloseIdleConnections();s.bad(w,err);return}
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
	var in struct {
		ID string `json:"id"`
	}
	if err := s.decode(r, &in); err != nil {
		s.bad(w, err)
		return
	}
	if d,ok:=s.store.Device(in.ID);ok&&d.RememberSignIn{
  if err:=s.removeNASCredential(in.ID);err!=nil{s.bad(w,err);return}
  if err:=s.store.Update(func(c *Config)error{for i:=range c.Devices{if c.Devices[i].ID==in.ID{c.Devices[i].RememberSignIn=false;return nil}};return errors.New("unknown appliance")});err!=nil{s.bad(w,err);return}
 }
 s.setNASRecoveryStatus(in.ID,"")
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
	session, ok := s.activeNASSession(r.Context(),in.ID)
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
	var resp *http.Response
 for attempt:=0;attempt<2;attempt++{
  req,err:=http.NewRequestWithContext(r.Context(),http.MethodGet,target,nil)
  if err!=nil{s.bad(w,err);return}
  req.Header.Set("Accept","application/json")
  req.Header.Set("User-Agent","Zero1Local-Desktop")
  resp,err=session.client.Do(req)
  if err!=nil{s.writeJSON(w,http.StatusBadGateway,map[string]string{"error":"Unable to reach the NAS: "+err.Error()});return}
  if resp.StatusCode==http.StatusUnauthorized && attempt==0{
   resp.Body.Close()
   refreshed,err:=s.renewRememberedNAS(r.Context(),in.ID,session)
   if err==nil{
    session=refreshed
    continue
   }
   s.dropNASSession(in.ID)
   s.writeJSON(w,http.StatusUnauthorized,map[string]string{"error":"NAS session expired. Sign in again if this NAS needs a verification code."})
   return
  }
  break
 }
 defer resp.Body.Close()
 if resp.StatusCode==http.StatusUnauthorized||resp.StatusCode==http.StatusForbidden{
  if resp.StatusCode==http.StatusUnauthorized{s.dropNASSession(in.ID)}
  s.writeJSON(w,resp.StatusCode,map[string]string{"error":"NAS session expired or this account is not permitted to access this section"})
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

// File operations are deliberately restricted to the NAS's existing,
// authenticated, share-rooted handlers. No arbitrary proxy requests or
// NAS shell commands are allowed. The NAS performs final ACL and path checks.
type nasFileAction struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}

func (s *Server) nasFileAction(w http.ResponseWriter, r *http.Request) {
	var in nasFileAction
	if err := s.decode(r, &in); err != nil {
		s.bad(w, err)
		return
	}
	session, ok := s.activeNASSession(r.Context(),in.ID)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "NAS administrator sign-in is required"})
		return
	}
	if in.Path == "" || len(in.Path) > 4096 {
		s.bad(w, errors.New("invalid file path"))
		return
	}
	endpoint := ""
	payload := map[string]string{"Path": in.Path}
	switch in.Action {
	case "mkdir":
		endpoint = "/api/files/mkdir"
		payload["Name"] = in.Name
	case "rename":
		endpoint = "/api/files/rename"
		payload["NewName"] = in.Name
	case "trash":
		endpoint = "/api/files/trash"
	default:
		s.bad(w, errors.New("unsupported file operation"))
		return
	}
	if in.Action != "trash" {
		if !validNASFileName(in.Name) {
			s.bad(w, errors.New("invalid file or folder name"))
			return
		}
	} else if in.Name != "" {
		s.bad(w, errors.New("unexpected trash operation argument"))
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		s.bad(w, err)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, session.origin+endpoint, bytes.NewReader(data))
	if err != nil {
		s.bad(w, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Zero1Local-Request", "1")
	req.Header.Set("User-Agent", "Zero1Local-Desktop")
	resp, err := session.client.Do(req)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS file operation failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
	if err != nil || !json.Valid(raw) {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS returned an invalid file operation response"})
		return
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS file operation response could not be decoded"})
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := result["error"].(string)
		s.writeJSON(w, resp.StatusCode, map[string]string{"error": limitedNASMessage(msg)})
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func validNASFileName(name string) bool {
	if name == "" || len(name) > 255 || name == "." || name == ".." || strings.TrimSpace(name) != name {
		return false
	}
	return !strings.ContainsAny(name, "/\\\x00\r\n")
}

const nasTransferLimit int64 = 256 << 20

func (s *Server) downloadNASFile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := s.decode(r, &in); err != nil {
		s.bad(w, err)
		return
	}
	session, ok := s.activeNASSession(r.Context(),in.ID)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "NAS administrator sign-in is required"})
		return
	}
	if in.Path == "" || len(in.Path) > 4096 {
		s.bad(w, errors.New("invalid file path"))
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
		session.origin+"/api/files/download?"+url.Values{"path": {in.Path}}.Encode(), nil)
	if err != nil {
		s.bad(w, err)
		return
	}
	req.Header.Set("User-Agent", "Zero1Local-Desktop")
	transferClient := *session.client
	transferClient.Timeout = 15 * time.Minute
	resp, err := transferClient.Do(req)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS download failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("NAS file download returned HTTP %d", resp.StatusCode),
		})
		return
	}
	if resp.ContentLength < 0 || resp.ContentLength > nasTransferLimit {
		s.writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
			"error": "This download is larger than 256 MiB or has no known size. Use the full NAS file manager for this file or folder.",
		})
		return
	}
	filename := path.Base(in.Path)
	if !validNASFileName(filename) {
		s.bad(w, errors.New("invalid download filename"))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.CopyN(w, resp.Body, resp.ContentLength)
}

func (s *Server) uploadNASFile(w http.ResponseWriter, r *http.Request) {
	id, folder := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	session, ok := s.activeNASSession(r.Context(),id)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "NAS administrator sign-in is required"})
		return
	}
	if folder == "" || len(folder) > 4096 {
		s.bad(w, errors.New("invalid upload folder"))
		return
	}
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != "multipart/form-data" {
		s.bad(w, errors.New("expected a multipart file upload"))
		return
	}
	if r.ContentLength < 0 || r.ContentLength > nasTransferLimit {
		s.writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Upload must be 256 MiB or smaller"})
		return
	}
	target := session.origin + "/api/files/upload?" + url.Values{"path": {folder}}.Encode()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, r.Body)
	if err != nil {
		s.bad(w, err)
		return
	}
	req.ContentLength = r.ContentLength
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	req.Header.Set("X-Zero1Local-Request", "1")
	req.Header.Set("User-Agent", "Zero1Local-Desktop")
	transferClient := *session.client
	transferClient.Timeout = 15 * time.Minute
	resp, err := transferClient.Do(req)
	if err != nil {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS upload failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
	if err != nil || !json.Valid(raw) {
		s.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NAS returned an invalid upload response"})
		return
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		s.bad(w, err)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := result["error"].(string)
		s.writeJSON(w, resp.StatusCode, map[string]string{"error": limitedNASMessage(msg)})
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}
