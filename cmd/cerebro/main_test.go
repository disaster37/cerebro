package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"cerebro/internal/config"
)

func TestSignalWaiterSIGHUPReloadsWithoutStopping(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	initial := "hosts:\n  - host: \"http://es1:9200\"\n    name: es1\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)

	mgr, err := config.NewManager(path, config.WithLogger(log), config.WithFSNotify(false), config.WithPollInterval(0))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	ready := make(chan struct{})
	result := make(chan error, 1)
	waiter := waitForSignals(mgr, log, func() {}, ready)
	go func() {
		result <- waiter(make(chan error))
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("signal waiter did not become ready")
	}

	updated := "hosts:\n  - host: \"http://es1:9200\"\n    name: es1\n  - host: \"http://es2:9200\"\n    name: es2\n"
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatalf("update config: %v", err)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("send SIGHUP: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(mgr.GetHostNames()) != 2 {
		time.Sleep(10 * time.Millisecond)
	}
	if got := mgr.GetHostNames(); len(got) != 2 {
		t.Fatalf("hosts after SIGHUP = %#v, want 2", got)
	}

	select {
	case err := <-result:
		t.Fatalf("signal waiter returned after SIGHUP: %v", err)
	default:
	}

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("signal waiter returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("signal waiter did not return after SIGTERM")
	}
}
