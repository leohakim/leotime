package store

import (
	"context"
	"testing"
)

func TestUpsertAISettingsPersistsGitAuthorEmail(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	settings, err := st.UpsertAISettings(ctx, user.ID, AISettingsInput{
		Enabled:        true,
		GitAuthorEmail: "dev@example.com",
	}, "")
	if err != nil {
		t.Fatalf("upsert ai settings: %v", err)
	}
	if !settings.Enabled || settings.GitAuthorEmail != "dev@example.com" {
		t.Fatalf("unexpected settings: %+v", settings)
	}

	loaded, err := st.AISettingsByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load ai settings: %v", err)
	}
	if loaded.GitAuthorEmail != "dev@example.com" || !loaded.Enabled {
		t.Fatalf("unexpected loaded settings: %+v", loaded)
	}
}

func TestUpsertAISettingsRejectsInvalidGitEmail(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	if _, err := st.UpsertAISettings(ctx, user.ID, AISettingsInput{
		GitAuthorEmail: "not-an-email",
	}, ""); err == nil {
		t.Fatal("expected invalid git author email")
	}
}
