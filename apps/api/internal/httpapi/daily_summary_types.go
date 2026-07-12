package httpapi

import "github.com/leotime/leotime/apps/api/internal/store"

type dailySummaryOptionsPayload struct {
	Date           string `json:"date"`
	Timezone       string `json:"timezone,omitempty"`
	Locale         string `json:"locale,omitempty"`
	IncludeClient  bool   `json:"includeClient"`
	IncludeProject bool   `json:"includeProject"`
	IncludeClosing bool   `json:"includeClosing"`
	BillableOnly   bool   `json:"billableOnly"`
	ManualNote     string `json:"manualNote,omitempty"`
	ClientID       string `json:"clientId"`
	ProjectID      string `json:"projectId"`
}

func (p dailySummaryOptionsPayload) toStore(fallbackDate string) store.DailySummaryOptions {
	date := p.Date
	if date == "" {
		date = fallbackDate
	}
	return store.DailySummaryOptions{
		Date:           date,
		Timezone:       p.Timezone,
		Locale:         p.Locale,
		IncludeClient:  p.IncludeClient,
		IncludeProject: p.IncludeProject,
		IncludeClosing: p.IncludeClosing,
		BillableOnly:   p.BillableOnly,
		ManualNote:     p.ManualNote,
		ClientID:       p.ClientID,
		ProjectID:      p.ProjectID,
	}
}
