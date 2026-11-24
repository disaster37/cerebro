package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func newTestLogger() (*logrus.Logger, *syncBuffer) {
	buf := &syncBuffer{}
	log := logrus.New()
	log.SetOutput(buf)
	log.SetLevel(logrus.DebugLevel)
	return log, buf
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func configWithHosts(hosts ...string) string {
	var b strings.Builder
	b.WriteString("secret: \"s\"\nbasePath: \"/\"\nhosts:\n")
	for i, h := range hosts {
		fmt.Fprintf(&b, "  - host: %q\n    name: %q\n", h, fmt.Sprintf("es%d", i+1))
	}
	return b.String()
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func TestManagerReloadTable(t *testing.T) {
	t.Run("valid change", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "application.yaml")
		writeConfig(t, path, configWithHosts("http://es1:9200"))

		reloads := 0
		mgr, err := NewManager(path, WithOnReload(func(_, _ *Snapshot) { reloads++ }))
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}
		if snap := mgr.Snapshot(); snap.Generation != 1 {
			t.Fatalf("generation = %d, want 1", snap.Generation)
		}

		writeConfig(t, path, configWithHosts("http://es1:9200", "http://es2:9200"))
		changed, err := mgr.Reload()
		if err != nil || !changed {
			t.Fatalf("Reload = (%t, %v), want (true, nil)", changed, err)
		}
		if got := mgr.GetHostNames(); len(got) != 2 {
			t.Fatalf("hosts = %#v, want 2", got)
		}
		if snap := mgr.Snapshot(); snap.Generation != 2 {
			t.Fatalf("generation = %d, want 2", snap.Generation)
		}
		if reloads != 1 {
			t.Fatalf("onReload calls = %d, want 1", reloads)
		}
	})

	invalid := []struct {
		name    string
		content string
	}{
		{"invalid yaml", "hosts: [\n"},
		{"non-map root", "- just-a-list\n"},
		{"oversized", strings.Repeat("a", MaxConfigBytes+1)},
		{"unknown type", configWithHosts("http://es1:9200") + "    type: mysql\n"},
		{"duplicate name", "hosts:\n  - host: \"http://es1:9200\"\n    name: dup\n  - host: \"http://es2:9200\"\n    name: dup\n"},
		{"empty host", "hosts:\n  - host: \"\"\n    name: es\n"},
		{"whitespace name", "hosts:\n  - host: \"http://es1:9200\"\n    name: \"   \"\n"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "application.yaml")
			writeConfig(t, path, configWithHosts("http://es1:9200"))
			log, buf := newTestLogger()
			mgr, err := NewManager(path, WithLogger(log))
			if err != nil {
				t.Fatalf("NewManager: %v", err)
			}
			before := mgr.Snapshot()

			writeConfig(t, path, tc.content)
			changed, err := mgr.Reload()
			if err == nil || changed {
				t.Fatalf("Reload = (%t, %v), want (false, error)", changed, err)
			}
			after := mgr.Snapshot()
			if after != before {
				t.Fatal("snapshot changed after failed reload")
			}
			if before.Generation != after.Generation {
				t.Fatalf("generation = %d, want %d", after.Generation, before.Generation)
			}
			if !strings.Contains(buf.String(), "configuration reload failed") {
				t.Fatalf("expected error log, got %q", buf.String())
			}
		})
	}

	t.Run("remove all hosts warns", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "application.yaml")
		writeConfig(t, path, configWithHosts("http://es1:9200"))
		log, buf := newTestLogger()
		mgr, err := NewManager(path, WithLogger(log))
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}
		writeConfig(t, path, "secret: \"s\"\nhosts: []\n")
		changed, err := mgr.Reload()
		if err != nil || !changed {
			t.Fatalf("Reload = (%t, %v), want (true, nil)", changed, err)
		}
		if got := mgr.GetHostNames(); len(got) != 0 {
			t.Fatalf("hosts = %#v, want empty", got)
		}
		if !strings.Contains(buf.String(), "removed all previously configured hosts") {
			t.Fatalf("expected empty-hosts warning, got %q", buf.String())
		}
	})

	t.Run("restart-only warning is not repeated", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "application.yaml")
		writeConfig(t, path, "secret: \"s\"\nbasePath: \"/a\"\nes:\n  gzip: false\nhosts:\n  - host: \"http://es1:9200\"\n    name: es1\n")
		log, buf := newTestLogger()
		mgr, err := NewManager(path, WithLogger(log))
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}

		writeConfig(t, path, "secret: \"s\"\nbasePath: \"/b\"\nes:\n  gzip: true\nhosts:\n  - host: \"http://es1:9200\"\n    name: es1\n  - host: \"http://es2:9200\"\n    name: es2\n")
		if changed, err := mgr.Reload(); err != nil || !changed {
			t.Fatalf("first Reload = (%t, %v)", changed, err)
		}
		first := buf.String()
		for _, field := range []string{"basePath", "es.gzip"} {
			if !strings.Contains(first, field) {
				t.Fatalf("first reload log missing %q: %s", field, first)
			}
		}

		writeConfig(t, path, "secret: \"s\"\nbasePath: \"/b\"\nes:\n  gzip: true\nhosts:\n  - host: \"http://es1:9200\"\n    name: es1\n  - host: \"http://es2:9200\"\n    name: es2\n  - host: \"http://es3:9200\"\n    name: es3\n")
		if changed, err := mgr.Reload(); err != nil || !changed {
			t.Fatalf("second Reload = (%t, %v)", changed, err)
		}
		if strings.Count(buf.String(), "non-reloadable") != 1 {
			t.Fatalf("restart-only warning not emitted exactly once: %s", buf.String())
		}
	})
}

func TestManagerReloadNoOpDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	content := configWithHosts("http://es1:9200")
	writeConfig(t, path, content)
	log, buf := newTestLogger()
	mgr, err := NewManager(path, WithLogger(log))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	before := mgr.Snapshot()
	writeConfig(t, path, content)
	changed, err := mgr.Reload()
	if err != nil || changed {
		t.Fatalf("Reload = (%t, %v), want (false, nil)", changed, err)
	}
	if mgr.Snapshot() != before {
		t.Fatal("snapshot changed on identical bytes")
	}
	if strings.Contains(buf.String(), "configuration reloaded") {
		t.Fatalf("unexpected reload log: %q", buf.String())
	}
}

func TestManagerLogsNeverContainCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	initial := `
secret: "super-secret"
auth:
  type: "basic"
  settings:
    password: "basic-password"
    bind-pw: "bind-password"
hosts:
  - host: "http://inlineuser:inlinepass@es1:9200"
    name: "es1"
    auth:
      username: "host-user"
      password: "host-password"
`
	writeConfig(t, path, initial)
	log, buf := newTestLogger()
	mgr, err := NewManager(path, WithLogger(log))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	rotated := strings.NewReplacer(
		"super-secret", "super-secret-2",
		"basic-password", "basic-password-2",
		"bind-password", "bind-password-2",
		"host-password", "host-password-2",
		"inlinepass", "inlinepass-2",
	).Replace(initial)
	writeConfig(t, path, rotated)
	if changed, err := mgr.Reload(); err != nil || !changed {
		t.Fatalf("Reload = (%t, %v)", changed, err)
	}

	writeConfig(t, path, "hosts: [\nsecret: \"super-secret-3\"\n")
	if _, err := mgr.Reload(); err == nil {
		t.Fatal("expected failed reload")
	}

	out := buf.String()
	for _, secret := range []string{
		"super-secret", "basic-password", "bind-password", "host-password",
		"host-user", "inlinepass", "inlineuser", "super-secret-2",
	} {
		if strings.Contains(out, secret) {
			t.Fatalf("logs leaked %q: %s", secret, out)
		}
	}
}

func TestManagerWatchDebounceCoalescesBurst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	writeConfig(t, path, configWithHosts("http://es1:9200"))
	mgr, err := NewManager(path, WithDebounce(80*time.Millisecond), WithPollInterval(0))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.Watch(ctx, nil)
	defer func() { _ = mgr.Close() }()
	time.Sleep(150 * time.Millisecond)

	for i := 0; i < 5; i++ {
		writeConfig(t, path, configWithHosts("http://es1:9200", "http://es2:9200"))
		time.Sleep(4 * time.Millisecond)
	}

	if !waitFor(t, 2*time.Second, func() bool { return mgr.Snapshot().Generation >= 2 }) {
		t.Fatal("reload did not happen")
	}
	time.Sleep(250 * time.Millisecond)
	if got := mgr.Snapshot().Generation; got != 2 {
		t.Fatalf("generation = %d, want 2 (burst not coalesced)", got)
	}
}

