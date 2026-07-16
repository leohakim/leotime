package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/leotime/leotime/apps/api/internal/store"
)

const seedHistoryMonths = 6

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
	Invoices         int            `json:"invoices"`
	OpenTimers       int            `json:"openTimers"`
	SkippedReason    string         `json:"skippedReason,omitempty"`
	ExistingOverview store.Overview `json:"existingOverview,omitempty"`
}

type Service struct {
	store *store.Store
	now   func() time.Time
}

type seedClient struct {
	record   *store.Client
	projects []seedProject
}

type seedProject struct {
	record *store.Project
	tasks  []seedTask
}

type seedTask struct {
	record       *store.Task
	descriptions []string
}

type seedTag struct {
	record *store.Tag
}

type timeSlot struct {
	startHour int
	duration  time.Duration
	project   *seedProject
	task      *seedTask
	descIndex int
	billable  bool
	tagIDs    func(tags map[string]seedTag) []string
}

func New(st *store.Store) *Service {
	return &Service{
		store: st,
		now:   time.Now,
	}
}

func NewWithNow(st *store.Store, now func() time.Time) *Service {
	service := New(st)
	service.now = now
	return service
}

func (s *Service) Run(ctx context.Context, opts Options) (*Summary, error) {
	if opts.UserID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	overview, err := s.store.Overview(ctx, opts.UserID)
	if err != nil {
		return nil, err
	}

	if overview.ClientsTotal > 0 {
		if !opts.Force {
			return &Summary{
				Status:           "skipped",
				SkippedReason:    "database already has clients; run make reset-data first or pass --force to wipe and reseed",
				ExistingOverview: overview,
			}, nil
		}
		if _, err := s.store.ClearUserData(ctx, opts.UserID); err != nil {
			return nil, fmt.Errorf("clear existing data before reseed: %w", err)
		}
	}

	clients, tags, err := s.createCatalog(ctx, opts.UserID)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	if err := s.createHistoricalEntries(ctx, opts.UserID, clients, tags, now); err != nil {
		return nil, err
	}

	if err := s.createInvoiceDrafts(ctx, opts.UserID, clients, now); err != nil {
		return nil, err
	}

	if err := s.createOpenTimer(ctx, opts.UserID, clients, tags); err != nil {
		return nil, err
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
		Invoices:    finalOverview.InvoicesTotal,
		OpenTimers:  finalOverview.OpenTimers,
	}, nil
}

