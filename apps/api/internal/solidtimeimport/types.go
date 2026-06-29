package solidtimeimport

type Options struct {
	FilePath  string
	UserEmail string
	DryRun    bool
}

type Summary struct {
	Provider     string      `json:"provider"`
	ExportID     string      `json:"exportId"`
	Version      string      `json:"version"`
	DryRun       bool        `json:"dryRun"`
	Organization EntityStats `json:"organization"`
	Members      EntityStats `json:"members"`
	Clients      EntityStats `json:"clients"`
	Projects     EntityStats `json:"projects"`
	Tasks        EntityStats `json:"tasks"`
	Tags         EntityStats `json:"tags"`
	TimeEntries  EntityStats `json:"timeEntries"`
	Warnings     []string    `json:"warnings"`
	Errors       []string    `json:"errors"`
}

type EntityStats struct {
	Seen    int `json:"seen"`
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

type Export struct {
	Meta          Meta
	Organizations []Organization
	Members       []Member
	Clients       []Client
	Projects      []Project
	Tasks         []Task
	Tags          []Tag
	TimeEntries   []TimeEntry
}

type Meta struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Organizations []string `json:"organizations"`
	ExportedAt    string   `json:"exported_at"`
}

type Organization struct {
	ID           string
	Name         string
	BillableRate string
	Currency     string
	CreatedAt    string
	UpdatedAt    string
}

type Member struct {
	ID             string
	UserID         string
	Name           string
	Email          string
	OrganizationID string
	BillableRate   string
	Role           string
	CreatedAt      string
	UpdatedAt      string
}

type Client struct {
	ID             string
	Name           string
	OrganizationID string
	ArchivedAt     string
	CreatedAt      string
	UpdatedAt      string
}

type Project struct {
	ID             string
	Name           string
	Color          string
	BillableRate   string
	IsPublic       string
	ClientID       string
	OrganizationID string
	IsBillable     string
	ArchivedAt     string
	CreatedAt      string
	UpdatedAt      string
}

type Task struct {
	ID             string
	Name           string
	ProjectID      string
	OrganizationID string
	DoneAt         string
	CreatedAt      string
	UpdatedAt      string
}

type Tag struct {
	ID             string
	Name           string
	OrganizationID string
	CreatedAt      string
	UpdatedAt      string
}

type TimeEntry struct {
	ID                     string
	Description            string
	Start                  string
	End                    string
	BillableRate           string
	Billable               string
	MemberID               string
	UserID                 string
	OrganizationID         string
	ClientID               string
	ProjectID              string
	TaskID                 string
	Tags                   string
	IsImported             string
	StillActiveEmailSentAt string
	CreatedAt              string
	UpdatedAt              string
}
