package processor

import (
	"os"
)

type Storage interface {
	Save(file *os.File) error
	Delete(filePath string) error
	Sync() error
}

type FileWatcher interface {
	Watch(path string, onChange, onDelete func(name string)) error
	Unwatch(path string) error
	Close() error
}