func (s *Service) createCatalog(ctx context.Context, userID string) ([]seedClient, map[string]seedTag, error) {
	rate90 := int64(9000)

	acme, err := s.store.CreateClient(ctx, userID, store.ClientInput{
		Name:                   "ACME Corp",
		Email:                  "billing@acme.example",
		TaxID:                  "B12345678",
		BillingAddress:         "Calle Mayor 12, Madrid",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 7500,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed client acme: %w", err)
	}

	northwind, err := s.store.CreateClient(ctx, userID, store.ClientInput{
		Name:                   "Northwind Labs",
		Email:                  "ap@northwind.example",
		TaxID:                  "US-998877",
		BillingAddress:         "500 Market St, San Francisco",
		DefaultCurrency:        "USD",
		DefaultHourlyRateMinor: 12000,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed client northwind: %w", err)
	}

	beta, err := s.store.CreateClient(ctx, userID, store.ClientInput{
		Name:                   "Beta Studio",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 0,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed client beta: %w", err)
	}

	legacy, err := s.store.CreateClient(ctx, userID, store.ClientInput{
		Name:                   "Legacy Client",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 5000,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed client legacy: %w", err)
	}

	website, err := s.store.CreateProject(ctx, userID, store.ProjectInput{
		ClientID: acme.ID,
		Name:     "Website redesign",
		Color:    "#2563eb",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed project website: %w", err)
	}

	apiProject, err := s.store.CreateProject(ctx, userID, store.ProjectInput{
		ClientID: acme.ID,
		Name:     "API migration",
		Color:    "#059669",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed project api: %w", err)
	}

	mobile, err := s.store.CreateProject(ctx, userID, store.ProjectInput{
		ClientID:               acme.ID,
		Name:                   "Mobile companion",
		Color:                  "#7c3aed",
		DefaultHourlyRateMinor: &rate90,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed project mobile: %w", err)
	}

	dataPipeline, err := s.store.CreateProject(ctx, userID, store.ProjectInput{
		ClientID: northwind.ID,
		Name:     "Data pipeline",
		Color:    "#dc2626",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed project data pipeline: %w", err)
	}

	internal, err := s.store.CreateProject(ctx, userID, store.ProjectInput{
		ClientID: beta.ID,
		Name:     "Internal ops",
		Color:    "#64748b",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed project internal: %w", err)
	}

	oldProject, err := s.store.CreateProject(ctx, userID, store.ProjectInput{
		ClientID: legacy.ID,
		Name:     "Sunset rollout",
		Color:    "#94a3b8",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed project legacy: %w", err)
	}

	landingTask, err := s.store.CreateTask(ctx, userID, store.TaskInput{
		ProjectID: website.ID,
		Name:      "Landing page",
		Billable:  true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed task landing: %w", err)
	}

	dashboardTask, err := s.store.CreateTask(ctx, userID, store.TaskInput{
		ProjectID: website.ID,
		Name:      "Dashboard UI",
		Billable:  true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed task dashboard: %w", err)
	}

	importTask, err := s.store.CreateTask(ctx, userID, store.TaskInput{
		ProjectID: apiProject.ID,
		Name:      "Solidtime import",
		Billable:  true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed task import: %w", err)
	}

	mobileTask, err := s.store.CreateTask(ctx, userID, store.TaskInput{
		ProjectID: mobile.ID,
		Name:      "Push notifications",
		Billable:  true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed task mobile: %w", err)
	}

	etlTask, err := s.store.CreateTask(ctx, userID, store.TaskInput{
		ProjectID: dataPipeline.ID,
		Name:      "ETL jobs",
		Billable:  true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed task etl: %w", err)
	}

	adminTask, err := s.store.CreateTask(ctx, userID, store.TaskInput{
		ProjectID: internal.ID,
		Name:      "Weekly planning",
		Billable:  false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed task admin: %w", err)
	}

	legacyTask, err := s.store.CreateTask(ctx, userID, store.TaskInput{
		ProjectID: oldProject.ID,
		Name:      "Maintenance window",
		Billable:  true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed task legacy: %w", err)
	}

	deepWork, err := s.store.CreateTag(ctx, userID, store.TagInput{Name: "Deep work", Color: "#7c3aed"})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed tag deep work: %w", err)
	}
	meeting, err := s.store.CreateTag(ctx, userID, store.TagInput{Name: "Meeting", Color: "#ea580c"})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed tag meeting: %w", err)
	}
	adminTag, err := s.store.CreateTag(ctx, userID, store.TagInput{Name: "Admin", Color: "#64748b"})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed tag admin: %w", err)
	}
	review, err := s.store.CreateTag(ctx, userID, store.TagInput{Name: "Review", Color: "#0891b2"})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed tag review: %w", err)
	}
	bugfix, err := s.store.CreateTag(ctx, userID, store.TagInput{Name: "Bugfix", Color: "#be123c"})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed tag bugfix: %w", err)
	}
	deprecatedTag, err := s.store.CreateTag(ctx, userID, store.TagInput{Name: "Deprecated", Color: "#a8a29e"})
	if err != nil {
		return nil, nil, fmt.Errorf("create seed tag deprecated: %w", err)
	}

	if err := s.store.ArchiveClient(ctx, userID, legacy.ID); err != nil {
		return nil, nil, fmt.Errorf("archive legacy client: %w", err)
	}
	if err := s.store.ArchiveTag(ctx, userID, deprecatedTag.ID); err != nil {
		return nil, nil, fmt.Errorf("archive deprecated tag: %w", err)
	}

	clients := []seedClient{
		{
			record: acme,
			projects: []seedProject{
				{
					record: website,
					tasks: []seedTask{
						{record: landingTask, descriptions: []string{"Layout polish", "Hero section", "Responsive QA"}},
						{record: dashboardTask, descriptions: []string{"Widget grid", "Client sync prep", "Chart styling"}},
					},
				},
				{
					record: apiProject,
					tasks: []seedTask{
						{record: importTask, descriptions: []string{"Import mapping review", "CSV validation", "Idempotency checks"}},
					},
				},
				{
					record: mobile,
					tasks: []seedTask{
						{record: mobileTask, descriptions: []string{"APNs wiring", "Foreground handler", "Release checklist"}},
					},
				},
			},
		},
		{
			record: northwind,
			projects: []seedProject{
				{
					record: dataPipeline,
					tasks: []seedTask{
						{record: etlTask, descriptions: []string{"Warehouse sync", "Retry policy", "Cost report"}},
					},
				},
			},
		},
		{
			record: beta,
			projects: []seedProject{
				{
					record: internal,
					tasks: []seedTask{
						{record: adminTask, descriptions: []string{"Planning and inbox", "Expense receipts", "Team retro notes"}},
					},
				},
			},
		},
		{
			record: legacy,
			projects: []seedProject{
				{
					record: oldProject,
					tasks: []seedTask{
						{record: legacyTask, descriptions: []string{"Final handoff", "DNS cutover"}},
					},
				},
			},
		},
	}

	tags := map[string]seedTag{
		"deep":       {record: deepWork},
		"meeting":    {record: meeting},
		"admin":      {record: adminTag},
		"review":     {record: review},
		"bugfix":     {record: bugfix},
		"deprecated": {record: deprecatedTag},
	}

	return clients, tags, nil
}

func (s *Service) createHistoricalEntries(ctx context.Context, userID string, clients []seedClient, tags map[string]seedTag, now time.Time) error {
	endDay := truncateDay(now)
	startDay := truncateDay(now.AddDate(0, -seedHistoryMonths, 0))

	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			if day.Day()%11 != 0 {
				continue
			}
		}

		slots := s.slotsForDay(day, clients, tags)
		for _, slot := range slots {
			desc := slot.task.descriptions[slot.descIndex%len(slot.task.descriptions)]
			startedAt := day.Add(time.Duration(slot.startHour)*time.Hour + time.Duration((day.Day()+slot.startHour)%4)*15*time.Minute)
			endedAt := startedAt.Add(slot.duration)

			if _, err := s.store.CreateTimeEntry(ctx, userID, store.TimeEntryInput{
				ClientID:    slot.project.record.ClientID,
				ProjectID:   slot.project.record.ID,
				TaskID:      slot.task.record.ID,
				TagIDs:      slot.tagIDs(tags),
				Description: desc,
				StartedAt:   startedAt.Format(time.RFC3339),
				EndedAt:     endedAt.Format(time.RFC3339),
				Billable:    slot.billable,
			}); err != nil {
				return fmt.Errorf("create seed time entry on %s: %w", day.Format("2006-01-02"), err)
			}
		}
	}

	return nil
}

func (s *Service) slotsForDay(day time.Time, clients []seedClient, tags map[string]seedTag) []timeSlot {
	acme := clients[0]
	northwind := clients[1]
	beta := clients[2]

	website := acme.projects[0]
	apiProject := acme.projects[1]
	mobile := acme.projects[2]
	dataPipeline := northwind.projects[0]
	internal := beta.projects[0]

	slots := []timeSlot{
		{
			startHour: 9,
			duration:  90 * time.Minute,
			project:   &website,
			task:      &website.tasks[day.Day()%len(website.tasks)],
			descIndex: day.Day(),
			billable:  true,
			tagIDs:    func(tags map[string]seedTag) []string { return []string{tags["deep"].record.ID} },
		},
		{
			startHour: 11,
			duration:  60 * time.Minute,
			project:   &apiProject,
			task:      &apiProject.tasks[0],
			descIndex: day.YearDay(),
			billable:  true,
			tagIDs:    func(tags map[string]seedTag) []string { return []string{tags["review"].record.ID} },
		},
		{
			startHour: 14,
			duration:  45 * time.Minute,
			project:   &internal,
			task:      &internal.tasks[0],
			descIndex: int(day.Month()),
			billable:  false,
			tagIDs:    func(tags map[string]seedTag) []string { return []string{tags["admin"].record.ID} },
		},
	}

	if day.Weekday() != time.Friday {
		slots = append(slots, timeSlot{
			startHour: 16,
			duration:  30 * time.Minute,
			project:   &website,
			task:      &website.tasks[1],
			descIndex: day.Day() + 1,
			billable:  true,
			tagIDs:    func(tags map[string]seedTag) []string { return []string{tags["meeting"].record.ID} },
		})
	}

	switch day.Weekday() {
	case time.Monday:
		_, week := day.ISOWeek()
		slots = append(slots, timeSlot{
			startHour: 17,
			duration:  40 * time.Minute,
			project:   &dataPipeline,
			task:      &dataPipeline.tasks[0],
			descIndex: week,
			billable:  true,
			tagIDs:    func(tags map[string]seedTag) []string { return []string{tags["deep"].record.ID} },
		})
	case time.Wednesday:
		slots = append(slots, timeSlot{
			startHour: 10,
			duration:  50 * time.Minute,
			project:   &mobile,
			task:      &mobile.tasks[0],
			descIndex: day.Day(),
			billable:  true,
			tagIDs: func(tags map[string]seedTag) []string {
				return []string{tags["bugfix"].record.ID, tags["review"].record.ID}
			},
		})
	}

	if day.Day()%9 == 0 {
		slots = append(slots, timeSlot{
			startHour: 18,
			duration:  25 * time.Minute,
			project:   &apiProject,
			task:      &apiProject.tasks[0],
			descIndex: day.Day(),
			billable:  true,
			tagIDs:    func(tags map[string]seedTag) []string { return []string{tags["bugfix"].record.ID} },
		})
	}

	return slots
}

func (s *Service) createInvoiceDrafts(ctx context.Context, userID string, clients []seedClient, now time.Time) error {
	acme := clients[0].record
	northwind := clients[1].record

	lastMonthEnd := truncateDay(now).AddDate(0, 0, -now.Day())
	lastMonthStart := lastMonthEnd.AddDate(0, -1, 1)
	twoMonthsEnd := lastMonthStart.AddDate(0, 0, -1)
	twoMonthsStart := twoMonthsEnd.AddDate(0, -1, 1)

	drafts := []struct {
		client *store.Client
		from   time.Time
		to     time.Time
		taxBP  int
		notes  string
	}{
		{acme, twoMonthsStart, twoMonthsEnd, 2100, "ACME monthly retainer"},
		{northwind, lastMonthStart, lastMonthEnd, 0, "Northwind consulting block"},
	}

	for _, draft := range drafts {
		if _, err := s.store.CreateInvoiceDraftFromTime(ctx, userID, store.InvoiceDraftFromTimeInput{
			ClientID:           draft.client.ID,
			From:               draft.from.Format(time.RFC3339),
			To:                 draft.to.Add(23*time.Hour + 59*time.Minute).Format(time.RFC3339),
			PeriodFrom:         draft.from.Format("2006-01-02"),
			PeriodTo:           draft.to.Format("2006-01-02"),
			TaxRateBasisPoints: draft.taxBP,
			Notes:              draft.notes,
		}); err != nil {
			return fmt.Errorf("create seed invoice draft for %s: %w", draft.client.Name, err)
		}
	}

	return nil
}

func (s *Service) createOpenTimer(ctx context.Context, userID string, clients []seedClient, tags map[string]seedTag) error {
	website := clients[0].projects[0]
	dashboardTask := website.tasks[1]

	if _, err := s.store.StartTimer(ctx, userID, store.TimerStartInput{
		ClientID:    clients[0].record.ID,
		ProjectID:   website.record.ID,
		TaskID:      dashboardTask.record.ID,
		TagIDs:      []string{tags["deep"].record.ID},
		Description: "Timer demo entry",
		Billable:    true,
	}); err != nil {
		return fmt.Errorf("create seed open timer: %w", err)
	}

	return nil
}

func truncateDay(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
