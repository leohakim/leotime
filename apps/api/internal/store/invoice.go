package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
)

var ErrInvoiceNotFound = errors.New("invoice not found")
var ErrInvalidInvoiceInput = errors.New("invalid invoice input")
var ErrInvoiceNotEditable = errors.New("invoice is not editable")

type Invoice struct {
	ID               string        `json:"id"`
	ClientID         string        `json:"clientId"`
	InvoiceNumber    string        `json:"invoiceNumber"`
	Status           string        `json:"status"`
	Currency         string        `json:"currency"`
	IssuedAt         string        `json:"issuedAt"`
	DueAt            string        `json:"dueAt"`
	SellerName       string        `json:"sellerName"`
	SellerTaxID      string        `json:"sellerTaxId"`
	SellerAddress    string        `json:"sellerAddress"`
	ClientName       string        `json:"clientName"`
	ClientTaxID      string        `json:"clientTaxId"`
	ClientAddress    string        `json:"clientAddress"`
	SubtotalMinor    int64         `json:"subtotalMinor"`
	TaxMinor         int64         `json:"taxMinor"`
	WithholdingMinor int64         `json:"withholdingMinor"`
	TotalMinor       int64         `json:"totalMinor"`
	Notes            string        `json:"notes"`
	Lines            []InvoiceLine `json:"lines"`
	CreatedAt        string        `json:"createdAt"`
	UpdatedAt        string        `json:"updatedAt"`
}

type InvoiceLine struct {
	ID                 string `json:"id"`
	TimeEntryID        string `json:"timeEntryId"`
	Description        string `json:"description"`
	QuantityMinutes    int    `json:"quantityMinutes"`
	UnitRateMinor      int64  `json:"unitRateMinor"`
	SubtotalMinor      int64  `json:"subtotalMinor"`
	TaxRateBasisPoints int    `json:"taxRateBasisPoints"`
	CreatedAt          string `json:"createdAt"`
}

type InvoiceDraftFromTimeInput struct {
	ClientID           string `json:"clientId"`
	From               string `json:"from"`
	To                 string `json:"to"`
	SellerName         string `json:"sellerName"`
	SellerTaxID        string `json:"sellerTaxId"`
	SellerAddress      string `json:"sellerAddress"`
	TaxRateBasisPoints int    `json:"taxRateBasisPoints"`
	WithholdingMinor   int64  `json:"withholdingMinor"`
	Notes              string `json:"notes"`
	DueAt              string `json:"dueAt"`
}

type InvoiceUpdateInput struct {
	DueAt              *string `json:"dueAt"`
	IssuedAt           *string `json:"issuedAt"`
	SellerName         *string `json:"sellerName"`
	SellerTaxID        *string `json:"sellerTaxId"`
	SellerAddress      *string `json:"sellerAddress"`
	ClientName         *string `json:"clientName"`
	ClientTaxID        *string `json:"clientTaxId"`
	ClientAddress      *string `json:"clientAddress"`
	WithholdingMinor   *int64  `json:"withholdingMinor"`
	Notes              *string `json:"notes"`
	TaxRateBasisPoints *int    `json:"taxRateBasisPoints"`
}

func (s *Store) ListInvoices(ctx context.Context, userID string) ([]Invoice, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, invoice_number, status, currency, COALESCE(issued_at, ''), COALESCE(due_at, ''),
			seller_name, seller_tax_id, seller_address, client_name, client_tax_id, client_address,
			subtotal_minor, tax_minor, withholding_minor, total_minor, notes, created_at, updated_at
		FROM invoices
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		invoice, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoices: %w", err)
	}
	return invoices, nil
}

