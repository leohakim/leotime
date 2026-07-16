package store

import (
	"fmt"
	"sort"
	"strings"
)

const (
	InvoiceLineDetailSummary   = "summary"
	InvoiceLineDetailByProject = "by_project"
	InvoiceLineDetailGranular  = "granular"
)

func normalizeInvoiceLineDetail(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case InvoiceLineDetailSummary:
		return InvoiceLineDetailSummary
	case InvoiceLineDetailByProject, "standard":
		return InvoiceLineDetailByProject
	case InvoiceLineDetailGranular, "detailed":
		return InvoiceLineDetailGranular
	default:
		return InvoiceLineDetailGranular
	}
}

func summaryInvoiceLineDescription(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "Software engineering hours"
	}
	return "Horas de Ingeniería de Software"
}

func roundMinutesToWholeHours(minutes int) int {
	if minutes <= 0 {
		return 0
	}
	hours := (minutes + 30) / 60
	if hours <= 0 {
		hours = 1
	}
	return hours * 60
}

func granularInvoiceLineDescription(entry TimeEntry) string {
	parts := make([]string, 0, 2)
	if entry.ProjectName != "" {
		parts = append(parts, entry.ProjectName)
	}
	if entry.TaskName != "" {
		parts = append(parts, entry.TaskName)
	}
	if len(parts) == 0 {
		return "Billable time"
	}
	return strings.Join(parts, " — ")
}

func buildInvoiceLineDrafts(entries []TimeEntry, client *Client, projectRates map[string]int64, taxRate int) []InvoiceLine {
	lineDrafts := make([]InvoiceLine, 0, len(entries))
	for _, entry := range entries {
		minutes := roundMinutesToWholeHours(entry.DurationSeconds / 60)
		if minutes <= 0 {
			continue
		}
		rate := resolveEntryHourlyRateMinor(entry, client, projectRates)
		lineDrafts = append(lineDrafts, InvoiceLine{
			TimeEntryID:        entry.ID,
			Description:        granularInvoiceLineDescription(entry),
			QuantityMinutes:    minutes,
			UnitRateMinor:      rate,
			SubtotalMinor:      lineSubtotalMinor(minutes, rate),
			TaxRateBasisPoints: taxRate,
		})
	}
	return lineDrafts
}

func PrepareInvoiceForDisplay(invoice *Invoice, entries []TimeEntry, locale string) *Invoice {
	if invoice == nil {
		return nil
	}

	display := *invoice
	entryByID := make(map[string]TimeEntry, len(entries))
	for _, entry := range entries {
		entryByID[entry.ID] = entry
	}

	display.Lines = mergeInvoiceLinesForDisplay(invoice.Lines, entryByID, invoice.InvoiceLineDetail, locale)
	totals := computeInvoiceTotals(display.Lines, invoice.WithholdingMinor)
	display.SubtotalMinor = totals.SubtotalMinor
	display.TaxMinor = totals.TaxMinor
	display.TotalMinor = totals.TotalMinor
	display.TotalQuantityMinutes = TotalInvoiceQuantityMinutes(display.Lines)
	return &display
}

type invoiceLineBucket struct {
	description string
	minutes     int
	subtotal    int64
	taxRate     int
}

func mergeInvoiceLinesForDisplay(lines []InvoiceLine, entryByID map[string]TimeEntry, detail string, locale string) []InvoiceLine {
	if len(lines) == 0 {
		return nil
	}

	addToBucket := func(buckets map[string]*invoiceLineBucket, key, description string, line InvoiceLine) {
		current, ok := buckets[key]
		if !ok {
			current = &invoiceLineBucket{description: description, taxRate: line.TaxRateBasisPoints}
			buckets[key] = current
		}
		current.minutes += line.QuantityMinutes
		current.subtotal += line.SubtotalMinor
	}

	switch normalizeInvoiceLineDetail(detail) {
	case InvoiceLineDetailSummary:
		totalMinutes := 0
		totalSubtotal := int64(0)
		taxRate := lines[0].TaxRateBasisPoints
		for _, line := range lines {
			totalMinutes += line.QuantityMinutes
			totalSubtotal += line.SubtotalMinor
			taxRate = line.TaxRateBasisPoints
		}
		roundedMinutes := roundMinutesToWholeHours(totalMinutes)
		unitRate := lines[0].UnitRateMinor
		if roundedMinutes > 0 && totalMinutes > 0 {
			unitRate = (totalSubtotal*60 + int64(roundedMinutes/2)) / int64(roundedMinutes)
		}
		return []InvoiceLine{{
			Description:        summaryInvoiceLineDescription(locale),
			QuantityMinutes:    roundedMinutes,
			UnitRateMinor:      unitRate,
			SubtotalMinor:      lineSubtotalMinor(roundedMinutes, unitRate),
			TaxRateBasisPoints: taxRate,
		}}
	case InvoiceLineDetailByProject:
		buckets := map[string]*invoiceLineBucket{}
		for _, line := range lines {
			entry, ok := entryByID[line.TimeEntryID]
			projectName := "Billable time"
			projectKey := "_default"
			if ok && entry.ProjectName != "" {
				projectName = entry.ProjectName
				projectKey = entry.ProjectID
				if projectKey == "" {
					projectKey = entry.ProjectName
				}
			}
			addToBucket(buckets, projectKey, projectName, line)
		}
		return materializeInvoiceLineBuckets(buckets, lines[0].TaxRateBasisPoints)
	default:
		merged := make([]InvoiceLine, 0, len(lines))
		for _, line := range lines {
			entry, ok := entryByID[line.TimeEntryID]
			description := line.Description
			if ok {
				description = granularInvoiceLineDescription(entry)
			}
			merged = append(merged, InvoiceLine{
				ID:                 line.ID,
				TimeEntryID:        line.TimeEntryID,
				Description:        description,
				QuantityMinutes:    line.QuantityMinutes,
				UnitRateMinor:      line.UnitRateMinor,
				SubtotalMinor:      line.SubtotalMinor,
				TaxRateBasisPoints: line.TaxRateBasisPoints,
				CreatedAt:          line.CreatedAt,
			})
		}
		return merged
	}
}

func materializeInvoiceLineBuckets(buckets map[string]*invoiceLineBucket, taxRate int) []InvoiceLine {
	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	merged := make([]InvoiceLine, 0, len(keys))
	for _, key := range keys {
		bucket := buckets[key]
		roundedMinutes := roundMinutesToWholeHours(bucket.minutes)
		unitRate := int64(0)
		if roundedMinutes > 0 && bucket.minutes > 0 {
			unitRate = (bucket.subtotal*60 + int64(roundedMinutes/2)) / int64(roundedMinutes)
		}
		merged = append(merged, InvoiceLine{
			Description:        bucket.description,
			QuantityMinutes:    roundedMinutes,
			UnitRateMinor:      unitRate,
			SubtotalMinor:      lineSubtotalMinor(roundedMinutes, unitRate),
			TaxRateBasisPoints: taxRate,
		})
	}
	return merged
}

func FormatInvoiceWholeHours(minutes int) string {
	if minutes <= 0 {
		return "0"
	}
	return fmt.Sprintf("%d", minutes/60)
}

func TotalInvoiceQuantityMinutes(lines []InvoiceLine) int {
	total := 0
	for _, line := range lines {
		total += line.QuantityMinutes
	}
	return total
}
