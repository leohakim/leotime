package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	SchedulerStillRunningDetected = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_still_running_timers_detected_total",
		Help: "Total still-running timer notifications enqueued.",
	})
	SchedulerScanErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_scheduler_scan_errors_total",
		Help: "Total scheduler scan failures.",
	})
	SchedulerOutboxErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_scheduler_outbox_errors_total",
		Help: "Total outbox processing failures.",
	})
	EmailOutboxSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_email_outbox_sent_total",
		Help: "Total outbox emails sent successfully.",
	})
	EmailOutboxRetried = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_email_outbox_retried_total",
		Help: "Total outbox emails scheduled for retry.",
	})
	EmailOutboxDead = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_email_outbox_dead_total",
		Help: "Total outbox emails marked dead.",
	})
	BackupLastSuccessTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "leotime_backup_last_success_timestamp",
		Help: "Unix timestamp of the last successful S3 backup.",
	})
	BackupFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_backup_failures_total",
		Help: "Total failed S3 backup runs.",
	})
	BackupDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "leotime_backup_duration_seconds",
		Help:    "Duration of S3 backup runs in seconds.",
		Buckets: prometheus.DefBuckets,
	})
	BackupRestoreSuccessTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_backup_restore_success_total",
		Help: "Total successful S3 restore operations.",
	})
	BackupRestoreFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "leotime_backup_restore_failures_total",
		Help: "Total failed S3 restore operations.",
	})
)

func NewBackupTimer() *prometheus.Timer {
	return prometheus.NewTimer(BackupDurationSeconds)
}

func init() {
	prometheus.MustRegister(
		SchedulerStillRunningDetected,
		SchedulerScanErrors,
		SchedulerOutboxErrors,
		EmailOutboxSent,
		EmailOutboxRetried,
		EmailOutboxDead,
		BackupLastSuccessTimestamp,
		BackupFailuresTotal,
		BackupDurationSeconds,
		BackupRestoreSuccessTotal,
		BackupRestoreFailuresTotal,
	)
}
