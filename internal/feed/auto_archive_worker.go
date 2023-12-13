package feed

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	autoArchiveCheckDelay = time.Hour
)

type AutoArchiveWorker struct {
	service *Service
}

func NewAutoArchiveWorker(s *Service) *AutoArchiveWorker {
	return &AutoArchiveWorker{
		service: s,
	}
}

func (w *AutoArchiveWorker) Start(ctx context.Context) error {
	for {
		start := time.Now()
		err := w.service.markExpiredAsAutoArchived(ctx)
		if err != nil {
			log.Error().Err(err).Msg("auto archive feed items")
		}

		log.Debug().Msgf("auto archive feed items completed: %v", time.Since(start))

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(autoArchiveCheckDelay):
		}
	}
}
