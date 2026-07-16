package billing

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/store"
)

const SnapshotVersion = "billing-documents-v1"

type WorkProtocolDetail string

const (
	WorkProtocolSummary  WorkProtocolDetail = "summary"
	WorkProtocolStandard WorkProtocolDetail = "standard"
	WorkProtocolDetailed WorkProtocolDetail = "detailed"
)

type SnapshotOptions struct {
	Preview             bool
	IssueAt             time.Time
	SeriesCode          string
	PaymentInstructions string
	Locale              string
}

type DocumentSnapshot struct {
	Version      string               `json:"version"`
	Invoice      InvoiceSnapshot      `json:"invoice"`
	WorkProtocol WorkProtocolSnapshot `json:"workProtocol"`
}

type InvoiceSnapshot struct {
	Number               string                `json:"number"`
	Status               string                `json:"status"`
	Currency             string                `json:"currency"`
	IssuedAt             string                `json:"issuedAt"`
	DueAt                string                `json:"dueAt"`
	SellerName           string                `json:"sellerName"`
	SellerTaxID          string                `json:"sellerTaxId"`
	SellerAddress        string                `json:"sellerAddress"`
	ClientName           string                `json:"clientName"`
	ClientTaxID          string                `json:"clientTaxId"`
	ClientAddress        string                `json:"clientAddress"`
	PeriodFrom           string                `json:"periodFrom"`
	PeriodTo             string                `json:"periodTo"`
	PaymentInstructions  string                `json:"paymentInstructions"`
	Notes                string                `json:"notes"`
	SubtotalMinor        int64                 `json:"subtotalMinor"`
	TaxMinor             int64                 `json:"taxMinor"`
	WithholdingMinor     int64                 `json:"withholdingMinor"`
	TotalMinor           int64                 `json:"totalMinor"`
	TotalQuantityMinutes int                   `json:"totalQuantityMinutes"`
	TaxLabel             string                `json:"taxLabel"`
	WithholdingLabel     string                `json:"withholdingLabel"`
	Lines                []InvoiceLineSnapshot `json:"lines"`
	Preview              bool                  `json:"preview"`
}

type InvoiceLineSnapshot struct {
	Description     string `json:"description"`
	QuantityMinutes int    `json:"quantityMinutes"`
	UnitRateMinor   int64  `json:"unitRateMinor"`
	SubtotalMinor   int64  `json:"subtotalMinor"`
}

type WorkProtocolSnapshot struct {
	Number string               `json:"number"`
	Detail WorkProtocolDetail   `json:"detail"`
	Rows   []WorkProtocolDayRow `json:"rows"`
}

type WorkProtocolDayRow struct {
	Date         string   `json:"date"`
	Hours        string   `json:"hours"`
	ProjectNames string   `json:"projectNames,omitempty"`
	Items        []string `json:"items,omitempty"`
}

func BuildDocumentSnapshot(invoice *store.Invoice, entries []store.TimeEntry, options SnapshotOptions) (DocumentSnapshot, error) {
	if invoice == nil {
		return DocumentSnapshot{}, fmt.Errorf("invoice is required")
	}

	number := invoice.InvoiceNumber
	if options.Preview {
		number = previewInvoiceNumber(invoice, options)
	}

	displayInvoice := store.PrepareInvoiceForDisplay(invoice, entries, options.Locale)

	lineSnapshots := make([]InvoiceLineSnapshot, 0, len(displayInvoice.Lines))
	for _, line := range displayInvoice.Lines {
		lineSnapshots = append(lineSnapshots, InvoiceLineSnapshot{
			Description:     line.Description,
			QuantityMinutes: line.QuantityMinutes,
			UnitRateMinor:   line.UnitRateMinor,
			SubtotalMinor:   line.SubtotalMinor,
		})
	}

	detail := WorkProtocolDetail(invoice.WorkProtocolDetail)
	switch detail {
	case WorkProtocolSummary, WorkProtocolStandard, WorkProtocolDetailed:
	default:
		detail = WorkProtocolStandard
	}

	rows, err := buildWorkProtocolRows(entries, detail, options.Locale)
	if err != nil {
		return DocumentSnapshot{}, err
	}

	issuedAt := invoice.IssuedAt
	if !options.Preview && !options.IssueAt.IsZero() {
		issuedAt = options.IssueAt.UTC().Format(time.RFC3339Nano)
	}

	return DocumentSnapshot{
		Version: SnapshotVersion,
		Invoice: InvoiceSnapshot{
			Number:               number,
			Status:               invoice.Status,
			Currency:             invoice.Currency,
			IssuedAt:             issuedAt,
			DueAt:                invoice.DueAt,
			SellerName:           invoice.SellerName,
			SellerTaxID:          invoice.SellerTaxID,
			SellerAddress:        invoice.SellerAddress,
			ClientName:           invoice.ClientName,
			ClientTaxID:          invoice.ClientTaxID,
			ClientAddress:        invoice.ClientAddress,
			PeriodFrom:           invoice.PeriodFrom,
			PeriodTo:             invoice.PeriodTo,
			PaymentInstructions:  strings.TrimSpace(options.PaymentInstructions),
			Notes:                invoice.Notes,
			SubtotalMinor:        displayInvoice.SubtotalMinor,
			TaxMinor:             displayInvoice.TaxMinor,
			WithholdingMinor:     displayInvoice.WithholdingMinor,
			TotalMinor:           displayInvoice.TotalMinor,
			TotalQuantityMinutes: store.TotalInvoiceQuantityMinutes(displayInvoice.Lines),
			TaxLabel:             store.DefaultTaxLabel(options.Locale),
			WithholdingLabel:     store.ResolveWithholdingLabel(invoice.WithholdingLabel, options.Locale),
			Lines:                lineSnapshots,
			Preview:              options.Preview,
		},
		WorkProtocol: WorkProtocolSnapshot{
			Number: number,
			Detail: detail,
			Rows:   rows,
		},
	}, nil
}

