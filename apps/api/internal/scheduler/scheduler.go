package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/metrics"
	"github.com/leotime/leotime/apps/api/internal/notify"
	"github.com/leotime/leotime/apps/api/internal/outbox"
)

type Scheduler struct {
	cfg       config.Config
	notifier  *notify.StillRunningNotifier
	processor *outbox.Processor
}

func New(cfg config.Config, notifier *notify.StillRunningNotifier, processor *outbox.Processor) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		notifier:  notifier,
		processor: processor,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	scanTicker := time.NewTicker(s.cfg.SchedulerScanInterval)
	outboxTicker := time.NewTicker(s.cfg.OutboxProcessInterval)
	defer scanTicker.Stop()
	defer outboxTicker.Stop()

	s.runScan(ctx)
	s.runOutbox(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-scanTicker.C:
			s.runScan(ctx)
		case <-outboxTicker.C:
			s.runOutbox(ctx)
		}
	}
}

func (s *Scheduler) runScan(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	enqueued, err := s.notifier.EnqueueDue(ctx)
	if err != nil {
		metrics.SchedulerScanErrors.Inc()
		log.Printf("scheduler still-running scan failed: %v", err)
		return
	}
	if enqueued > 0 {
		metrics.SchedulerStillRunningDetected.Add(float64(enqueued))
		log.Printf("scheduler enqueued %d still-running timer notification(s)", enqueued)
	}
}

func (s *Scheduler) runOutbox(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	result, err := s.processor.ProcessOnce(ctx)
	if err != nil {
		metrics.SchedulerOutboxErrors.Inc()
		log.Printf("scheduler outbox processing failed: %v", err)
		return
	}

	if result.Sent > 0 {
		metrics.EmailOutboxSent.Add(float64(result.Sent))
	}
	if result.Retried > 0 {
		metrics.EmailOutboxRetried.Add(float64(result.Retried))
	}
	if result.Dead > 0 {
		metrics.EmailOutboxDead.Add(float64(result.Dead))
	}
}
