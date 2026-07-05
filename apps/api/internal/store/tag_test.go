package store

import (
	"context"
	"errors"
	"testing"
)

func TestTagLifecycle(t *testing.T) {
	ctx := context.Background()
	st, user := newTagTestStore(t, ctx)

	tag, err := st.CreateTag(ctx, user.ID, TagInput{
		Name:  "  Deep Work  ",
		Color: "#2563eb",
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if tag.Name != "Deep Work" || tag.Color != "#2563eb" {
		t.Fatalf("expected normalized tag, got %+v", tag)
	}

	tags, err := st.ListTags(ctx, user.ID)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected one tag, got %d", len(tags))
	}

	updated, err := st.UpdateTag(ctx, user.ID, tag.ID, TagInput{
		Name:  "Focus",
		Color: "#0f7a5b",
	})
	if err != nil {
		t.Fatalf("update tag: %v", err)
	}
	if updated.Name != "Focus" || updated.Color != "#0f7a5b" {
		t.Fatalf("unexpected updated tag: %+v", updated)
	}

	if err := st.DeleteTag(ctx, user.ID, tag.ID); err != nil {
		t.Fatalf("delete tag: %v", err)
	}

	remaining, err := st.ListTags(ctx, user.ID)
	if err != nil {
		t.Fatalf("list tags after delete: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no tags, got %d", len(remaining))
	}
}

func TestCreateTagValidatesInput(t *testing.T) {
	ctx := context.Background()
	st, user := newTagTestStore(t, ctx)

	if _, err := st.CreateTag(ctx, user.ID, TagInput{Name: "", Color: "#2563eb"}); !errors.Is(err, ErrInvalidTagInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}

	if _, err := st.CreateTag(ctx, user.ID, TagInput{Name: "Valid", Color: "blue"}); !errors.Is(err, ErrInvalidTagInput) {
		t.Fatalf("expected invalid color, got %v", err)
	}
}

func TestTagNameMustBeUnique(t *testing.T) {
	ctx := context.Background()
	st, user := newTagTestStore(t, ctx)

	if _, err := st.CreateTag(ctx, user.ID, TagInput{Name: "Deep Work", Color: "#2563eb"}); err != nil {
		t.Fatalf("create first tag: %v", err)
	}

	if _, err := st.CreateTag(ctx, user.ID, TagInput{Name: "deep work", Color: "#64748b"}); !errors.Is(err, ErrDuplicateTagName) {
		t.Fatalf("expected duplicate name, got %v", err)
	}
}

func newTagTestStore(t *testing.T, ctx context.Context) (*Store, *User) {
	t.Helper()

	st, user := newTaskTestStore(t, ctx)
	return st, user
}
