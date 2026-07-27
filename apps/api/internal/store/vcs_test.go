package store

import (
	"context"
	"errors"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/db"
)

func TestVCSRepositoryInheritsConnectionClient(t *testing.T) {
	ctx := context.Background()
	st, user := newVCSTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Osoigo", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	connection, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider:        "gitea",
		BaseURL:         "https://gitea.osoigo.test",
		OwnerIdentity:   "leo@example.com",
		DefaultClientID: client.ID,
	}, "encrypted-token")
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	repository, err := st.UpsertVCSRepository(ctx, user.ID, VCSRepositoryInput{
		ConnectionID: connection.ID,
		Owner:        "osoigo",
		Name:         "backend",
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if repository.EffectiveClientID != client.ID {
		t.Fatalf("expected inherited client %q, got %q", client.ID, repository.EffectiveClientID)
	}
	if !connection.TokenConfigured {
		t.Fatal("expected masked token state")
	}
}

func TestVCSRepositoryRejectsAmbiguousAndCrossClientLinks(t *testing.T) {
	ctx := context.Background()
	st, user := newVCSTestStore(t, ctx)

	clientA, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Client A", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client A: %v", err)
	}
	clientB, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Client B", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client B: %v", err)
	}
	projectB, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: clientB.ID, Name: "Platform", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project B: %v", err)
	}

	withoutDefault, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider: "gitea",
		BaseURL:  "https://gitea.shared.test",
	}, "encrypted-token")
	if err != nil {
		t.Fatalf("create connection without default: %v", err)
	}
	if _, err := st.UpsertVCSRepository(ctx, user.ID, VCSRepositoryInput{
		ConnectionID: withoutDefault.ID,
		Owner:        "shared",
		Name:         "unassigned",
	}); !errors.Is(err, ErrInvalidVCSInput) {
		t.Fatalf("expected invalid repository without client, got %v", err)
	}

	withDefault, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider:        "gitea",
		BaseURL:         "https://gitea.client-a.test",
		DefaultClientID: clientA.ID,
	}, "encrypted-token")
	if err != nil {
		t.Fatalf("create connection with default: %v", err)
	}
	if _, err := st.UpsertVCSRepository(ctx, user.ID, VCSRepositoryInput{
		ConnectionID: withDefault.ID,
		Owner:        "client-a",
		Name:         "wrong-project",
		ProjectID:    projectB.ID,
	}); !errors.Is(err, ErrInvalidVCSInput) {
		t.Fatalf("expected invalid cross-client project link, got %v", err)
	}
}

func TestUpdateVCSConnectionKeepsTokenWhenEmpty(t *testing.T) {
	ctx := context.Background()
	st, user := newVCSTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Osoigo", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	created, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider:        "gitea",
		BaseURL:         "https://gitea.osoigo.test",
		OwnerIdentity:   "leo",
		DefaultClientID: client.ID,
	}, "encrypted-token-v1")
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	updated, err := st.UpdateVCSConnection(ctx, user.ID, created.ID, VCSConnectionInput{
		Provider:        "gitea",
		BaseURL:         "https://gitea.osoigo.test",
		OwnerIdentity:   "leo@example.com",
		DefaultClientID: client.ID,
	}, "")
	if err != nil {
		t.Fatalf("update connection: %v", err)
	}
	if updated.OwnerIdentity != "leo@example.com" {
		t.Fatalf("expected updated owner identity, got %q", updated.OwnerIdentity)
	}
	if !updated.TokenConfigured {
		t.Fatal("expected existing token to remain configured")
	}

	record, err := st.vcsConnectionRecordByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("reload connection: %v", err)
	}
	if record.TokenEnc != "encrypted-token-v1" {
		t.Fatalf("expected token preserved, got %q", record.TokenEnc)
	}
}

func TestUpdateVCSRepositoryChangesClientLink(t *testing.T) {
	ctx := context.Background()
	st, user := newVCSTestStore(t, ctx)

	clientA, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Client A", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client A: %v", err)
	}
	clientB, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Client B", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client B: %v", err)
	}
	connection, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider:        "gitea",
		BaseURL:         "https://gitea.osoigo.test",
		DefaultClientID: clientA.ID,
	}, "encrypted-token")
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	repository, err := st.UpsertVCSRepository(ctx, user.ID, VCSRepositoryInput{
		ConnectionID: connection.ID,
		Owner:        "osoigo",
		Name:         "backend",
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	updated, err := st.UpdateVCSRepository(ctx, user.ID, repository.ID, VCSRepositoryInput{
		ConnectionID: connection.ID,
		Owner:        "osoigo",
		Name:         "backend",
		ClientID:     clientB.ID,
	})
	if err != nil {
		t.Fatalf("update repository: %v", err)
	}
	if updated.ClientID != clientB.ID || updated.EffectiveClientID != clientB.ID {
		t.Fatalf("expected client B link, got client=%q effective=%q", updated.ClientID, updated.EffectiveClientID)
	}
}

func TestDeleteVCSConnectionCascadesRepositories(t *testing.T) {
	ctx := context.Background()
	st, user := newVCSTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Osoigo", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	connection, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider: "gitea", BaseURL: "https://gitea.osoigo.test", DefaultClientID: client.ID,
	}, "encrypted-token")
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	repository, err := st.UpsertVCSRepository(ctx, user.ID, VCSRepositoryInput{
		ConnectionID: connection.ID, Owner: "osoigo", Name: "backend",
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if err := st.DeleteVCSConnection(ctx, user.ID, connection.ID); err != nil {
		t.Fatalf("delete connection: %v", err)
	}
	if _, err := st.VCSRepositoryByID(ctx, user.ID, repository.ID); !errors.Is(err, ErrVCSRepositoryNotFound) {
		t.Fatalf("expected cascaded repository delete, got %v", err)
	}
}

func TestVCSRepositoryNormalizesOwnerNamePath(t *testing.T) {
	ctx := context.Background()
	st, user := newVCSTestStore(t, ctx)
	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Osoigo", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	connection, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider: "gitea", BaseURL: "https://gitea.osoigo.test", DefaultClientID: client.ID,
	}, "encrypted-token")
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	repository, err := st.UpsertVCSRepository(ctx, user.ID, VCSRepositoryInput{
		ConnectionID: connection.ID,
		Owner:        "leonardo.hakim@osoigo.com",
		Name:         "git.osoigo.com/ENACT/backend",
	})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if repository.Owner != "ENACT" || repository.Name != "backend" {
		t.Fatalf("expected normalized ENACT/backend, got %s/%s", repository.Owner, repository.Name)
	}
}

func TestVCSRepositoryRejectsEmailOwnerWithoutParsableName(t *testing.T) {
	ctx := context.Background()
	st, user := newVCSTestStore(t, ctx)
	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Osoigo", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	connection, err := st.CreateVCSConnection(ctx, user.ID, VCSConnectionInput{
		Provider: "gitea", BaseURL: "https://gitea.osoigo.test", DefaultClientID: client.ID,
	}, "encrypted-token")
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	if _, err := st.UpsertVCSRepository(ctx, user.ID, VCSRepositoryInput{
		ConnectionID: connection.ID,
		Owner:        "leonardo.hakim@osoigo.com",
		Name:         "backend",
	}); !errors.Is(err, ErrInvalidVCSInput) {
		t.Fatalf("expected invalid email owner, got %v", err)
	}
}

func newVCSTestStore(t *testing.T, ctx context.Context) (*Store, *User) {
	t.Helper()

	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})
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
