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
	"earthmanlabs/resumaker/internal/gemini"
	"earthmanlabs/resumaker/internal/store"
)

func TestValidFeature(t *testing.T) {
	if !ValidFeature(FeatureGemini) {
		t.Error("gemini should be a known feature")
	}
	if ValidFeature("nope") {
		t.Error("unknown feature accepted")
	}
}

func TestGeminiFeatureGatesTailoring(t *testing.T) {
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

	keys, err := envelope.NewKeyring(bytes.Repeat([]byte{5}, envelope.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	gen := &fakeGenerator{result: gemini.Result{Name: "Backend", Resume: []byte(`{"fullname": "Jane", "comments": ["_Gemini: Done."]}`)}}
	svc := New(st, keys, WithGenerator(gen))

	user, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Jane"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := svc.CreateResume(ctx, user, ResumeInput{Name: "Backend", TemplateID: "default", Data: []byte(validJSON)})
	if err != nil {
		t.Fatal(err)
	}

	if svc.AIAvailableFor(ctx, user) {
		t.Fatal("Gemini available before the feature is enabled")
	}
	if _, err := svc.StartTailor(ctx, user, id, "Emphasise Go."); err == nil {
		t.Fatal("tailoring started without the feature")
	} else {
		var inputErr *InputError
		if !errors.As(err, &inputErr) {
			t.Fatalf("err = %v, want an InputError", err)
		}
	}
	if gen.calls != 0 {
		t.Fatal("generator called without the feature")
	}

	if err := st.SetFeature(ctx, user.ID, FeatureGemini, true); err != nil {
		t.Fatal(err)
	}
	if !svc.AIAvailableFor(ctx, user) {
		t.Fatal("Gemini unavailable after the feature is enabled")
	}
	jobID, err := svc.StartTailor(ctx, user, id, "Emphasise Go.")
	if err != nil {
		t.Fatal(err)
	}
	if job := waitForTailoring(t, svc, user, jobID); job.Status != TailoringDone {
		t.Fatalf("job status = %q, error %q", job.Status, job.Error)
	}

	if err := st.SetFeature(ctx, user.ID, FeatureGemini, false); err != nil {
		t.Fatal(err)
	}
	if svc.AIAvailableFor(ctx, user) {
		t.Fatal("Gemini still available after the feature is disabled")
	}
}
