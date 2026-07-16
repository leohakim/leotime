package store

import (
	"context"
	"database/sql"
	"fmt"
)

type UserDataResetSummary struct {
	BillingDocuments int      `json:"billingDocuments"`
	InvoiceLines     int      `json:"invoiceLines"`
	Invoices         int      `json:"invoices"`
	TimeEntries      int      `json:"timeEntries"`
	Rates            int      `json:"rates"`
	Tasks            int      `json:"tasks"`
	Projects         int      `json:"projects"`
	Clients          int      `json:"clients"`
	Tags             int      `json:"tags"`
	InvoiceSeries    int      `json:"invoiceSeries"`
	DailySummaries   int      `json:"dailySummaries"`
	DailySummaryAI   int      `json:"dailySummaryAiRuns"`
	EmailOutbox      int      `json:"emailOutbox"`
	ExternalMappings int      `json:"externalMappings"`
	ImportRuns       int      `json:"importRuns"`
	DocumentPaths    []string `json:"documentPaths"`
}

func (s *Store) ClearUserData(ctx context.Context, userID string) (*UserDataResetSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	summary := &UserDataResetSummary{}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin clear user data: %w", err)
	}
	defer tx.Rollback()

	documentPaths, err := listBillingDocumentPathsTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	summary.DocumentPaths = documentPaths

	if err := nullInvoiceSeriesReferencesTx(ctx, tx, userID); err != nil {
		return nil, err
	}

	steps := []struct {
		query string
		into  *int
		args  []any
	}{
		{`DELETE FROM billing_documents WHERE user_id = ?`, &summary.BillingDocuments, []any{userID}},
		{`DELETE FROM invoice_lines WHERE invoice_id IN (SELECT id FROM invoices WHERE user_id = ?)`, &summary.InvoiceLines, []any{userID}},
		{`DELETE FROM invoices WHERE user_id = ?`, &summary.Invoices, []any{userID}},
		{`DELETE FROM time_entry_tags WHERE time_entry_id IN (SELECT id FROM time_entries WHERE user_id = ?)`, nil, []any{userID}},
		{`DELETE FROM email_outbox WHERE user_id = ?`, &summary.EmailOutbox, []any{userID}},
		{
			`DELETE FROM external_mappings WHERE internal_id IN (
				SELECT id FROM clients WHERE user_id = ?
				UNION ALL SELECT id FROM projects WHERE user_id = ?
				UNION ALL SELECT id FROM tasks WHERE user_id = ?
				UNION ALL SELECT id FROM tags WHERE user_id = ?
				UNION ALL SELECT id FROM time_entries WHERE user_id = ?
				UNION ALL SELECT id FROM invoices WHERE user_id = ?
			)`,
			&summary.ExternalMappings,
			[]any{userID, userID, userID, userID, userID, userID},
		},
		{`DELETE FROM time_entries WHERE user_id = ?`, &summary.TimeEntries, []any{userID}},
		{`DELETE FROM rates WHERE user_id = ?`, &summary.Rates, []any{userID}},
		{`DELETE FROM tasks WHERE user_id = ?`, &summary.Tasks, []any{userID}},
		{`DELETE FROM projects WHERE user_id = ?`, &summary.Projects, []any{userID}},
		{`DELETE FROM clients WHERE user_id = ?`, &summary.Clients, []any{userID}},
		{`DELETE FROM tags WHERE user_id = ?`, &summary.Tags, []any{userID}},
		{`DELETE FROM invoice_series WHERE user_id = ?`, &summary.InvoiceSeries, []any{userID}},
		{`DELETE FROM daily_summary_records WHERE user_id = ?`, &summary.DailySummaries, []any{userID}},
		{`DELETE FROM daily_summary_ai_runs WHERE user_id = ?`, &summary.DailySummaryAI, []any{userID}},
		{`DELETE FROM import_runs`, &summary.ImportRuns, nil},
	}

	for _, step := range steps {
		result, err := tx.ExecContext(ctx, step.query, step.args...)
		if err != nil {
			return nil, fmt.Errorf("clear user data query failed: %w", err)
		}
		if step.into != nil {
			affected, err := result.RowsAffected()
			if err != nil {
				return nil, fmt.Errorf("inspect clear user data rows: %w", err)
			}
			*step.into = int(affected)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit clear user data: %w", err)
	}

	if err := s.ensureDefaultInvoiceSeries(ctx, userID); err != nil {
		return nil, fmt.Errorf("ensure default invoice series after reset: %w", err)
	}

	return summary, nil
}

func listBillingDocumentPathsTx(ctx context.Context, tx *sql.Tx, userID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT storage_path
		FROM billing_documents
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list billing document paths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan billing document path: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate billing document paths: %w", err)
	}
	return paths, nil
}

func nullInvoiceSeriesReferencesTx(ctx context.Context, tx *sql.Tx, userID string) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE clients
		SET default_invoice_series_id = NULL, updated_at = ?
		WHERE user_id = ? AND default_invoice_series_id IS NOT NULL
	`, nowString(), userID); err != nil {
		return fmt.Errorf("clear client invoice series references: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE app_settings
		SET default_invoice_series_id = NULL, updated_at = ?
		WHERE user_id = ? AND default_invoice_series_id IS NOT NULL
	`, nowString(), userID); err != nil {
		return fmt.Errorf("clear app settings invoice series reference: %w", err)
	}

	return nil
}
