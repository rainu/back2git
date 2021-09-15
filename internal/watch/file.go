package watch

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"sync"
	"time"
)

type callbacks struct {
	onChange func(name string)
	onDelete func(name string)
}

type fileWatcher struct {
	watcher       *fsnotify.Watcher
	callbacks     map[string]callbacks
	callbackMutex sync.RWMutex

	creationWatchCtx      context.Context
	creationWatchCancelFn context.CancelFunc
	creationWatch         map[string]callbacks
	creationWatchMutex    sync.RWMutex
}

func NewFileWatcher() (*fileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("unable to establish file watcher: %w", err)
	}

	result := &fileWatcher{
		watcher:            watcher,
		callbacks:          map[string]callbacks{},
		callbackMutex:      sync.RWMutex{},
		creationWatch:      map[string]callbacks{},
		creationWatchMutex: sync.RWMutex{},
	}

	go result.startWatch()

	result.creationWatchCtx, result.creationWatchCancelFn = context.WithCancel(context.Background())
	go result.startWatchCreation()

	return result, nil
}

func (f *fileWatcher) startWatch() {
	for {
		select {
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			f.callbackMutex.RLock()
			callbacks, isWatched := f.callbacks[event.Name]
			f.callbackMutex.RUnlock()

			if isWatched {
				if isTypeOf(&event, fsnotify.Write) {
					callbacks.onChange(event.Name)
				} else if isTypeOf(&event, fsnotify.Remove) || isTypeOf(&event, fsnotify.Rename) {
					callbacks.onDelete(event.Name)

					//rewatch file -> otherwise we will not be notified
					err := f.Unwatch(event.Name)
					if err != nil && strings.HasSuffix(err.Error(), "can't remove non-existent inotify watch for:") {
						log.WithError(err).WithField("path", event.Name).Warn("Unable to unwatch file.")
					}
					f.watchCreation(event.Name, callbacks.onChange, callbacks.onDelete)
				}
			}
		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			log.WithError(err).Error("Watcher error occurred!")
		}
	}
}

func isTypeOf(event *fsnotify.Event, op fsnotify.Op) bool {
	return event.Op&op == op
}

func (f *fileWatcher) startWatchCreation() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-f.creationWatchCtx.Done():
			return
		case <-ticker.C:
			f.creationWatchMutex.RLock()
			for path, callbacks := range f.creationWatch {
				go f.checkForCreation(path, callbacks.onChange, callbacks.onDelete)
			}
			f.creationWatchMutex.RUnlock()
		}
	}
}

func (f *fileWatcher) checkForCreation(path string, onChange, onDelete func(name string)) {
	if fileExists(path) {
		err := f.watchChange(path, onChange, onDelete)
		if err != nil {
			log.WithError(err).WithField("path", path).Error("Unable to watch file!")
		}
		onChange(path)

		f.creationWatchMutex.Lock()
		defer f.creationWatchMutex.Unlock()

		delete(f.creationWatch, path)
	}
}

func (f *fileWatcher) Watch(path string, onChange, onDelete func(name string)) error {
	if !fileExists(path) {
		//watch for creation
		f.watchCreation(path, onChange, onDelete)
		return nil
	}

	return f.watchChange(path, onChange, onDelete)
}

func (f *fileWatcher) watchChange(path string, onChange, onDelete func(name string)) error {
	f.callbackMutex.Lock()
	defer f.callbackMutex.Unlock()

	f.callbacks[path] = callbacks{
		onChange: onChange,
		onDelete: onDelete,
	}

	return f.watcher.Add(path)
}

func (f *fileWatcher) watchCreation(path string, onChange, onDelete func(name string)) {
	f.creationWatchMutex.Lock()
	defer f.creationWatchMutex.Unlock()

	f.creationWatch[path] = callbacks{
		onChange: onChange,
		onDelete: onDelete,
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func (f *fileWatcher) Unwatch(path string) error {
	f.callbackMutex.Lock()
	defer f.callbackMutex.Unlock()

	delete(f.callbacks, path)
	return f.watcher.Remove(path)
}

func (f *fileWatcher) Close() error {
	//stop creation watch
	f.creationWatchCancelFn()

	//stop fsnotify
	return f.watcher.Close()
}
