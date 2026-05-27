package sync

import (
	"fmt"
	"log/slog"
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
	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return fmt.Errorf("watch %q: %w", path, err)
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
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				w.logger.Debug("watchdog: file changed, debouncing", "path", event.Name)
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(watchdogDebounce, func() {
					if _, err := w.dispatcher.Submit("watchdog"); err != nil && err != ErrSyncBusy {
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
