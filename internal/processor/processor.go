package processor

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

type processor struct {
	storage Storage
	watcher FileWatcher
}

func NewProcessor(watcher FileWatcher, storage Storage) *processor {
	return &processor{
		watcher: watcher,
		storage: storage,
	}
}

func (p *processor) Manage(filePath string) {
	if !path.IsAbs(filePath) {
		log.WithField("path", filePath).Warn("Only absolute path are supported!")
		return
	}

	err := p.watcher.Watch(filePath, p.onChange, p.onDelete)
	if err != nil {
		log.WithError(err).WithField("path", filePath).Error("Unable to watch file!")
	}
}

func (p *processor) onChange(filePath string) {
	iLog := log.WithField("path", filePath)
	iLog.Info("File change detected.")

	file, err := os.Open(filePath)
	if err != nil {
		iLog.WithError(err).Error("Unable to read file content!")
		return
	}
	defer file.Close()

	err = p.storage.Save(strings.TrimLeft(filePath, "/"), file)
	if err != nil {
		iLog.WithError(err).Error("Unable to save file!")
		return
	}
}

func (p *processor) onDelete(filePath string) {
	iLog := log.WithField("path", filePath)
	iLog.Info("File deletion detected.")

	err := p.storage.Save(strings.TrimLeft(filePath, "/"), strings.NewReader(""))
	if err != nil {
		iLog.WithError(err).Error("Unable to save file!")
		return
	}
}

func (p *processor) Close() error {
	log.Info("Synchronize storage...")
	err := p.storage.Sync()
	if err == nil {
		log.Info("Synchronization finished successfully.")
	}

	return err
}