func TestManagerWatchKubernetesSymlinkSwap(t *testing.T) {
	if err := probeFSNotify(t.TempDir()); err != nil {
		t.Skipf("fsnotify unavailable: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	writeTimestamped(t, dir, "..2026_11_24_01", configWithHosts("http://es1:9200"))
	linkTimestamped(t, dir, "..2026_11_24_01")

	mgr, err := NewManager(path, WithDebounce(50*time.Millisecond), WithPollInterval(0))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.Watch(ctx, nil)
	defer func() { _ = mgr.Close() }()
	time.Sleep(150 * time.Millisecond)

	swapTimestamped(t, dir, "..2026_11_24_02", configWithHosts("http://es1:9200", "http://es2:9200"), "..2026_11_24_01")
	if !waitFor(t, 2*time.Second, func() bool { return len(mgr.GetHostNames()) == 2 }) {
		t.Fatalf("swap #1 not picked up: %#v", mgr.GetHostNames())
	}

	swapTimestamped(t, dir, "..2026_11_24_03", configWithHosts("http://es3:9200"), "..2026_11_24_02")
	if !waitFor(t, 2*time.Second, func() bool {
		names := mgr.GetHostNames()
		return len(names) == 1 && names[0] == "es1"
	}) {
		t.Fatalf("swap #2 not picked up: %#v", mgr.GetHostNames())
	}
}

func probeFSNotify(dir string) error {
	probe, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = probe.Close() }()
	return probe.Add(dir)
}

func writeTimestamped(t *testing.T, dir, name, content string) {
	t.Helper()
	sub := filepath.Join(dir, name)
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", sub, err)
	}
	writeConfig(t, filepath.Join(sub, "application.yaml"), content)
}

func linkTimestamped(t *testing.T, dir, target string) {
	t.Helper()
	if err := os.Symlink(target, filepath.Join(dir, "..data")); err != nil {
		t.Fatalf("symlink ..data: %v", err)
	}
	if err := os.Symlink(filepath.Join("..data", "application.yaml"), filepath.Join(dir, "application.yaml")); err != nil {
		t.Fatalf("symlink application.yaml: %v", err)
	}
}

func swapTimestamped(t *testing.T, dir, newDir, content, oldDir string) {
	t.Helper()
	writeTimestamped(t, dir, newDir, content)
	tmp := filepath.Join(dir, "..data_tmp")
	_ = os.Remove(tmp)
	if err := os.Symlink(newDir, tmp); err != nil {
		t.Fatalf("symlink ..data_tmp: %v", err)
	}
	if err := os.Rename(tmp, filepath.Join(dir, "..data")); err != nil {
		t.Fatalf("rename ..data: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(dir, oldDir)); err != nil {
		t.Fatalf("remove old dir: %v", err)
	}
}

func TestManagerWatchPollingFallbackWithoutFSNotify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	writeConfig(t, path, configWithHosts("http://es1:9200"))
	mgr, err := NewManager(path, WithFSNotify(false), WithPollInterval(40*time.Millisecond))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.Watch(ctx, nil)
	defer func() { _ = mgr.Close() }()

	writeConfig(t, path, configWithHosts("http://es1:9200", "http://es2:9200"))
	if !waitFor(t, 500*time.Millisecond, func() bool { return len(mgr.GetHostNames()) == 2 }) {
		t.Fatalf("polling fallback did not reload: %#v", mgr.GetHostNames())
	}
}

func TestManagerConcurrentReadsAndReloadsRace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	writeConfig(t, path, configWithHosts("http://es1:9200"))
	mgr, err := NewManager(path, WithDebounce(10*time.Millisecond), WithPollInterval(0))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.Watch(ctx, nil)
	defer func() { _ = mgr.Close() }()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				mgr.GetHost("es1")
				mgr.GetHostNames()
				mgr.Snapshot()
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if (i+j)%2 == 0 {
					writeConfig(t, path, configWithHosts("http://es1:9200", "http://es2:9200"))
				} else {
					writeConfig(t, path, "hosts: [\n")
				}
				_, _ = mgr.Reload()
			}
		}(i)
	}
	time.Sleep(250 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestManagerCloseIdempotentAndNoGoroutineLeak(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yaml")
	writeConfig(t, path, configWithHosts("http://es1:9200"))

	baseline := runtime.NumGoroutine()
	mgr, err := NewManager(path, WithPollInterval(10*time.Millisecond))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.Watch(ctx, nil)
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := mgr.Close(); err != nil {
				t.Errorf("Close: %v", err)
			}
		}()
	}
	wg.Wait()

	if !waitFor(t, 2*time.Second, func() bool { return runtime.NumGoroutine() <= baseline }) {
		t.Fatalf("goroutines = %d, baseline = %d", runtime.NumGoroutine(), baseline)
	}

	neverStarted, err := NewManager(path)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := neverStarted.Close(); err != nil {
		t.Fatalf("Close without Watch: %v", err)
	}
	if err := neverStarted.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
