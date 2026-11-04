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
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/render"
)

// TestProcessRenderJobEndToEnd needs TEST_DATABASE_URL and a Chrome/Edge binary.
func TestProcessRenderJobEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
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
	jobID, err := svc.RequestRender(ctx, user, resumeID)
	if err != nil {
		t.Fatal(err)
	}

	// Jobs left over from earlier runs are claimed first; discard them.
	var claimed store.ClaimedRenderJob
	for i := 0; i < 50; i++ {
		job, err := svc.ClaimRenderJob(ctx)
		if errors.Is(err, store.ErrNotFound) {
			t.Fatal("queued job was not claimable")
		}
		if err != nil {
			t.Fatal(err)
		}
		if job.ID == jobID {
			claimed = job
			break
		}
		if err := svc.FailRenderJob(ctx, job.ID, "test cleanup"); err != nil {
			t.Fatal(err)
		}
	}
	if claimed.ID != jobID {
		t.Fatal("job was not claimed")
	}

	renderer, err := render.NewRenderer(render.RendererOptions{Timeout: 60 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Close()

	if err := svc.ProcessRenderJob(ctx, claimed, renderer); err != nil {
		t.Fatal(err)
	}

	jobs, err := svc.RenderJobs(ctx, user, resumeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) == 0 || jobs[0].Status != store.RenderSucceeded {
		t.Fatalf("job did not succeed: %+v", jobs)
	}

	pdf, contentType, err := svc.RenderArtifact(ctx, user, jobID, "pdf")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/pdf" || !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatalf("unexpected PDF: %q %q", contentType, firstBytes(pdf))
	}

	html, contentType, err := svc.RenderArtifact(ctx, user, jobID, "html")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "text/html; charset=utf-8" || !strings.Contains(string(html), "Jane Example") {
		t.Fatalf("unexpected HTML artifact: %q", contentType)
	}

	// Another user must not be able to read the artifact.
	other, err := svc.SignIn(ctx, auth.Identity{Provider: "test", ProviderUserID: uuid.NewString(), Name: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.RenderArtifact(ctx, other, jobID, "pdf"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other user read artifact: %v", err)
	}
}

func firstBytes(b []byte) string {
	if len(b) > 8 {
		b = b[:8]
	}
	return string(b)
}
