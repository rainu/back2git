package processor

import (
	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"os"
	"path"
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

	file, _ := os.Open(filePath)
	if file != nil {
		defer file.Close()

		err = p.storage.Save(file)
		if err != nil {
			log.WithError(err).WithField("path", filePath).Error("Unable to save file!")
		}
	}
}

func (p *processor) onChange(filePath string) {
	iLog := log.WithField("path", filePath)
	iLog.Info("File change detected.")

	file, err := os.Open(filePath)
	if err != nil {
		iLog.WithError(err).Error("Unable to open file!")
		return
	}
	defer file.Close()

	err = p.storage.Save(file)
	if err != nil {
		iLog.WithError(err).Error("Unable to save file!")
		return
	}
}

func (p *processor) onDelete(filePath string) {
	iLog := log.WithField("path", filePath)
	iLog.Info("File deletion detected.")

	err := p.storage.Delete(filePath)
	if err != nil {
		iLog.WithError(err).Error("Unable to save file!")
		return
	}
}

func (p *processor) Close() error {
	log.Info("Synchronize storage...")
	syncErr := p.storage.Sync()
	if syncErr == nil {
		log.Info("Synchronization finished successfully.")
	}

	watchErr := p.watcher.Close()

	return multierr.Combine(syncErr, watchErr)
}
