package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/leotime/leotime/apps/api/internal/backup"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/maintenance"
	"github.com/leotime/leotime/apps/api/internal/metrics"
	"github.com/leotime/leotime/apps/api/internal/notify"
	"github.com/leotime/leotime/apps/api/internal/outbox"
)

type Scheduler struct {
	cfg       config.Config
	notifier  *notify.StillRunningNotifier
	processor *outbox.Processor
	backups   *backup.Service
}

func New(cfg config.Config, notifier *notify.StillRunningNotifier, processor *outbox.Processor, backups *backup.Service) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		notifier:  notifier,
		processor: processor,
		backups:   backups,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	scanTicker := time.NewTicker(s.cfg.SchedulerScanInterval)
	outboxTicker := time.NewTicker(s.cfg.OutboxProcessInterval)
	backupTicker := time.NewTicker(s.cfg.BackupSchedulerInterval)
	defer scanTicker.Stop()
	defer outboxTicker.Stop()
	defer backupTicker.Stop()

	if s.cfg.SchedulerEnabled {
		s.runScan(ctx)
	}
	s.runOutbox(ctx)
	if s.cfg.BackupSchedulerEnabled {
		s.runBackup(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-scanTicker.C:
			if s.cfg.SchedulerEnabled {
				s.runScan(ctx)
			}
		case <-outboxTicker.C:
			s.runOutbox(ctx)
		case <-backupTicker.C:
			if s.cfg.BackupSchedulerEnabled {
				s.runBackup(ctx)
			}
		}
	}
}

func (s *Scheduler) runScan(ctx context.Context) {
	if ctx.Err() != nil || maintenance.Enabled() {
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
	if ctx.Err() != nil || maintenance.Enabled() {
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

func (s *Scheduler) runBackup(ctx context.Context) {
	if ctx.Err() != nil || s.backups == nil || maintenance.Enabled() {
		return
	}

	if err := s.backups.RunScheduled(ctx); err != nil {
		log.Printf("scheduler backup run failed: %v", err)
	}
}
