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
	// Each style file has a distinctive declaration, which proves PreviewHTML
	// picked the template's own stylesheet.
	signatures := map[string]string{
		"default":  "Latin Modern Roman",
		"modern":   "--accent: #0f766e",
		"sidebar":  "grid-template-columns",
		"comic":    "Comic Sans MS",
		"dingbats": "Wingdings-adjacent",
	}

	var firstID uuid.UUID
	for _, tpl := range Templates() {
		signature, ok := signatures[tpl.ID]
		if !ok {
			t.Fatalf("test has no style signature for template %q", tpl.ID)
		}

		id, err := svc.CreateResume(ctx, user, ResumeInput{
			Name:       tpl.Name,
			TemplateID: tpl.ID,
			Data:       []byte(validJSON),
		})
		if err != nil {
			t.Fatalf("create %s: %v", tpl.ID, err)
		}
		if firstID == uuid.Nil {
			firstID = id
		}

		doc, err := svc.PreviewHTML(ctx, user, id)
		if err != nil {
			t.Fatalf("preview %s: %v", tpl.ID, err)
		}
		for _, want := range []string{"Jane Example", "data-print", "/static/print.js", signature} {
			if !strings.Contains(string(doc), want) {
				t.Errorf("%s preview missing %q", tpl.ID, want)
			}
		}
	}

	other, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewHTML(ctx, other, firstID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other user previewed resume: %v", err)
	}
}
