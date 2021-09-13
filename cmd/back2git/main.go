package main

import (
	"context"
	"github.com/rainu/back2git/internal/git"
	"github.com/rainu/back2git/internal/processor"
	internalSync "github.com/rainu/back2git/internal/sync"
	"github.com/rainu/back2git/internal/watch"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = time.RFC3339
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
}

func main() {
	config := LoadConfig()

	// Catch signals to enable graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancelFn := context.WithCancel(context.Background())

	fw, err := watch.NewFileWatcher()
	if err != nil {
		log.WithError(err).Fatal("Unable to establish file watcher!")
	}

	auth, err := config.Authentication()
	if err != nil {
		log.WithError(err).Fatal("Unable to extract authentication information from configuration!")
	}

	s, err := git.NewGitStore(config.Repository.Url, config.Repository.Branch, config.Repository.Path, auth)
	if err != nil {
		log.WithError(err).Fatal("Unable to initialise git repository!")
	}

	p := processor.NewProcessor(fw, s)

	for filePath, _ := range config.Files {
		log.WithField("path", filePath).Info("Overwatch file.")
		p.Manage(filePath)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		pusher := internalSync.NewPusher(config.Repository.PushInterval, s)
		pusher.Start(ctx)
	}()

	select {
	case <-sigs:
	}

	log.Info("Initialise shutdown of application.")
	cancelFn()

	err = p.Close()
	if err != nil {
		log.WithError(err).Fatal("Unable to shutdown correctly!")
	}

	if !waitTimeout(&wg) {
		log.Fatal("Unable to shutdown correctly!")
	}
}

func waitTimeout(wg *sync.WaitGroup) bool {
	wgChan := make(chan interface{})
	go func() {
		defer close(wgChan)
		wg.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		return false
	case <-wgChan:
		return true
	}
}
