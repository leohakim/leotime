package backup

import (
	"errors"
	"time"
)

var ErrBusy = errors.New("backup job already running")

// ErrRemoteStorage marks failures talking to remote backup storage (S3-compatible).
// HTTP handlers should not expose the wrapped cause to clients.
var ErrRemoteStorage = errors.New("remote storage operation failed")

func IsDue(settings EnabledSettings, timezone string, now time.Time, force bool) (bool, error) {
	if !settings.Enabled {
		return false, nil
	}
	if force {
		return true, nil
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}

	localNow := now.In(location)
	if localNow.Hour() < settings.ScheduleHour {
		return false, nil
	}

	if settings.LastRunAt == nil || *settings.LastRunAt == "" {
		return true, nil
	}

	lastRun, err := time.Parse(time.RFC3339, *settings.LastRunAt)
	if err != nil {
		return true, nil
	}

	lastLocal := lastRun.In(location)
	if lastLocal.Year() == localNow.Year() && lastLocal.YearDay() == localNow.YearDay() && settings.LastStatus == "success" {
		return false, nil
	}

	return true, nil
}

type EnabledSettings struct {
	Enabled      bool
	ScheduleHour int
	LastRunAt    *string
	LastStatus   string
}
