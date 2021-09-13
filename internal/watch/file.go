package watch

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"sync"
)

type callbacks struct {
	onChange func(name string)
	onDelete func(name string)
}

type fileWatcher struct {
	watcher       *fsnotify.Watcher
	callbacks     map[string]callbacks
	callbackMutex sync.RWMutex
}

func NewFileWatcher() (*fileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("unable to establish file watcher: %w", err)
	}

	result := &fileWatcher{
		watcher:       watcher,
		callbacks:     map[string]callbacks{},
		callbackMutex: sync.RWMutex{},
	}

	go result.start()

	return result, nil
}

func (f *fileWatcher) start() {
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
				if event.Op&fsnotify.Write == fsnotify.Write {
					callbacks.onChange(event.Name)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					callbacks.onDelete(event.Name)
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					callbacks.onDelete(event.Name)
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

func (f *fileWatcher) Watch(path string, onChange, onDelete func(name string)) error {
	f.callbackMutex.Lock()
	defer f.callbackMutex.Unlock()

	f.callbacks[path] = callbacks{
		onChange: onChange,
		onDelete: onDelete,
	}

	return f.watcher.Add(path)
}

func (f *fileWatcher) Unwatch(path string) error {
	f.callbackMutex.Lock()
	defer f.callbackMutex.Unlock()

	delete(f.callbacks, path)
	return f.watcher.Remove(path)
}
