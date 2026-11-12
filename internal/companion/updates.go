package companion

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

const publicReleasesURL = "https://github.com/gigabytegrove/zero1local-desktop/releases"

// UpdateStatus is shown in Settings without giving the remote NAS access to
// the desktop update engine. Updates are always initiated by the operator.
type UpdateStatus struct {
	Version    string    `json:"version"`
	Channel    string    `json:"channel"`
	Supported  bool      `json:"supported"`
	Reason     string    `json:"reason,omitempty"`
	Checking   bool      `json:"checking"`
	LastError  string    `json:"lastError,omitempty"`
	LastCheck  time.Time `json:"lastCheck,omitempty"`
	ReleaseURL string    `json:"releaseUrl"`
}

type updateControl struct {
	mu     sync.Mutex
	state  UpdateStatus
	run    func(context.Context) error
	cancel context.CancelFunc
}

func (s *Server) ConfigureUpdater(version, channel string, run func(context.Context) error, unavailableReason string) {
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	s.updates.run = run
	s.updates.state = UpdateStatus{
		Version:    version,
		Channel:    channel,
		Supported:  run != nil,
		Reason:     unavailableReason,
		ReleaseURL: publicReleasesURL,
	}
}

func (s *Server) UpdateStatus() UpdateStatus {
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	info := s.updates.state
	if info.Version == "" {
		info.Version = "development preview"
		info.Channel = "beta"
		info.Reason = "The updater is available in the native desktop application."
		info.ReleaseURL = publicReleasesURL
	}
	return info
}

// StartUpdateCheck is shared by the trusted local Settings UI and native
// tray menu. It prevents two interactive updater flows from racing.
func (s *Server) StartUpdateCheck() error {
	s.updates.mu.Lock()
	if s.updates.run == nil {
		reason := s.updates.state.Reason
		s.updates.mu.Unlock()
		if reason == "" {
			reason = "Updates are only supported in the installed native desktop application."
		}
		return errors.New(reason)
	}
	if s.updates.state.Checking {
		s.updates.mu.Unlock()
		return errors.New("An update check is already running.")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.updates.cancel = cancel
	s.updates.state.Checking = true
	s.updates.state.LastError = ""
	run := s.updates.run
	s.updates.mu.Unlock()

	go func() {
		err := run(ctx)
		s.updates.mu.Lock()
		s.updates.state.Checking = false
		s.updates.state.LastCheck = time.Now().UTC()
		if err != nil {
			s.updates.state.LastError = err.Error()
		}
		s.updates.cancel = nil
		s.updates.mu.Unlock()
		cancel()
	}()
	return nil
}

func (s *Server) stopUpdater() {
	s.updates.mu.Lock()
	cancel := s.updates.cancel
	s.updates.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Server) updateCheckAPI(w http.ResponseWriter, r *http.Request) {
	if err := s.StartUpdateCheck(); err != nil {
		s.bad(w, err)
		return
	}
	s.writeJSON(w, http.StatusAccepted, map[string]any{"started": true})
}

func (s *Server) updateReleasesAPI(w http.ResponseWriter, r *http.Request) {
	if s.launcher == nil {
		s.bad(w, errors.New("Open the release URL in a browser: "+publicReleasesURL))
		return
	}
	if err := s.launcher(publicReleasesURL); err != nil {
		s.bad(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"opened": true})
}
