package config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	"cerebro/internal/domain"
)

const (
	// DefaultDebounceInterval coalesces bursts of fsnotify events.
	DefaultDebounceInterval = 300 * time.Millisecond
	// DefaultPollInterval is the content-hash fallback polling interval.
	DefaultPollInterval = 30 * time.Second
)

// Snapshot is an immutable, fully validated configuration snapshot.
type Snapshot struct {
	Config     *Config
	Hosts      *Hosts
	Digest     [32]byte
	Generation uint64
	LoadedAt   time.Time
}

// ManagerOption customizes a Manager.
type ManagerOption func(*Manager)

// WithLogger sets the logger used for reload and watcher messages.
func WithLogger(log logrus.FieldLogger) ManagerOption {
	return func(m *Manager) {
		if log != nil {
			m.log = log
		}
	}
}

// WithDebounce sets the fsnotify burst coalescing window.
func WithDebounce(d time.Duration) ManagerOption {
	return func(m *Manager) { m.debounce = d }
}

// WithPollInterval sets the content-hash polling interval (0 disables it).
func WithPollInterval(d time.Duration) ManagerOption {
	return func(m *Manager) { m.pollInterval = d }
}

// WithFSNotify enables or disables the inotify watcher.
func WithFSNotify(enabled bool) ManagerOption {
	return func(m *Manager) { m.fsnotifyOn = enabled }
}

// WithOnReload registers a callback invoked when hosts change on reload.
func WithOnReload(fn func(prev, next *Snapshot)) ManagerOption {
	return func(m *Manager) { m.onReload = fn }
}

// Manager owns the current configuration snapshot and reloads it safely.
type Manager struct {
	absPath string
	dir     string
	base    string

	current  atomic.Pointer[Snapshot]
	reloadMu sync.Mutex

	log          logrus.FieldLogger
	debounce     time.Duration
	pollInterval time.Duration
	fsnotifyOn   bool
	onReload     func(prev, next *Snapshot)

	lifecycleMu sync.Mutex
	closeOnce   sync.Once
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewManager loads path and returns a Manager holding the initial snapshot.
func NewManager(path string, opts ...ManagerOption) (*Manager, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("config path must not be empty")
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("resolve config path %s: %w", path, err)
	}

	m := &Manager{
		absPath:      absPath,
		dir:          filepath.Dir(absPath),
		base:         filepath.Base(absPath),
		log:          logrus.StandardLogger(),
		debounce:     DefaultDebounceInterval,
		pollInterval: DefaultPollInterval,
		fsnotifyOn:   true,
		stopCh:       make(chan struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}

	raw, err := readConfigFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", absPath, err)
	}
	cfg, err := parseConfigBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", absPath, err)
	}
	hosts, err := NewHosts(cfg.Hosts)
	if err != nil {
		return nil, fmt.Errorf("validate config %s: %w", absPath, err)
	}

	m.current.Store(&Snapshot{
		Config:     cfg,
		Hosts:      hosts,
		Digest:     sha256.Sum256(raw),
		Generation: 1,
		LoadedAt:   time.Now().UTC(),
	})
	return m, nil
}

// Snapshot returns the current immutable snapshot.
func (m *Manager) Snapshot() *Snapshot {
	if m == nil {
		return nil
	}
	return m.current.Load()
}

// Reload re-reads the configuration file and swaps the snapshot when its
// content changed. Any validation failure keeps the last-known-good snapshot.
func (m *Manager) Reload() (bool, error) {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()

	raw, err := readConfigFile(m.absPath)
	if err != nil {
		m.log.WithError(err).WithField("path", m.absPath).Error("configuration reload failed; keeping previous snapshot")
		return false, err
	}

	digest := sha256.Sum256(raw)
	prev := m.current.Load()
	if prev != nil && digest == prev.Digest {
		return false, nil
	}

	cfg, err := parseConfigBytes(raw)
	if err != nil {
		m.log.WithError(err).WithField("path", m.absPath).Error("configuration reload failed; keeping previous snapshot")
		return false, err
	}
	hosts, err := NewHosts(cfg.Hosts)
	if err != nil {
		m.log.WithError(err).WithField("path", m.absPath).Error("configuration reload failed; keeping previous snapshot")
		return false, err
	}

	next := &Snapshot{
		Config:     cfg,
		Hosts:      hosts,
		Digest:     digest,
		Generation: 1,
		LoadedAt:   time.Now().UTC(),
	}
	if prev != nil {
		next.Generation = prev.Generation + 1
	}
	m.current.Store(next)

	if prev == nil {
		return true, nil
	}

	hostsDiff := DiffHosts(prev.Hosts, next.Hosts)
	restartFields := restartOnlyDiff(prev.Config, next.Config)

	if !hostsDiff.IsEmpty() && m.onReload != nil {
		m.onReload(prev, next)
	}

	if len(prev.Hosts.GetHostNames()) > 0 && len(next.Hosts.GetHostNames()) == 0 {
		m.log.WithField("prev_hosts_count", len(prev.Hosts.GetHostNames())).Warn("reloaded configuration removed all previously configured hosts")
	}
	if len(restartFields) > 0 {
		m.log.WithField("fields", restartFields).Warn("configuration change to non-reloadable field(s) requires restart; reloadable fields were applied")
	}

	m.log.WithFields(logrus.Fields{
		"generation":    next.Generation,
		"digest":        fmt.Sprintf("%x", next.Digest[:6]),
		"hosts_total":   len(next.Hosts.GetHostNames()),
		"hosts_added":   hostsDiff.Added,
		"hosts_removed": hostsDiff.Removed,
		"hosts_changed": hostsDiff.Changed,
	}).Info("configuration reloaded")

	return true, nil
}

