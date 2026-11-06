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

func TestNormalizeAIResume(t *testing.T) {
	out, err := normalizeAIResume([]byte(`{
		"fullname": "Jane Example",
		"ai": false,
		"comments": "  Emphasised Go because the job asks for it.  "
	}`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		`"ai": true`,
		`"comments": "Emphasised Go because the job asks for it."`,
		`"contact": []`,
		`"experience": []`,
		`"education": []`,
		`"keywords": []`,
		`"skills": []`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("normalized JSON missing %q:\n%s", want, text)
		}
	}
}

func TestNormalizeAIResumeRequiresComments(t *testing.T) {
	_, err := normalizeAIResume([]byte(`{"fullname": "Jane", "ai": true, "comments": "  "}`))
	if err == nil || !strings.Contains(err.Error(), "comments are required") {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeAIResumeRejectsUnknownFields(t *testing.T) {
	_, err := normalizeAIResume([]byte(`{"fullname": "Jane", "comments": "x", "invented": true}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("err = %v", err)
	}
}

func TestAIResumeName(t *testing.T) {
	if got := aiResumeName("Backend"); got != "Backend (AI)" {
		t.Errorf("aiResumeName = %q", got)
	}
	long := strings.Repeat("a", maxNameLength)
	if got := aiResumeName(long); len(got) > maxNameLength {
		t.Errorf("aiResumeName length = %d, want at most %d", len(got), maxNameLength)
	}
}

type fakeGenerator struct {
	data   []byte
	err    error
	prompt string
	calls  int
}

func (f *fakeGenerator) TailorResume(_ context.Context, _ []byte, prompt string) ([]byte, error) {
	f.calls++
	f.prompt = prompt
	return f.data, f.err
}

func TestTailorResumeCreatesNewVersion(t *testing.T) {
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

	keys, err := envelope.NewKeyring(bytes.Repeat([]byte{7}, envelope.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	gen := &fakeGenerator{data: []byte(`{
		"fullname": "Jane Tailored",
		"skills": [{"category": "Languages", "items": ["Go", "SQL"]}],
		"ai": false,
		"comments": "Moved Go first because the job emphasises it."
	}`)}
	svc := New(st, keys, WithGenerator(gen))

	user, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Jane"})
	if err != nil {
		t.Fatal(err)
	}
	originalID, err := svc.CreateResume(ctx, user, ResumeInput{
		Name:       "Backend",
		TemplateID: "modern",
		Data:       []byte(validJSON),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Empty prompts are rejected before calling Gemini.
	if _, err := svc.TailorResume(ctx, user, originalID, "  "); err == nil {
		t.Fatal("expected empty prompt to fail")
	}
	if gen.calls != 0 {
		t.Fatalf("generator called for an empty prompt")
	}

	newID, err := svc.TailorResume(ctx, user, originalID, "Read the job description and emphasise Go.")
	if err != nil {
		t.Fatal(err)
	}
	if newID == originalID {
		t.Fatal("tailored resume reused the original ID")
	}
	if gen.prompt != "Read the job description and emphasise Go." {
		t.Errorf("prompt = %q", gen.prompt)
	}

	original, err := svc.GetResume(ctx, user, originalID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(original.Data), `"fullname": "Jane Example"`) {
		t.Errorf("original changed: %s", original.Data)
	}
	if strings.Contains(string(original.Data), `"ai": true`) {
		t.Errorf("original was marked as AI-generated: %s", original.Data)
	}
	if original.TemplateID != "modern" {
		t.Errorf("original template = %q", original.TemplateID)
	}

	tailored, err := svc.GetResume(ctx, user, newID)
	if err != nil {
		t.Fatal(err)
	}
	if tailored.Name != "Backend (AI)" {
		t.Errorf("tailored name = %q", tailored.Name)
	}
	if tailored.TemplateID != "modern" {
		t.Errorf("tailored template = %q", tailored.TemplateID)
	}
	for _, want := range []string{`"ai": true`, `"comments": "Moved Go first`, "Jane Tailored"} {
		if !strings.Contains(string(tailored.Data), want) {
			t.Errorf("tailored data missing %q:\n%s", want, tailored.Data)
		}
	}

	other, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TailorResume(ctx, other, originalID, "instructions"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other user tailored resume: %v", err)
	}
}
