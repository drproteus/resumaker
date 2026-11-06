package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/auth"
	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/gemini"
	"earthmanlabs/resumaker/internal/store"
)

func TestNormalizeAIResumeMergesComments(t *testing.T) {
	out, err := normalizeAIResume([]byte(`{
		"fullname": "Jane Example",
		"comments": ["Moved Go first.", "Gemini: Added the Acme keyword.", "  ", "Jane's own note"]
	}`), []string{"TODO: send the thank-you email"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		`"TODO: send the thank-you email"`,
		`"_Gemini: Moved Go first."`,
		`"_Gemini: Added the Acme keyword."`,
		`"_Gemini: Jane's own note"`,
		`"contact": []`,
		`"keywords": []`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("normalized JSON missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "_Gemini: _Gemini:") {
		t.Errorf("prefix was doubled:\n%s", text)
	}
	if strings.Contains(text, `"  "`) {
		t.Errorf("blank note was kept:\n%s", text)
	}
	if strings.Contains(text, `"response"`) {
		t.Errorf("normalized JSON still has a response field:\n%s", text)
	}
}

func TestNormalizeAIResumeDropsEchoedUserComments(t *testing.T) {
	out, err := normalizeAIResume([]byte(`{
		"fullname": "Jane Example",
		"comments": ["TODO: send the thank-you email", "Swapped the skills order."]
	}`), []string{"TODO: send the thank-you email"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), "TODO: send the thank-you email"); n != 1 {
		t.Errorf("user comment appears %d times, want 1:\n%s", n, out)
	}
}

func TestNormalizeAIResumeAddsTodos(t *testing.T) {
	out, err := normalizeAIResume([]byte(`{
		"fullname": "Jane Example",
		"comments": ["Moved Go first."]
	}`), nil, []string{"Verify the Acme start date", "  ", "Verify the Acme start date", "_TODO: Prepare a Go example"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		`"_TODO: Verify the Acme start date"`,
		`"_TODO: Prepare a Go example"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("normalized JSON missing %q:\n%s", want, text)
		}
	}
	if n := strings.Count(text, "Verify the Acme start date"); n != 1 {
		t.Errorf("duplicate todo kept (%d times):\n%s", n, text)
	}
	if strings.Contains(text, "_TODO: _TODO:") {
		t.Errorf("todo prefix was doubled:\n%s", text)
	}
}

func TestNormalizeAIResumeRequiresGeminiNote(t *testing.T) {
	_, err := normalizeAIResume([]byte(`{"fullname": "Jane", "comments": []}`), []string{"mine"}, nil)
	if err == nil || !strings.Contains(err.Error(), "no comments") {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeAIResumeRejectsUnknownFields(t *testing.T) {
	_, err := normalizeAIResume([]byte(`{"fullname": "Jane", "comments": ["x"], "invented": true}`), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("err = %v", err)
	}
}

func TestGeneratedResumeName(t *testing.T) {
	if got := generatedResumeName("  Senior Go Engineer - Acme  ", "Backend"); got != "Senior Go Engineer - Acme" {
		t.Errorf("suggested name = %q", got)
	}
	if got := generatedResumeName("", "Backend"); got != "Backend (AI)" {
		t.Errorf("fallback name = %q", got)
	}
	long := strings.Repeat("a", maxNameLength+20)
	if got := generatedResumeName(long, "Backend"); len(got) != maxNameLength {
		t.Errorf("long name length = %d, want %d", len(got), maxNameLength)
	}
}

type fakeGenerator struct {
	result gemini.Result
	err    error
	prompt string
	calls  int
}

func (f *fakeGenerator) TailorResume(_ context.Context, _ []byte, prompt string) (gemini.Result, error) {
	f.calls++
	f.prompt = prompt
	return f.result, f.err
}

// waitForTailoring polls a job until it leaves the pending state.
func waitForTailoring(t *testing.T, svc *Service, user store.User, jobID string) Tailoring {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		job, err := svc.Tailoring(user, jobID)
		if err != nil {
			t.Fatal(err)
		}
		if job.Status != TailoringPending {
			return job
		}
		if time.Now().After(deadline) {
			t.Fatal("tailoring job did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
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
	gen := &fakeGenerator{result: gemini.Result{
		Name: "Senior Go Engineer - Acme",
		Resume: []byte(`{
		"fullname": "Jane Tailored",
		"skills": [{"category": "Languages", "items": ["Go", "SQL"]}],
		"comments": ["Moved Go first because the job emphasises it."]
	}`),
	}}
	svc := New(st, keys, WithGenerator(gen))

	user, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Jane"})
	if err != nil {
		t.Fatal(err)
	}
	originalID, err := svc.CreateResume(ctx, user, ResumeInput{
		Name:       "Backend",
		TemplateID: "modern",
		Data:       []byte(`{"fullname": "Jane Example", "comments": ["Remember to update the portfolio link"]}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Empty prompts are rejected before any work starts.
	if _, err := svc.StartTailor(ctx, user, originalID, "  "); err == nil {
		t.Fatal("expected empty prompt to fail")
	}
	if gen.calls != 0 {
		t.Fatal("generator called for an empty prompt")
	}

	jobID, err := svc.StartTailor(ctx, user, originalID, "Read the job description and emphasise Go.")
	if err != nil {
		t.Fatal(err)
	}
	job := waitForTailoring(t, svc, user, jobID)
	if job.Status != TailoringDone {
		t.Fatalf("job status = %q, error %q", job.Status, job.Error)
	}
	if job.ResumeID == originalID || job.ResumeID == uuid.Nil {
		t.Fatalf("tailored resume ID = %s", job.ResumeID)
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
	if strings.Contains(string(original.Data), "_Gemini:") {
		t.Errorf("original gained AI notes: %s", original.Data)
	}
	if original.TemplateID != "modern" {
		t.Errorf("original template = %q", original.TemplateID)
	}

	tailored, err := svc.GetResume(ctx, user, job.ResumeID)
	if err != nil {
		t.Fatal(err)
	}
	if tailored.Name != "Senior Go Engineer - Acme" {
		t.Errorf("tailored name = %q", tailored.Name)
	}
	if tailored.TemplateID != "modern" {
		t.Errorf("tailored template = %q", tailored.TemplateID)
	}
	for _, want := range []string{
		`"Remember to update the portfolio link"`,
		`"_Gemini: Moved Go first because the job emphasises it."`,
		"Jane Tailored",
	} {
		if !strings.Contains(string(tailored.Data), want) {
			t.Errorf("tailored data missing %q:\n%s", want, tailored.Data)
		}
	}

	other, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartTailor(ctx, other, originalID, "instructions"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other user tailored resume: %v", err)
	}
	if _, err := svc.Tailoring(other, jobID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other user saw a tailoring job: %v", err)
	}
}

func TestTailoringReportsFailure(t *testing.T) {
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

	keys, err := envelope.NewKeyring(bytes.Repeat([]byte{8}, envelope.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	gen := &fakeGenerator{err: errors.New("upstream exploded")}
	svc := New(st, keys, WithGenerator(gen))

	user, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Jane"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := svc.CreateResume(ctx, user, ResumeInput{Name: "Backend", TemplateID: "default", Data: []byte(validJSON)})
	if err != nil {
		t.Fatal(err)
	}

	jobID, err := svc.StartTailor(ctx, user, id, "Emphasise Go.")
	if err != nil {
		t.Fatal(err)
	}
	job := waitForTailoring(t, svc, user, jobID)
	if job.Status != TailoringFailed {
		t.Fatalf("job status = %q", job.Status)
	}
	if strings.Contains(job.Error, "upstream exploded") {
		t.Errorf("internal error leaked to the user: %q", job.Error)
	}
}
