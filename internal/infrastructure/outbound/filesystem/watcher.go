package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
)

// Watcher watches for YAML file changes and triggers a reload callback.
type Watcher struct {
	rootDir  string
	debounce time.Duration
	logger   ports.Logger
	watcher  *fsnotify.Watcher
	onReload func()
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewWatcher creates a file watcher for the given directory.
func NewWatcher(rootDir string, debounce time.Duration, logger ports.Logger, onReload func()) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		rootDir:  rootDir,
		debounce: debounce,
		logger:   logger,
		watcher:  fsWatcher,
		onReload: onReload,
		done:     make(chan struct{}),
	}

	if err := w.addRecursive(rootDir); err != nil {
		_ = fsWatcher.Close()
		return nil, err
	}

	return w, nil
}

// Start begins watching for file changes in a goroutine.
func (w *Watcher) Start() {
	w.wg.Add(1)
	go w.loop()
}

// Stop terminates the watcher.
func (w *Watcher) Stop() {
	close(w.done)
	_ = w.watcher.Close()
	w.wg.Wait()
}

func (w *Watcher) loop() {
	defer w.wg.Done()

	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about YAML files.
			if !isYAMLFile(event.Name) {
				// Check if a new directory was created.
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = w.addRecursive(event.Name)
					}
				}
				continue
			}

			w.logger.Debug("file change detected", "file", event.Name, "op", event.Op.String())

			// Debounce: reset timer on each event.
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(w.debounce)
			timerC = timer.C

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("watcher error", "error", err)

		case <-timerC:
			w.logger.Info("reloading scenarios due to file changes")
			w.onReload()
			timerC = nil
		}
	}
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	})
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}
