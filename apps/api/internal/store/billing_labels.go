package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func DefaultTaxLabel(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "Tax"
	}
	return "IVA"
}

func DefaultWithholdingLabel(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "Withholding"
	}
	return "Retención (IRPF)"
}

func ResolveWithholdingLabel(custom, locale string) string {
	if label := strings.TrimSpace(custom); label != "" {
		return label
	}
	return DefaultWithholdingLabel(locale)
}

func (s *Store) InvoiceWithholdingLabelSetting(ctx context.Context, userID string) (string, error) {
	var label string
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(invoice_withholding_label, '')
		FROM app_settings
		WHERE user_id = ?
	`, userID).Scan(&label)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("load invoice withholding label setting: %w", err)
	}
	return strings.TrimSpace(label), nil
}
