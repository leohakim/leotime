package seed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leotime/leotime/apps/api/internal/store"
)

var ErrAlreadySeeded = errors.New("database already has product data")

type Options struct {
	UserID string
	Force  bool
}

type Summary struct {
	Status           string         `json:"status"`
	Clients          int            `json:"clients"`
	Projects         int            `json:"projects"`
	Tasks            int            `json:"tasks"`
	Tags             int            `json:"tags"`
	TimeEntries      int            `json:"timeEntries"`
	OpenTimers       int            `json:"openTimers"`
	SkippedReason    string         `json:"skippedReason,omitempty"`
	ExistingOverview store.Overview `json:"existingOverview,omitempty"`
}

type Service struct {
	store *store.Store
	now   func() time.Time
}

func New(st *store.Store) *Service {
	return &Service{
		store: st,
		now:   time.Now,
	}
}

func (s *Service) Run(ctx context.Context, opts Options) (*Summary, error) {
	if opts.UserID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	overview, err := s.store.Overview(ctx, opts.UserID)
	if err != nil {
		return nil, err
	}

	if overview.ClientsTotal > 0 && !opts.Force {
		return &Summary{
			Status:           "skipped",
			SkippedReason:    "database already has clients; pass --force on an empty database or delete existing data first",
			ExistingOverview: overview,
		}, nil
	}
	if overview.ClientsTotal > 0 && opts.Force {
		return nil, fmt.Errorf("%w: remove existing clients first or use a fresh database", ErrAlreadySeeded)
	}

	acme, err := s.store.CreateClient(ctx, opts.UserID, store.ClientInput{
		Name:                   "ACME Corp",
		Email:                  "billing@acme.example",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 7500,
	})
	if err != nil {
		return nil, fmt.Errorf("create seed client acme: %w", err)
	}

	beta, err := s.store.CreateClient(ctx, opts.UserID, store.ClientInput{
		Name:                   "Beta Studio",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("create seed client beta: %w", err)
	}

	website, err := s.store.CreateProject(ctx, opts.UserID, store.ProjectInput{
		ClientID: acme.ID,
		Name:     "Website redesign",
		Color:    "#2563eb",
	})
	if err != nil {
		return nil, fmt.Errorf("create seed project website: %w", err)
	}

	apiProject, err := s.store.CreateProject(ctx, opts.UserID, store.ProjectInput{
		ClientID: acme.ID,
		Name:     "API migration",
		Color:    "#059669",
	})
	if err != nil {
		return nil, fmt.Errorf("create seed project api: %w", err)
	}

	internal, err := s.store.CreateProject(ctx, opts.UserID, store.ProjectInput{
		ClientID: beta.ID,
		Name:     "Internal ops",
		Color:    "#64748b",
	})
	if err != nil {
		return nil, fmt.Errorf("create seed project internal: %w", err)
	}

	landingTask, err := s.store.CreateTask(ctx, opts.UserID, store.TaskInput{
		ProjectID: website.ID,
		Name:      "Landing page",
		Billable:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("create seed task landing: %w", err)
	}

	dashboardTask, err := s.store.CreateTask(ctx, opts.UserID, store.TaskInput{
		ProjectID: website.ID,
		Name:      "Dashboard UI",
		Billable:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("create seed task dashboard: %w", err)
	}

	importTask, err := s.store.CreateTask(ctx, opts.UserID, store.TaskInput{
		ProjectID: apiProject.ID,
		Name:      "Solidtime import",
		Billable:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("create seed task import: %w", err)
	}

	adminTask, err := s.store.CreateTask(ctx, opts.UserID, store.TaskInput{
		ProjectID: internal.ID,
		Name:      "Weekly planning",
		Billable:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("create seed task admin: %w", err)
	}

	deepWork, err := s.store.CreateTag(ctx, opts.UserID, store.TagInput{Name: "Deep work", Color: "#7c3aed"})
	if err != nil {
		return nil, fmt.Errorf("create seed tag deep work: %w", err)
	}
	meeting, err := s.store.CreateTag(ctx, opts.UserID, store.TagInput{Name: "Meeting", Color: "#ea580c"})
	if err != nil {
		return nil, fmt.Errorf("create seed tag meeting: %w", err)
	}
	adminTag, err := s.store.CreateTag(ctx, opts.UserID, store.TagInput{Name: "Admin", Color: "#64748b"})
	if err != nil {
		return nil, fmt.Errorf("create seed tag admin: %w", err)
	}

	now := s.now().UTC()
	for dayOffset := 13; dayOffset >= 0; dayOffset-- {
		day := truncateDay(now.AddDate(0, 0, -dayOffset))
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}

		slots := []struct {
			startHour int
			duration  time.Duration
			projectID string
			taskID    string
			clientID  string
			desc      string
			billable  bool
			tagIDs    []string
		}{
			{9, 90 * time.Minute, website.ID, landingTask.ID, acme.ID, "Layout polish", true, []string{deepWork.ID}},
			{11, 60 * time.Minute, apiProject.ID, importTask.ID, acme.ID, "Import mapping review", true, []string{deepWork.ID}},
			{14, 45 * time.Minute, internal.ID, adminTask.ID, beta.ID, "Planning and inbox", false, []string{adminTag.ID}},
		}
		if dayOffset%3 == 0 {
			slots = append(slots, struct {
				startHour int
				duration  time.Duration
				projectID string
				taskID    string
				clientID  string
				desc      string
				billable  bool
				tagIDs    []string
			}{16, 30 * time.Minute, website.ID, dashboardTask.ID, acme.ID, "Client sync", true, []string{meeting.ID}})
		}

		for _, slot := range slots {
			startedAt := day.Add(time.Duration(slot.startHour) * time.Hour)
			endedAt := startedAt.Add(slot.duration)
			if _, err := s.store.CreateTimeEntry(ctx, opts.UserID, store.TimeEntryInput{
				ClientID:    slot.clientID,
				ProjectID:   slot.projectID,
				TaskID:      slot.taskID,
				TagIDs:      slot.tagIDs,
				Description: slot.desc,
				StartedAt:   startedAt.Format(time.RFC3339),
				EndedAt:     endedAt.Format(time.RFC3339),
				Billable:    slot.billable,
			}); err != nil {
				return nil, fmt.Errorf("create seed time entry: %w", err)
			}
		}
	}

	if _, err := s.store.StartTimer(ctx, opts.UserID, store.TimerStartInput{
		ClientID:    acme.ID,
		ProjectID:   website.ID,
		TaskID:      dashboardTask.ID,
		TagIDs:      []string{deepWork.ID},
		Description: "Timer demo entry",
		Billable:    true,
	}); err != nil {
		return nil, fmt.Errorf("create seed open timer: %w", err)
	}

	finalOverview, err := s.store.Overview(ctx, opts.UserID)
	if err != nil {
		return nil, err
	}

	return &Summary{
		Status:      "seeded",
		Clients:     finalOverview.ClientsTotal,
		Projects:    finalOverview.ProjectsTotal,
		Tasks:       finalOverview.TasksTotal,
		Tags:        finalOverview.TagsTotal,
		TimeEntries: finalOverview.TimeEntriesTotal,
		OpenTimers:  finalOverview.OpenTimers,
	}, nil
}

func truncateDay(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
