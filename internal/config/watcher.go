package config

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

const debounceDuration = 500 * time.Millisecond

// Watcher monitors the config file for changes and calls a callback
// with the newly loaded config when a valid change is detected.
type Watcher struct {
	watcher  *fsnotify.Watcher
	callback func(Config)
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewWatcher creates a Watcher that calls cb whenever the config file
// changes and the new config is valid. Invalid configs are logged and skipped.
func NewWatcher(cb func(Config)) (*Watcher, error) {
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
		watcher:  fw,
		callback: cb,
		done:     make(chan struct{}),
	}

	w.wg.Add(1)
	go w.loop()

	return w, nil
}

func (w *Watcher) loop() {
	defer w.wg.Done()

	var timer *time.Timer
	configBase := filepath.Base(ConfigFile())

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != configBase {
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
				cfg, err := Load()
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
	close(w.done)
	err := w.watcher.Close()
	w.wg.Wait()
	return err
}
