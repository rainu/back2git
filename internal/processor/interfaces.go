package processor

import "io"

type Storage interface {
	Save(name string, content io.Reader) error
	Sync() error
}

type FileWatcher interface {
	Watch(path string, onChange, onDelete func(name string)) error
	Unwatch(path string) error
}
