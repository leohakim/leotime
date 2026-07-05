package store

import (
	"context"
	"errors"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/db"
)

func TestClientLifecycle(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

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

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:                   "  Osoigo  ",
		Email:                  "billing@example.com",
		TaxID:                  "B12345678",
		BillingAddress:         "Madrid",
		DefaultCurrency:        "eur",
		DefaultHourlyRateMinor: 7500,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if client.Name != "Osoigo" || client.DefaultCurrency != "EUR" {
		t.Fatalf("expected normalized client, got %+v", client)
	}

	clients, err := st.ListClients(ctx, user.ID, false)
	if err != nil {
		t.Fatalf("list clients: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected one client, got %d", len(clients))
	}

	updated, err := st.UpdateClient(ctx, user.ID, client.ID, ClientInput{
		Name:                   "Osoigo SL",
		DefaultCurrency:        "USD",
		DefaultHourlyRateMinor: 9000,
	})
	if err != nil {
		t.Fatalf("update client: %v", err)
	}
	if updated.Name != "Osoigo SL" || updated.DefaultCurrency != "USD" || updated.DefaultHourlyRateMinor != 9000 {
		t.Fatalf("unexpected updated client: %+v", updated)
	}

	if err := st.ArchiveClient(ctx, user.ID, client.ID); err != nil {
		t.Fatalf("archive client: %v", err)
	}

	activeClients, err := st.ListClients(ctx, user.ID, false)
	if err != nil {
		t.Fatalf("list active clients: %v", err)
	}
	if len(activeClients) != 0 {
		t.Fatalf("expected no active clients, got %d", len(activeClients))
	}

	allClients, err := st.ListClients(ctx, user.ID, true)
	if err != nil {
		t.Fatalf("list all clients: %v", err)
	}
	if len(allClients) != 1 || allClients[0].ArchivedAt == "" {
		t.Fatalf("expected archived client, got %+v", allClients)
	}

	restored, err := st.RestoreClient(ctx, user.ID, client.ID)
	if err != nil {
		t.Fatalf("restore client: %v", err)
	}
	if restored.ArchivedAt != "" {
		t.Fatalf("expected restored client without archivedAt, got %+v", restored)
	}

	activeClientsAfterRestore, err := st.ListClients(ctx, user.ID, false)
	if err != nil {
		t.Fatalf("list active clients after restore: %v", err)
	}
	if len(activeClientsAfterRestore) != 1 {
		t.Fatalf("expected one active client after restore, got %d", len(activeClientsAfterRestore))
	}
}

func TestCreateClientValidatesInput(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := New(database)
	tests := []struct {
		name  string
		input ClientInput
	}{
		{
			name:  "missing name",
			input: ClientInput{Name: "", DefaultCurrency: "EUR"},
		},
		{
			name:  "invalid currency",
			input: ClientInput{Name: "Client", DefaultCurrency: "EURO"},
		},
		{
			name:  "invalid email",
			input: ClientInput{Name: "Client", Email: "billing", DefaultCurrency: "EUR"},
		},
		{
			name:  "negative rate",
			input: ClientInput{Name: "Client", DefaultCurrency: "EUR", DefaultHourlyRateMinor: -1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := st.CreateClient(ctx, "usr_missing", test.input); !errors.Is(err, ErrInvalidClientInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}