func (s DocumentSnapshot) JSON() (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("marshal document snapshot: %w", err)
	}
	return string(payload), nil
}

func previewInvoiceNumber(invoice *store.Invoice, options SnapshotOptions) string {
	year := time.Now().UTC().Year()
	if !options.IssueAt.IsZero() {
		year = options.IssueAt.Year()
	}
	code := strings.TrimSpace(options.SeriesCode)
	if code == "" {
		code = "MAIN"
	}
	return fmt.Sprintf("PREVIEW-%d-%s-0001", year, code)
}

func buildWorkProtocolRows(entries []store.TimeEntry, detail WorkProtocolDetail, locale string) ([]WorkProtocolDayRow, error) {
	grouped := map[string][]store.TimeEntry{}
	for _, entry := range entries {
		day, err := entryDay(entry.StartedAt)
		if err != nil {
			return nil, err
		}
		grouped[day] = append(grouped[day], entry)
	}

	days := make([]string, 0, len(grouped))
	for day := range grouped {
		days = append(days, day)
	}
	sort.Strings(days)

	rows := make([]WorkProtocolDayRow, 0, len(days))
	for _, day := range days {
		dayEntries := grouped[day]
		totalSeconds := 0
		projectNames := map[string]struct{}{}
		for _, entry := range dayEntries {
			totalSeconds += entry.DurationSeconds
			if entry.ProjectName != "" {
				projectNames[entry.ProjectName] = struct{}{}
			}
		}

		row := WorkProtocolDayRow{
			Date:  day,
			Hours: formatHours(totalSeconds),
		}

		switch detail {
		case WorkProtocolSummary:
			label := joinSortedKeys(projectNames)
			if label == "" {
				label = summaryWorkProtocolLabel(locale)
			}
			row.Items = []string{label}
		case WorkProtocolStandard:
			row.Items = standardWorkItems(dayEntries)
		case WorkProtocolDetailed:
			row.Items = detailedWorkItems(dayEntries)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func standardWorkItems(entries []store.TimeEntry) []string {
	grouped := map[string]int{}
	for _, entry := range entries {
		key := strings.TrimSpace(entry.ProjectName)
		if key == "" {
			key = "Billable time"
		}
		grouped[key] += entry.DurationSeconds
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]string, 0, len(keys))
	for _, key := range keys {
		items = append(items, fmt.Sprintf("%s — %s h", key, formatProtocolHours(grouped[key])))
	}
	return items
}

func detailedWorkItems(entries []store.TimeEntry) []string {
	sorted := append([]store.TimeEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartedAt < sorted[j].StartedAt
	})

	items := make([]string, 0, len(sorted))
	for _, entry := range sorted {
		parts := make([]string, 0, 5)
		if start, end, ok := entryTimeRange(entry); ok {
			parts = append(parts, fmt.Sprintf("%s–%s", start, end))
		}
		if entry.ProjectName != "" {
			parts = append(parts, entry.ProjectName)
		}
		if entry.TaskName != "" {
			parts = append(parts, entry.TaskName)
		}
		if strings.TrimSpace(entry.Description) != "" {
			parts = append(parts, strings.TrimSpace(entry.Description))
		}
		if len(entry.Tags) > 0 {
			tagNames := make([]string, 0, len(entry.Tags))
			for _, tag := range entry.Tags {
				tagNames = append(tagNames, tag.Name)
			}
			sort.Strings(tagNames)
			parts = append(parts, strings.Join(tagNames, ", "))
		}
		label := strings.Join(parts, " · ")
		if label == "" {
			label = "Billable time"
		}
		items = append(items, fmt.Sprintf("%s (%s h)", label, formatProtocolHours(entry.DurationSeconds)))
	}
	return items
}

func entryTimeRange(entry store.TimeEntry) (string, string, bool) {
	started, err := parseEntryTimestamp(entry.StartedAt)
	if err != nil {
		return "", "", false
	}
	if strings.TrimSpace(entry.EndedAt) == "" {
		return started.Format("15:04"), "", true
	}
	ended, err := parseEntryTimestamp(entry.EndedAt)
	if err != nil {
		return started.Format("15:04"), "", true
	}
	return started.Format("15:04"), ended.Format("15:04"), true
}

func parseEntryTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Parse(time.RFC3339, value)
	}
	return parsed, nil
}

func formatProtocolHours(totalSeconds int) string {
	if totalSeconds <= 0 {
		return "0.00"
	}
	hours := float64(totalSeconds) / 3600
	return fmt.Sprintf("%.2f", hours)
}

func formatHours(totalSeconds int) string {
	return formatProtocolHours(totalSeconds)
}

func summaryWorkProtocolLabel(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "Billable professional services"
	}
	return "Servicios profesionales facturables"
}

func entryDay(value string) (string, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return "", fmt.Errorf("parse entry day: %w", err)
		}
	}
	return parsed.UTC().Format("2006-01-02"), nil
}

func joinSortedKeys(values map[string]struct{}) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
