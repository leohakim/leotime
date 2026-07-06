package store

import (
	"context"
	"errors"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/db"
)

func newProfileTestStore(t *testing.T, ctx context.Context) (*Store, *User) {
	t.Helper()

	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	user, err := st.Authenticate(ctx, "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	return st, user
}

func TestProfileUpdateAndChangePassword(t *testing.T) {
	ctx := context.Background()
	st, user := newProfileTestStore(t, ctx)

	profile, err := st.ProfileByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if profile.Settings.DefaultCurrency != "EUR" {
		t.Fatalf("expected default currency EUR, got %q", profile.Settings.DefaultCurrency)
	}

	updated, err := st.UpdateProfile(ctx, user.ID, ProfileUpdateInput{
		Name:                "Leo",
		Email:               "leo@example.com",
		Locale:              "en",
		LayoutMode:          "compact",
		TaskProjectRequired: true,
		DefaultCurrency:     "USD",
		Timezone:            "America/New_York",
		ThemeMode:           "dark",
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Name != "Leo" || updated.Email != "leo@example.com" || updated.Locale != "en" {
		t.Fatalf("unexpected profile fields: %+v", updated)
	}
	if !updated.Settings.TaskProjectRequired || updated.Settings.DefaultCurrency != "USD" {
		t.Fatalf("unexpected settings: %+v", updated.Settings)
	}

	if err := st.ChangePassword(ctx, user.ID, ChangePasswordInput{
		CurrentPassword: "change-me-now",
		NewPassword:     "new-password-123",
	}); err != nil {
		t.Fatalf("change password: %v", err)
	}

	if _, err := st.Authenticate(ctx, "leo@example.com", "change-me-now"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected old password to fail, got %v", err)
	}
	if _, err := st.Authenticate(ctx, "leo@example.com", "new-password-123"); err != nil {
		t.Fatalf("authenticate with new password: %v", err)
	}
}

func TestProfileValidation(t *testing.T) {
	ctx := context.Background()
	st, user := newProfileTestStore(t, ctx)

	if _, err := st.UpdateProfile(ctx, user.ID, ProfileUpdateInput{
		Name:            "Leo",
		Email:           "admin@example.com",
		Locale:          "es",
		LayoutMode:      "solid",
		DefaultCurrency: "EURO",
		Timezone:        "Europe/Madrid",
		ThemeMode:       "solid",
	}); !errors.Is(err, ErrInvalidProfileInput) {
		t.Fatalf("expected invalid currency, got %v", err)
	}

	if err := st.ChangePassword(ctx, user.ID, ChangePasswordInput{
		CurrentPassword: "wrong",
		NewPassword:     "new-password-123",
	}); !errors.Is(err, ErrInvalidPasswordChange) {
		t.Fatalf("expected invalid password change, got %v", err)
	}
}

func TestProfileEmailTaken(t *testing.T) {
	ctx := context.Background()
	st, user := newProfileTestStore(t, ctx)

	if _, err := st.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, locale, layout_mode, created_at, updated_at)
		VALUES ('usr_other', 'other@example.com', 'Other', 'hash', 'es', 'solid', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert other user: %v", err)
	}

	if _, err := st.UpdateProfile(ctx, user.ID, ProfileUpdateInput{
		Name:            "Leo",
		Email:           "other@example.com",
		Locale:          "es",
		LayoutMode:      "solid",
		DefaultCurrency: "EUR",
		Timezone:        "Europe/Madrid",
		ThemeMode:       "solid",
	}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected email taken, got %v", err)
	}
}
