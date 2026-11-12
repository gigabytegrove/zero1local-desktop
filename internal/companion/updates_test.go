package companion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDesktopUpdaterStatesAndSingleFlight(t *testing.T) {
	store := tempStore(t)
	svc, err := New(store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if svc.UpdateStatus().Supported {
		t.Fatal("preview must not claim that the native updater works")
	}
	block := make(chan struct{})
	started := make(chan struct{})
	svc.ConfigureUpdater("0.1.0-beta.2", "beta", func(ctx context.Context) error {
		close(started)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-block:
			return errors.New("example network failure")
		}
	}, "")
	if err := svc.StartUpdateCheck(); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := svc.StartUpdateCheck(); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("parallel updater started: %v", err)
	}
	if !svc.UpdateStatus().Checking {
		t.Fatal("checking state was lost")
	}
	close(block)
	deadline := time.Now().Add(time.Second)
	for svc.UpdateStatus().Checking && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	result := svc.UpdateStatus()
	if result.Checking || result.LastError != "example network failure" || result.LastCheck.IsZero() {
		t.Fatalf("unexpected state after error: %+v", result)
	}
}

func TestUpdaterUnavailableForManagedInstaller(t *testing.T) {
	svc, err := New(tempStore(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.ConfigureUpdater("0.1.0-beta.2", "beta", nil, "Install the updated Debian package.")
	if err := svc.StartUpdateCheck(); err == nil || !strings.Contains(err.Error(), "Debian") {
		t.Fatalf("unsupported updater incorrectly started: %v", err)
	}
	if svc.UpdateStatus().Supported {
		t.Fatal("managed package must not advertise binary replacement")
	}
}
