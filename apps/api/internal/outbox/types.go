package outbox

const (
	KindTimerStillRunning = "timer_still_running"
	KindPasswordReset     = "password_reset"
	KindBackupSuccess     = "backup_success"
	KindBackupFailure     = "backup_failure"
	KindRestoreSuccess    = "restore_success"
	KindRestoreFailure    = "restore_failure"

	StatusPending = "pending"
	StatusSent    = "sent"
	StatusDead    = "dead"
)

type Email struct {
	ID          string
	UserID      string
	TimeEntryID string
	Kind        string
	ToAddress   string
	Subject     string
	BodyText    string
	Status      string
	Attempts    int
	MaxAttempts int
	NextRetryAt string
	LastError   string
	SentAt      string
	CreatedAt   string
	UpdatedAt   string
}

type EnqueueInput struct {
	UserID      string
	TimeEntryID string
	Kind        string
	ToAddress   string
	Subject     string
	BodyText    string
	MaxAttempts int
}
