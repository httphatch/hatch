package config

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

const debounceDuration = 500 * time.Millisecond

// Watcher monitors the config file and project hatch.yml files for changes,
// calling a callback with the newly loaded config when a valid change is detected.
type Watcher struct {
	watcher     *fsnotify.Watcher
	callback    func(Config)
	done        chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
	projectDirs map[string]struct{}
	closed      atomic.Bool
}

// NewWatcher creates a Watcher that calls cb whenever the config file or any
// linked project's hatch.yml changes and the new config is valid.
func NewWatcher(initialCfg Config, cb func(Config)) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the directory (not the file) to handle vim-style delete+recreate saves.
	dir := ConfigFileDir()
	if err := fw.Add(dir); err != nil {
		_ = fw.Close()
		return nil, err
	}

	w := &Watcher{
		watcher:     fw,
		callback:    cb,
		done:        make(chan struct{}),
		projectDirs: make(map[string]struct{}),
	}

	w.UpdateProjectWatches(initialCfg)

	w.wg.Add(1)
	go w.loop()

	return w, nil
}

// UpdateProjectWatches diffs the desired project paths against currently watched
// paths, adding and removing fsnotify watches as needed.
func (w *Watcher) UpdateProjectWatches(cfg Config) {
	w.mu.Lock()
	defer w.mu.Unlock()

	desired := make(map[string]struct{})
	for _, proj := range cfg.Projects {
		if proj.Path != "" {
			desired[proj.Path] = struct{}{}
		}
	}

	// Remove watches for projects no longer in config.
	for dir := range w.projectDirs {
		if _, ok := desired[dir]; !ok {
			if err := w.watcher.Remove(dir); err != nil {
				log.Debug().Err(err).Str("dir", dir).Msg("failed to remove project watch")
			}
			delete(w.projectDirs, dir)
		}
	}

	// Add watches for new projects.
	for dir := range desired {
		if _, ok := w.projectDirs[dir]; !ok {
			if err := w.watcher.Add(dir); err != nil {
				log.Debug().Err(err).Str("dir", dir).Msg("failed to watch project directory")
				continue
			}
			w.projectDirs[dir] = struct{}{}
		}
	}
}

func (w *Watcher) isWatchedFile(name string) bool {
	base := filepath.Base(name)
	if base == filepath.Base(ConfigFile()) {
		return true
	}
	if base == "hatch.yml" {
		w.mu.Lock()
		_, ok := w.projectDirs[filepath.Dir(name)]
		w.mu.Unlock()
		return ok
	}
	return false
}

func (w *Watcher) loop() {
	defer w.wg.Done()

	var timer *time.Timer

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if !w.isWatchedFile(event.Name) {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Reset debounce timer
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDuration, func() {
				if w.closed.Load() {
					return
				}
				cfg, err := LoadWithProjectConfigs()
				if err != nil {
					var ve *ValidationErrors
					if errors.As(err, &ve) {
						log.Warn().Int("count", len(ve.Errs)).Msg("config reload skipped due to validation errors")
						for _, e := range ve.Errs {
							log.Warn().Msgf("  - %s", e)
						}
					} else {
						log.Warn().Err(err).Msg("config reload skipped")
					}
					return
				}
				log.Info().Msg("config reloaded")
				w.UpdateProjectWatches(cfg)
				w.callback(cfg)
			})

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("config watcher error")

		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return
		}
	}
}

// Close stops the watcher and waits for the loop to exit.
func (w *Watcher) Close() error {
	w.closed.Store(true)
	close(w.done)
	err := w.watcher.Close()
	w.wg.Wait()
	return err
}
