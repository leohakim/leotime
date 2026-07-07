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
)

func init() {
	prometheus.MustRegister(
		SchedulerStillRunningDetected,
		SchedulerScanErrors,
		SchedulerOutboxErrors,
		EmailOutboxSent,
		EmailOutboxRetried,
		EmailOutboxDead,
	)
}
