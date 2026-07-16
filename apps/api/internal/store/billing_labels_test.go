package store

import (
	"context"
	"testing"
)

func TestResolveWithholdingLabel(t *testing.T) {
	if got := ResolveWithholdingLabel("Income tax", "en"); got != "Income tax" {
		t.Fatalf("expected custom label, got %q", got)
	}
	if got := DefaultTaxLabel("es"); got != "IVA" {
		t.Fatalf("expected IVA, got %q", got)
	}
	if got := ResolveWithholdingLabel("", "en"); got != "Withholding" {
		t.Fatalf("expected default en label, got %q", got)
	}
}

func TestInvoiceWithholdingLabelSetting(t *testing.T) {
	ctx := context.Background()
	st, user := newProfileTestStore(t, ctx)

	if _, err := st.UpdateProfile(ctx, user.ID, ProfileUpdateInput{
		Name:                    user.Name,
		Email:                   user.Email,
		Locale:                  "en",
		LayoutMode:              "solid",
		DefaultCurrency:         "EUR",
		Timezone:                "Europe/Madrid",
		ThemeMode:               "solid",
		TimerStillRunningHours:  8,
		InvoiceWithholdingLabel: "PAYE",
	}); err != nil {
		t.Fatalf("update profile: %v", err)
	}

	label, err := st.InvoiceWithholdingLabelSetting(ctx, user.ID)
	if err != nil {
		t.Fatalf("load withholding label: %v", err)
	}
	if label != "PAYE" {
		t.Fatalf("expected PAYE, got %q", label)
	}
}
