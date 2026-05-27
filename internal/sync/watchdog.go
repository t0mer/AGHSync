package sync

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watchdogDebounce = 2 * time.Second

// Watchdog watches a file path for write/create events and triggers a sync via
// the Dispatcher. Changes are debounced to avoid rapid-fire triggers when
// AdGuardHome rewrites its config file in multiple steps.
type Watchdog struct {
	dispatcher *Dispatcher
	logger     *slog.Logger
	mu         sync.Mutex
	path       string
	watcher    *fsnotify.Watcher
	stop       chan struct{}
	stopped    chan struct{}
}

// NewWatchdog creates a Watchdog backed by the given Dispatcher.
func NewWatchdog(dispatcher *Dispatcher, logger *slog.Logger) *Watchdog {
	return &Watchdog{
		dispatcher: dispatcher,
		logger:     logger,
	}
}

// Start begins watching path for changes. Any existing watch is stopped first.
// Pass "" to stop without starting a new watch.
func (w *Watchdog) Start(path string) error {
	w.Stop()
	if path == "" {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	// Watch the parent directory rather than the file itself. AdGuardHome writes
	// its config atomically (write temp file → rename), which makes a file-level
	// watch go stale after the first save. A directory watch survives renames and
	// receives a Create/Rename event for the target file on every atomic replace.
	dir := filepath.Dir(path)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return fmt.Errorf("watch %q: %w", dir, err)
	}

	stop := make(chan struct{})
	stopped := make(chan struct{})

	w.mu.Lock()
	w.path = path
	w.watcher = watcher
	w.stop = stop
	w.stopped = stopped
	w.mu.Unlock()

	go w.run(watcher, stop, stopped)
	w.logger.Info("watchdog: started", "path", path)
	return nil
}

// Stop stops the watchdog if it is running.
func (w *Watchdog) Stop() {
	w.mu.Lock()
	stop := w.stop
	stopped := w.stopped
	watcher := w.watcher
	w.stop = nil
	w.stopped = nil
	w.watcher = nil
	w.path = ""
	w.mu.Unlock()

	if stop != nil {
		close(stop)
		<-stopped
	}
	if watcher != nil {
		watcher.Close()
	}
}

// IsRunning reports whether the watchdog is currently active.
func (w *Watchdog) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stop != nil
}

// Path returns the currently watched path, or "" if not watching.
func (w *Watchdog) Path() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.path
}

func (w *Watchdog) run(watcher *fsnotify.Watcher, stop, stopped chan struct{}) {
	defer close(stopped)

	w.mu.Lock()
	filename := filepath.Base(w.path)
	w.mu.Unlock()

	var debounce *time.Timer
	for {
		select {
		case <-stop:
			if debounce != nil {
				debounce.Stop()
			}
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Ignore events for other files in the same directory.
			if filepath.Base(event.Name) != filename {
				continue
			}
			// Write: in-place edit. Create/Rename: atomic replace (write temp → rename).
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				w.logger.Debug("watchdog: file changed, debouncing", "path", event.Name, "op", event.Op)
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(watchdogDebounce, func() {
					if _, err := w.dispatcher.Submit(context.Background(), "watchdog"); err != nil && err != ErrSyncBusy && err != ErrNoSlaves {
						w.logger.Warn("watchdog: failed to submit sync", "err", err)
					} else if err == nil {
						w.logger.Info("watchdog: sync triggered by file change")
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			w.logger.Warn("watchdog: watcher error", "err", err)
		}
	}
}
