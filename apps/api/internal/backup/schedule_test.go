package backup

import (
	"testing"
	"time"
)

func TestIsDueDefaultHour(t *testing.T) {
	lastRun := "2026-07-06T00:30:00Z"
	settings := EnabledSettings{
		Enabled:      true,
		ScheduleHour: 1,
		LastRunAt:    &lastRun,
		LastStatus:   "success",
	}

	now := time.Date(2026, 7, 6, 0, 30, 0, 0, time.UTC)
	due, err := IsDue(settings, "UTC", now, false)
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("expected not due before schedule hour")
	}

	now = time.Date(2026, 7, 6, 1, 5, 0, 0, time.UTC)
	due, err = IsDue(settings, "UTC", now, false)
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("expected not due after successful run same day")
	}

	now = time.Date(2026, 7, 7, 1, 5, 0, 0, time.UTC)
	due, err = IsDue(settings, "UTC", now, false)
	if err != nil {
		t.Fatal(err)
	}
	if !due {
		t.Fatal("expected due on next day")
	}
}

func TestIsDueForce(t *testing.T) {
	settings := EnabledSettings{Enabled: true, ScheduleHour: 1}
	due, err := IsDue(settings, "UTC", time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !due {
		t.Fatal("expected force due")
	}
}
