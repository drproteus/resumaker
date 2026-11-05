package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/auth"
	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/store"
)

func TestPreviewHTML(t *testing.T) {
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

	keys, err := envelope.NewKeyring(bytes.Repeat([]byte{9}, envelope.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, keys)

	user, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Jane"})
	if err != nil {
		t.Fatal(err)
	}
	resumeID, err := svc.CreateResume(ctx, user, ResumeInput{
		Name:       "Backend",
		TemplateID: "default",
		Data:       []byte(validJSON),
	})
	if err != nil {
		t.Fatal(err)
	}

	doc, err := svc.PreviewHTML(ctx, user, resumeID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Jane Example",
		"data-print",
		"/static/print.js",
		"data:font/woff2;base64,",
	} {
		if !strings.Contains(string(doc), want) {
			t.Errorf("preview missing %q", want)
		}
	}

	other, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewHTML(ctx, other, resumeID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other user previewed resume: %v", err)
	}
}
