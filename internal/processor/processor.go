package processor

import (
	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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
	iLog := log.WithField("path", filePath)
	if !path.IsAbs(filePath) {
		iLog.Warn("Only absolute path are supported!")
		return
	}

	//if the path is a directory
	if fi, _ := os.Stat(filePath); fi != nil && fi.IsDir() {
		iLog.Info("Scan directory...")
		filepath.Walk(filePath, func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				p.Manage(path)
			}
			return nil
		})
		return
	}

	iLog.Info("Overwatch file.")
	err := p.watcher.Watch(filePath, p.onChange, p.onDelete)
	if err != nil {
		iLog.WithError(err).Error("Unable to watch file!")
	}

	file, _ := os.Open(filePath)
	if file != nil {
		defer file.Close()

		err = p.storage.Save(file)
		if err != nil {
			iLog.WithError(err).Error("Unable to save file!")
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
