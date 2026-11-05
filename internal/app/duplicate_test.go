package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/auth"
	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/store"
)

func TestDuplicateResume(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	keys, err := envelope.NewKeyring(bytes.Repeat([]byte{3}, envelope.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, keys)

	user, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Jane"})
	if err != nil {
		t.Fatal(err)
	}
	originalID, err := svc.CreateResume(ctx, user, ResumeInput{
		Name:       "Backend",
		TemplateID: "default",
		Data:       []byte(validJSON),
	})
	if err != nil {
		t.Fatal(err)
	}

	copyID, err := svc.DuplicateResume(ctx, user, originalID)
	if err != nil {
		t.Fatal(err)
	}
	if copyID == originalID {
		t.Fatal("duplicate reused the original resume ID")
	}

	copied, err := svc.GetResume(ctx, user, copyID)
	if err != nil {
		t.Fatal(err)
	}
	original, err := svc.GetResume(ctx, user, originalID)
	if err != nil {
		t.Fatal(err)
	}
	if copied.Name != "Backend (copy)" {
		t.Errorf("copied name = %q", copied.Name)
	}
	if copied.TemplateID != original.TemplateID {
		t.Errorf("copied template = %q, want %q", copied.TemplateID, original.TemplateID)
	}
	if !bytes.Equal(copied.Data, original.Data) {
		t.Errorf("copied data = %q, want %q", copied.Data, original.Data)
	}

	// Editing the copy must not modify the original.
	if err := svc.UpdateResume(ctx, user, copyID, ResumeInput{
		Name:       copied.Name,
		TemplateID: copied.TemplateID,
		Data:       []byte(`{"fullname": "Jane Copy"}`),
	}); err != nil {
		t.Fatal(err)
	}
	stillOriginal, err := svc.GetResume(ctx, user, originalID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stillOriginal.Data, original.Data) {
		t.Errorf("original changed after editing the copy: %q", stillOriginal.Data)
	}

	other, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DuplicateResume(ctx, other, originalID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other user duplicated resume: %v", err)
	}
}
