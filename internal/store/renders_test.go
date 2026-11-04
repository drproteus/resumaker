package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRenderQueueLifecycle(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	// Leftover jobs from an earlier run would be claimed first.
	if _, err := st.db.Exec(ctx, `DELETE FROM render_jobs WHERE status = 'queued'`); err != nil {
		t.Fatal(err)
	}

	owner := newUser(t, st)
	other := newUser(t, st)
	resumeID := uuid.New()
	revID := uuid.New()
	if err := st.CreateResume(ctx, owner.ID,
		Resume{ID: resumeID, Name: "Backend", TemplateID: "default"},
		Revision{ID: revID, Ciphertext: []byte("ciphertext")}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.db.Exec(ctx, `DELETE FROM resumes WHERE id = $1`, resumeID) })

	job, err := st.EnqueueRender(ctx, resumeID, revID)
	if err != nil {
		t.Fatal(err)
	}
	// A second request for the same resume must reuse the outstanding job.
	again, err := st.EnqueueRender(ctx, resumeID, revID)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != job.ID {
		t.Fatalf("enqueue created a duplicate job: %s vs %s", again.ID, job.ID)
	}

	claimed, err := st.ClaimRenderJob(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.ID != job.ID || claimed.UserID != owner.ID {
		t.Fatalf("claimed wrong job: %+v", claimed)
	}
	if _, err := st.ClaimRenderJob(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty queue: expected ErrNotFound, got %v", err)
	}

	src, err := st.RenderSource(ctx, resumeID, revID)
	if err != nil {
		t.Fatal(err)
	}
	if src.User.ID != owner.ID || src.TemplateID != "default" || string(src.Ciphertext) != "ciphertext" {
		t.Fatalf("unexpected source: %+v", src)
	}

	if err := st.CompleteRenderJob(ctx, job.ID, []byte("html"), []byte("pdf")); err != nil {
		t.Fatal(err)
	}
	got, html, pdf, err := st.RenderArtifact(ctx, owner.ID, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != RenderSucceeded || string(html) != "html" || string(pdf) != "pdf" {
		t.Fatalf("unexpected artifact: %+v %q %q", got, html, pdf)
	}
	if _, _, _, err := st.RenderArtifact(ctx, other.ID, job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user downloaded artifact: %v", err)
	}

	if err := st.DeleteResume(ctx, owner.ID, resumeID); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := st.RenderArtifact(ctx, owner.ID, job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("artifact survived resume deletion: %v", err)
	}
}

func TestFailAndRequeueJobs(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	if _, err := st.db.Exec(ctx, `DELETE FROM render_jobs WHERE status IN ('queued','running')`); err != nil {
		t.Fatal(err)
	}

	user := newUser(t, st)
	resumeID := uuid.New()
	revID := uuid.New()
	if err := st.CreateResume(ctx, user.ID,
		Resume{ID: resumeID, Name: "x", TemplateID: "default"},
		Revision{ID: revID, Ciphertext: []byte("c")}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.db.Exec(ctx, `DELETE FROM resumes WHERE id = $1`, resumeID) })

	job, err := st.EnqueueRender(ctx, resumeID, revID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.ClaimRenderJob(ctx); err != nil {
		t.Fatal(err)
	}

	// A job running longer than the threshold is presumed abandoned.
	if _, err := st.db.Exec(ctx,
		`UPDATE render_jobs SET started_at = now() - interval '1 hour' WHERE id = $1`, job.ID); err != nil {
		t.Fatal(err)
	}
	n, err := st.RequeueStaleRenderJobs(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("requeued %d jobs, want at least 1", n)
	}
	reclaimed, err := st.ClaimRenderJob(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.ID != job.ID {
		t.Fatalf("reclaimed %s, want %s", reclaimed.ID, job.ID)
	}

	if err := st.FailRenderJob(ctx, job.ID, "Rendering failed."); err != nil {
		t.Fatal(err)
	}
	jobs, err := st.ListRenderJobs(ctx, user.ID, resumeID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].Status != RenderFailed || jobs[0].Error != "Rendering failed." {
		t.Fatalf("unexpected jobs: %+v", jobs)
	}
	if _, _, _, err := st.RenderArtifact(ctx, user.ID, job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed job has an artifact: %v", err)
	}
}