func (s *Store) InvoiceByID(ctx context.Context, userID string, invoiceID string) (*Invoice, error) {
	invoice, err := queryInvoice(ctx, s.db, `
		SELECT id, client_id, invoice_number, status, currency, COALESCE(issued_at, ''), COALESCE(due_at, ''),
			seller_name, seller_tax_id, seller_address, client_name, client_tax_id, client_address,
			subtotal_minor, tax_minor, withholding_minor, total_minor, notes, created_at, updated_at
		FROM invoices
		WHERE user_id = ? AND id = ?
	`, userID, invoiceID)
	if err != nil {
		return nil, err
	}

	lines, err := s.listInvoiceLines(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	invoice.Lines = lines
	return invoice, nil
}

func (s *Store) CreateInvoiceDraftFromTime(ctx context.Context, userID string, input InvoiceDraftFromTimeInput) (*Invoice, error) {
	clientID := strings.TrimSpace(input.ClientID)
	if clientID == "" {
		return nil, fmt.Errorf("%w: client is required", ErrInvalidInvoiceInput)
	}
	from := strings.TrimSpace(input.From)
	to := strings.TrimSpace(input.To)
	if from == "" || to == "" {
		return nil, fmt.Errorf("%w: from and to are required", ErrInvalidInvoiceInput)
	}

	client, err := s.ClientByID(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}

	user, err := s.userByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	entries, err := s.listBillableUninvoicedEntries(ctx, userID, clientID, from, to)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: no billable uninvoiced time entries in range", ErrInvalidInvoiceInput)
	}

	projectRates, err := s.projectRateMap(ctx, userID)
	if err != nil {
		return nil, err
	}

	taxRate := input.TaxRateBasisPoints
	if taxRate < 0 {
		return nil, fmt.Errorf("%w: tax rate cannot be negative", ErrInvalidInvoiceInput)
	}
	withholding := input.WithholdingMinor
	if withholding < 0 {
		return nil, fmt.Errorf("%w: withholding cannot be negative", ErrInvalidInvoiceInput)
	}

	sellerName := strings.TrimSpace(input.SellerName)
	if sellerName == "" {
		sellerName = user.Name
	}

	invoiceNumber, err := s.nextInvoiceNumber(ctx, userID)
	if err != nil {
		return nil, err
	}

	invoiceID, err := newID("inv")
	if err != nil {
		return nil, err
	}
	now := nowString()

	lineDrafts := make([]InvoiceLine, 0, len(entries))
	for _, entry := range entries {
		minutes := entry.DurationSeconds / 60
		if minutes <= 0 {
			continue
		}
		rate := resolveEntryHourlyRateMinor(entry, client, projectRates)
		subtotal := lineSubtotalMinor(minutes, rate)
		description := invoiceLineDescription(entry)
		lineDrafts = append(lineDrafts, InvoiceLine{
			TimeEntryID:        entry.ID,
			Description:        description,
			QuantityMinutes:    minutes,
			UnitRateMinor:      rate,
			SubtotalMinor:      subtotal,
			TaxRateBasisPoints: taxRate,
		})
	}
	if len(lineDrafts) == 0 {
		return nil, fmt.Errorf("%w: no billable time with positive duration", ErrInvalidInvoiceInput)
	}

	totals := computeInvoiceTotals(lineDrafts, withholding)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin invoice draft: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO invoices (
			id, user_id, client_id, invoice_number, status, currency, issued_at, due_at,
			seller_name, seller_tax_id, seller_address, client_name, client_tax_id, client_address,
			subtotal_minor, tax_minor, withholding_minor, total_minor, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'draft', ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, invoiceID, userID, clientID, invoiceNumber, strings.ToUpper(strings.TrimSpace(client.DefaultCurrency)),
		nullIfEmpty(strings.TrimSpace(input.DueAt)),
		sellerName, strings.TrimSpace(input.SellerTaxID), strings.TrimSpace(input.SellerAddress),
		client.Name, client.TaxID, client.BillingAddress,
		totals.SubtotalMinor, totals.TaxMinor, totals.WithholdingMinor, totals.TotalMinor,
		strings.TrimSpace(input.Notes), now, now)
	if err != nil {
		return nil, fmt.Errorf("insert invoice: %w", err)
	}

	for _, line := range lineDrafts {
		lineID, err := newID("inl")
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO invoice_lines (
				id, invoice_id, time_entry_id, description, quantity_minutes, unit_rate_minor,
				subtotal_minor, tax_rate_basis_points, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, lineID, invoiceID, nullIfEmpty(line.TimeEntryID), line.Description, line.QuantityMinutes,
			line.UnitRateMinor, line.SubtotalMinor, line.TaxRateBasisPoints, now)
		if err != nil {
			return nil, fmt.Errorf("insert invoice line: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice draft: %w", err)
	}
	return s.InvoiceByID(ctx, userID, invoiceID)
}

func (s *Store) UpdateInvoice(ctx context.Context, userID string, invoiceID string, input InvoiceUpdateInput) (*Invoice, error) {
	invoice, err := s.InvoiceByID(ctx, userID, invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.Status != "draft" {
		return nil, ErrInvoiceNotEditable
	}

	if input.DueAt != nil {
		invoice.DueAt = strings.TrimSpace(*input.DueAt)
	}
	if input.IssuedAt != nil {
		invoice.IssuedAt = strings.TrimSpace(*input.IssuedAt)
	}
	if input.SellerName != nil {
		invoice.SellerName = strings.TrimSpace(*input.SellerName)
	}
	if input.SellerTaxID != nil {
		invoice.SellerTaxID = strings.TrimSpace(*input.SellerTaxID)
	}
	if input.SellerAddress != nil {
		invoice.SellerAddress = strings.TrimSpace(*input.SellerAddress)
	}
	if input.ClientName != nil {
		invoice.ClientName = strings.TrimSpace(*input.ClientName)
	}
	if input.ClientTaxID != nil {
		invoice.ClientTaxID = strings.TrimSpace(*input.ClientTaxID)
	}
	if input.ClientAddress != nil {
		invoice.ClientAddress = strings.TrimSpace(*input.ClientAddress)
	}
	if input.Notes != nil {
		invoice.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.WithholdingMinor != nil {
		if *input.WithholdingMinor < 0 {
			return nil, fmt.Errorf("%w: withholding cannot be negative", ErrInvalidInvoiceInput)
		}
		invoice.WithholdingMinor = *input.WithholdingMinor
	}

	if input.TaxRateBasisPoints != nil {
		if *input.TaxRateBasisPoints < 0 {
			return nil, fmt.Errorf("%w: tax rate cannot be negative", ErrInvalidInvoiceInput)
		}
		for index := range invoice.Lines {
			invoice.Lines[index].TaxRateBasisPoints = *input.TaxRateBasisPoints
		}
	}

	totals := computeInvoiceTotals(invoice.Lines, invoice.WithholdingMinor)
	invoice.SubtotalMinor = totals.SubtotalMinor
	invoice.TaxMinor = totals.TaxMinor
	invoice.TotalMinor = totals.TotalMinor
	now := nowString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin invoice update: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE invoices
		SET issued_at = ?, due_at = ?, seller_name = ?, seller_tax_id = ?, seller_address = ?,
			client_name = ?, client_tax_id = ?, client_address = ?,
			subtotal_minor = ?, tax_minor = ?, withholding_minor = ?, total_minor = ?, notes = ?, updated_at = ?
		WHERE user_id = ? AND id = ?
	`, nullIfEmpty(invoice.IssuedAt), nullIfEmpty(invoice.DueAt),
		invoice.SellerName, invoice.SellerTaxID, invoice.SellerAddress,
		invoice.ClientName, invoice.ClientTaxID, invoice.ClientAddress,
		invoice.SubtotalMinor, invoice.TaxMinor, invoice.WithholdingMinor, invoice.TotalMinor,
		invoice.Notes, now, userID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("update invoice: %w", err)
	}

	if input.TaxRateBasisPoints != nil {
		for _, line := range invoice.Lines {
			_, err = tx.ExecContext(ctx, `
				UPDATE invoice_lines SET tax_rate_basis_points = ? WHERE id = ? AND invoice_id = ?
			`, line.TaxRateBasisPoints, line.ID, invoiceID)
			if err != nil {
				return nil, fmt.Errorf("update invoice line tax: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice update: %w", err)
	}
	return s.InvoiceByID(ctx, userID, invoiceID)
}

func (s *Store) UpdateInvoiceStatus(ctx context.Context, userID string, invoiceID string, status string) (*Invoice, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "draft", "issued", "paid", "cancelled":
	default:
		return nil, fmt.Errorf("%w: invalid status", ErrInvalidInvoiceInput)
	}

	invoice, err := s.InvoiceByID(ctx, userID, invoiceID)
	if err != nil {
		return nil, err
	}

	now := nowString()
	issuedAt := invoice.IssuedAt
	if status == "issued" && issuedAt == "" {
		issuedAt = now
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE invoices SET status = ?, issued_at = ?, updated_at = ? WHERE user_id = ? AND id = ?
	`, status, nullIfEmpty(issuedAt), now, userID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("update invoice status: %w", err)
	}
	return s.InvoiceByID(ctx, userID, invoiceID)
}

func (s *Store) DeleteInvoice(ctx context.Context, userID string, invoiceID string) error {
	invoice, err := s.InvoiceByID(ctx, userID, invoiceID)
	if err != nil {
		return err
	}
	if invoice.Status != "draft" {
		return ErrInvoiceNotEditable
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM invoices WHERE user_id = ? AND id = ?`, userID, invoiceID)
	if err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete invoice rows: %w", err)
	}
	if rows == 0 {
		return ErrInvoiceNotFound
	}
	return nil
}

func (s *Store) RenderInvoiceHTML(invoice *Invoice) string {
	var builder strings.Builder
	builder.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>`)
	builder.WriteString(html.EscapeString(invoice.InvoiceNumber))
	builder.WriteString(`</title><style>
body{font-family:system-ui,sans-serif;color:#111;max-width:820px;margin:40px auto;padding:0 24px}
header{display:flex;justify-content:space-between;gap:24px;margin-bottom:32px}
h1{margin:0;font-size:28px}
.meta{color:#555;font-size:14px;line-height:1.5}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:24px;margin-bottom:32px}
.box{border:1px solid #ddd;border-radius:8px;padding:16px}
.box h2{margin:0 0 8px;font-size:13px;text-transform:uppercase;letter-spacing:.08em;color:#666}
table{width:100%;border-collapse:collapse;margin-bottom:24px}
th,td{border-bottom:1px solid #eee;padding:10px 8px;text-align:left;font-size:14px}
th{font-size:12px;text-transform:uppercase;color:#666}
.num{text-align:right;white-space:nowrap}
.totals{margin-left:auto;width:320px}
.totals div{display:flex;justify-content:space-between;padding:6px 0}
.totals strong{font-size:18px;border-top:1px solid #111;padding-top:10px;margin-top:8px}
.notes{margin-top:24px;color:#444;font-size:14px;white-space:pre-wrap}
@media print{body{margin:0;max-width:none}}
</style></head><body>`)

	builder.WriteString(`<header><div><h1>`)
	builder.WriteString(html.EscapeString(invoice.InvoiceNumber))
	builder.WriteString(`</h1><div class="meta">`)
	builder.WriteString(html.EscapeString(strings.ToUpper(invoice.Status)))
	builder.WriteString(` · `)
	builder.WriteString(html.EscapeString(invoice.Currency))
	if invoice.IssuedAt != "" {
		builder.WriteString(` · `)
		builder.WriteString(html.EscapeString(formatInvoiceDate(invoice.IssuedAt)))
	}
	builder.WriteString(`</div></div>`)
	if invoice.DueAt != "" {
		builder.WriteString(`<div class="meta"><strong>Due</strong><br>`)
		builder.WriteString(html.EscapeString(formatInvoiceDate(invoice.DueAt)))
		builder.WriteString(`</div>`)
	}
	builder.WriteString(`</header>`)

	builder.WriteString(`<div class="grid"><div class="box"><h2>From</h2>`)
	builder.WriteString(html.EscapeString(invoice.SellerName))
	if invoice.SellerTaxID != "" {
		builder.WriteString(`<br>`)
		builder.WriteString(html.EscapeString(invoice.SellerTaxID))
	}
	if invoice.SellerAddress != "" {
		builder.WriteString(`<br>`)
		builder.WriteString(html.EscapeString(invoice.SellerAddress))
	}
	builder.WriteString(`</div><div class="box"><h2>To</h2>`)
	builder.WriteString(html.EscapeString(invoice.ClientName))
	if invoice.ClientTaxID != "" {
		builder.WriteString(`<br>`)
		builder.WriteString(html.EscapeString(invoice.ClientTaxID))
	}
	if invoice.ClientAddress != "" {
		builder.WriteString(`<br>`)
		builder.WriteString(html.EscapeString(invoice.ClientAddress))
	}
	builder.WriteString(`</div></div>`)

	builder.WriteString(`<table><thead><tr><th>Description</th><th class="num">Qty (min)</th><th class="num">Rate</th><th class="num">Subtotal</th></tr></thead><tbody>`)
	for _, line := range invoice.Lines {
		builder.WriteString(`<tr><td>`)
		builder.WriteString(html.EscapeString(line.Description))
		builder.WriteString(`</td><td class="num">`)
		builder.WriteString(fmt.Sprintf("%d", line.QuantityMinutes))
		builder.WriteString(`</td><td class="num">`)
		builder.WriteString(html.EscapeString(formatMoneyMinor(line.UnitRateMinor, invoice.Currency)))
		builder.WriteString(`</td><td class="num">`)
		builder.WriteString(html.EscapeString(formatMoneyMinor(line.SubtotalMinor, invoice.Currency)))
		builder.WriteString(`</td></tr>`)
	}
	builder.WriteString(`</tbody></table>`)

	builder.WriteString(`<div class="totals"><div><span>Subtotal</span><span>`)
	builder.WriteString(html.EscapeString(formatMoneyMinor(invoice.SubtotalMinor, invoice.Currency)))
	builder.WriteString(`</span></div><div><span>Tax</span><span>`)
	builder.WriteString(html.EscapeString(formatMoneyMinor(invoice.TaxMinor, invoice.Currency)))
	builder.WriteString(`</span></div>`)
	if invoice.WithholdingMinor > 0 {
		builder.WriteString(`<div><span>Withholding</span><span>-`)
		builder.WriteString(html.EscapeString(formatMoneyMinor(invoice.WithholdingMinor, invoice.Currency)))
		builder.WriteString(`</span></div>`)
	}
	builder.WriteString(`<div><strong>Total</strong><strong>`)
	builder.WriteString(html.EscapeString(formatMoneyMinor(invoice.TotalMinor, invoice.Currency)))
	builder.WriteString(`</strong></div></div>`)

	if invoice.Notes != "" {
		builder.WriteString(`<div class="notes"><strong>Notes</strong><br>`)
		builder.WriteString(html.EscapeString(invoice.Notes))
		builder.WriteString(`</div>`)
	}

	builder.WriteString(`</body></html>`)
	return builder.String()
}

type invoiceTotals struct {
	SubtotalMinor    int64
	TaxMinor         int64
	WithholdingMinor int64
	TotalMinor       int64
}

func computeInvoiceTotals(lines []InvoiceLine, withholdingMinor int64) invoiceTotals {
	totals := invoiceTotals{WithholdingMinor: withholdingMinor}
	for _, line := range lines {
		totals.SubtotalMinor += line.SubtotalMinor
		totals.TaxMinor += lineTaxMinor(line)
	}
	totals.TotalMinor = totals.SubtotalMinor + totals.TaxMinor - totals.WithholdingMinor
	if totals.TotalMinor < 0 {
		totals.TotalMinor = 0
	}
	return totals
}

func lineTaxMinor(line InvoiceLine) int64 {
	if line.TaxRateBasisPoints <= 0 {
		return 0
	}
	return (line.SubtotalMinor*int64(line.TaxRateBasisPoints) + 5000) / 10000
}

func lineSubtotalMinor(quantityMinutes int, unitRateMinor int64) int64 {
	return (int64(quantityMinutes)*unitRateMinor + 30) / 60
}

func invoiceLineDescription(entry TimeEntry) string {
	parts := make([]string, 0, 3)
	if entry.ProjectName != "" {
		parts = append(parts, entry.ProjectName)
	}
	if entry.TaskName != "" {
		parts = append(parts, entry.TaskName)
	}
	description := strings.TrimSpace(entry.Description)
	if description != "" {
		parts = append(parts, description)
	}
	if len(parts) == 0 {
		return "Billable time"
	}
	return strings.Join(parts, " — ")
}

func resolveEntryHourlyRateMinor(entry TimeEntry, client *Client, projectRates map[string]int64) int64 {
	if entry.ProjectID != "" {
		if rate, ok := projectRates[entry.ProjectID]; ok {
			return rate
		}
	}
	if client != nil {
		return client.DefaultHourlyRateMinor
	}
	return 0
}

func (s *Store) listBillableUninvoicedEntries(ctx context.Context, userID string, clientID string, from string, to string) ([]TimeEntry, error) {
	entries, err := s.ListTimeEntries(ctx, userID, TimeEntryListOptions{
		From:     from,
		To:       to,
		ClientID: clientID,
	})
	if err != nil {
		return nil, err
	}

	filtered := make([]TimeEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.Billable {
			continue
		}
		invoiced, err := s.isTimeEntryInvoiced(ctx, entry.ID)
		if err != nil {
			return nil, err
		}
		if invoiced {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}

func (s *Store) isTimeEntryInvoiced(ctx context.Context, timeEntryID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM invoice_lines il
		INNER JOIN invoices i ON i.id = il.invoice_id
		WHERE il.time_entry_id = ? AND i.status != 'cancelled'
	`, timeEntryID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check invoiced entry: %w", err)
	}
	return count > 0, nil
}

func (s *Store) projectRateMap(ctx context.Context, userID string) (map[string]int64, error) {
	projects, err := s.ListProjects(ctx, userID, true, "")
	if err != nil {
		return nil, err
	}
	rates := make(map[string]int64, len(projects))
	for _, project := range projects {
		if project.DefaultHourlyRateMinor != nil {
			rates[project.ID] = *project.DefaultHourlyRateMinor
		}
	}
	return rates, nil
}

func (s *Store) nextInvoiceNumber(ctx context.Context, userID string) (string, error) {
	year := time.Now().UTC().Year()
	prefix := fmt.Sprintf("INV-%d-", year)
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM invoices WHERE user_id = ? AND invoice_number LIKE ?
	`, userID, prefix+"%").Scan(&count)
	if err != nil {
		return "", fmt.Errorf("count invoices: %w", err)
	}
	return fmt.Sprintf("%s%03d", prefix, count+1), nil
}

func (s *Store) userByID(ctx context.Context, userID string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, locale, layout_mode, created_at, updated_at
		FROM users WHERE id = ?
	`, userID)
	user := &User{}
	if err := row.Scan(&user.ID, &user.Email, &user.Name, &user.Locale, &user.LayoutMode, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("load user: %w", err)
	}
	return user, nil
}

func (s *Store) listInvoiceLines(ctx context.Context, invoiceID string) ([]InvoiceLine, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(time_entry_id, ''), description, quantity_minutes, unit_rate_minor,
			subtotal_minor, tax_rate_basis_points, created_at
		FROM invoice_lines
		WHERE invoice_id = ?
		ORDER BY created_at ASC, id ASC
	`, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("list invoice lines: %w", err)
	}
	defer rows.Close()

	var lines []InvoiceLine
	for rows.Next() {
		line := InvoiceLine{}
		if err := rows.Scan(&line.ID, &line.TimeEntryID, &line.Description, &line.QuantityMinutes,
			&line.UnitRateMinor, &line.SubtotalMinor, &line.TaxRateBasisPoints, &line.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invoice line: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice lines: %w", err)
	}
	return lines, nil
}

func queryInvoice(ctx context.Context, db *sql.DB, query string, args ...any) (*Invoice, error) {
	invoice, err := scanInvoice(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	return &invoice, nil
}

type invoiceScanner interface {
	Scan(dest ...any) error
}

func scanInvoice(scanner invoiceScanner) (Invoice, error) {
	var invoice Invoice
	var clientID sql.NullString
	if err := scanner.Scan(
		&invoice.ID, &clientID, &invoice.InvoiceNumber, &invoice.Status, &invoice.Currency,
		&invoice.IssuedAt, &invoice.DueAt, &invoice.SellerName, &invoice.SellerTaxID, &invoice.SellerAddress,
		&invoice.ClientName, &invoice.ClientTaxID, &invoice.ClientAddress,
		&invoice.SubtotalMinor, &invoice.TaxMinor, &invoice.WithholdingMinor, &invoice.TotalMinor,
		&invoice.Notes, &invoice.CreatedAt, &invoice.UpdatedAt,
	); err != nil {
		return Invoice{}, err
	}
	if clientID.Valid {
		invoice.ClientID = clientID.String
	}
	return invoice, nil
}

func formatMoneyMinor(amountMinor int64, currency string) string {
	sign := ""
	if amountMinor < 0 {
		sign = "-"
		amountMinor = -amountMinor
	}
	whole := amountMinor / 100
	fraction := amountMinor % 100
	return fmt.Sprintf("%s%s %d.%02d", sign, strings.ToUpper(strings.TrimSpace(currency)), whole, fraction)
}

func formatInvoiceDate(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return value
		}
	}
	return parsed.UTC().Format("2006-01-02")
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
