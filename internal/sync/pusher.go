package sync

import (
	"context"
	log "github.com/sirupsen/logrus"
	"time"
)

type pusher struct {
	ticker *time.Ticker
	syncer Syncer
}

func NewPusher(interval time.Duration, syncer Syncer) *pusher {
	result := &pusher{
		syncer: syncer,
	}

	if interval > 0 {
		result.ticker = time.NewTicker(interval)
	}

	return result
}

func (p *pusher) Start(ctx context.Context) {
	if p.ticker == nil {
		return
	}

	for {
		select {
		case <-p.ticker.C:
		case <-ctx.Done():
			return
		}

		if err := p.syncer.Sync(); err != nil {
			log.WithError(err).Error("Error while syncing.")
		}
	}
}