// Watch blocks until ctx is cancelled or Close is called, reloading on
// debounced fsnotify events and the polling fallback.
func (m *Manager) Watch(ctx context.Context, logger logrus.FieldLogger) {
	m.lifecycleMu.Lock()
	select {
	case <-m.stopCh:
		m.lifecycleMu.Unlock()
		return
	default:
	}
	if ctx.Err() != nil {
		m.lifecycleMu.Unlock()
		return
	}
	m.wg.Add(1)
	m.lifecycleMu.Unlock()
	defer m.wg.Done()

	log := m.log
	if logger != nil {
		log = logger
	}

	var eventsCh <-chan fsnotify.Event
	var errorsCh <-chan error
	var watcher *fsnotify.Watcher
	if m.fsnotifyOn {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			log.WithError(err).Warn("fsnotify watcher unavailable; relying on polling fallback")
		} else if err := w.Add(m.dir); err != nil {
			log.WithError(err).Warn("fsnotify watcher unavailable; relying on polling fallback")
			_ = w.Close()
		} else {
			watcher = w
			eventsCh = w.Events
			errorsCh = w.Errors
			defer func() { _ = w.Close() }()
		}
	}

	var debounceC <-chan time.Time
	var debounceTimer *time.Timer
	if m.debounce > 0 {
		debounceTimer = time.NewTimer(m.debounce)
		if !debounceTimer.Stop() {
			<-debounceTimer.C
		}
		defer debounceTimer.Stop()
	}
	onEvent := func() {
		if debounceTimer == nil {
			_, _ = m.Reload()
			return
		}
		if !debounceTimer.Stop() {
			select {
			case <-debounceTimer.C:
			default:
			}
		}
		debounceTimer.Reset(m.debounce)
		debounceC = debounceTimer.C
	}

	var pollC <-chan time.Time
	var ticker *time.Ticker
	if m.pollInterval > 0 {
		ticker = time.NewTicker(m.pollInterval)
		defer ticker.Stop()
		pollC = ticker.C
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case event, ok := <-eventsCh:
			if !ok {
				eventsCh = nil
				continue
			}
			if isRelevantEvent(event, m.base) {
				onEvent()
			}
		case err, ok := <-errorsCh:
			if !ok {
				errorsCh = nil
				continue
			}
			log.WithError(err).Warn("fsnotify watcher error")
		case <-debounceC:
			debounceC = nil
			_, _ = m.Reload()
		case <-pollC:
			if watcher != nil {
				_ = watcher.Add(m.dir)
			}
			_, _ = m.Reload()
		}
	}
}

// Close stops the watcher and waits for its goroutine to exit. It is
// idempotent and safe to call when Watch was never started.
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.closeOnce.Do(func() {
		m.lifecycleMu.Lock()
		close(m.stopCh)
		m.lifecycleMu.Unlock()
	})
	m.wg.Wait()
	return nil
}

// GetHost resolves a host from the current snapshot.
func (m *Manager) GetHost(name string) (domain.Host, bool) {
	snap := m.Snapshot()
	if snap == nil {
		return domain.Host{}, false
	}
	return snap.Hosts.GetHost(name)
}

// GetHostNames returns the configured host names from the current snapshot.
func (m *Manager) GetHostNames() []string {
	snap := m.Snapshot()
	if snap == nil {
		return []string{}
	}
	return snap.Hosts.GetHostNames()
}

func isRelevantEvent(event fsnotify.Event, base string) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
		return false
	}
	name := filepath.Base(event.Name)
	return name == base || name == "..data"
}
