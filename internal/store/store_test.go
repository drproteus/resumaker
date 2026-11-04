package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// openTestStore needs a disposable PostgreSQL database in TEST_DATABASE_URL.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	st, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return st
}

func newUser(t *testing.T, st *Store) User {
	t.Helper()
	u := User{ID: uuid.New(), DisplayName: "Test", WrappedKey: []byte("wrapped")}
	if err := st.CreateUser(context.Background(), u, "test", uuid.NewString(), "t@example.com"); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestMigrateIsIdempotent(t *testing.T) {
	st := openTestStore(t)
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestUserAndSessionLifecycle(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	u := newUser(t, st)
	providerID := uuid.NewString()
	u2 := User{ID: uuid.New(), DisplayName: "Other", WrappedKey: []byte("k")}
	if err := st.CreateUser(ctx, u2, "test", providerID, ""); err != nil {
		t.Fatal(err)
	}

	got, err := st.UserByIdentity(ctx, "test", providerID)
	if err != nil || got.ID != u2.ID {
		t.Fatalf("UserByIdentity = %v, %v", got.ID, err)
	}
	if _, err := st.UserByIdentity(ctx, "test", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	hash := []byte("session-hash-" + u.ID.String())
	if err := st.CreateSession(ctx, hash, u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UserBySession(ctx, hash); err != nil {
		t.Fatalf("active session: %v", err)
	}
	if err := st.DeleteSession(ctx, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UserBySession(ctx, hash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted session still valid: %v", err)
	}

	expired := []byte("expired-" + u.ID.String())
	if err := st.CreateSession(ctx, expired, u.ID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UserBySession(ctx, expired); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session still valid: %v", err)
	}
}

func TestResumeRevisionsAndOwnership(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	owner := newUser(t, st)
	other := newUser(t, st)

	id := uuid.New()
	rev1 := Revision{ID: uuid.New(), Ciphertext: []byte("v1")}
	if err := st.CreateResume(ctx, owner.ID, Resume{ID: id, Name: "Backend", TemplateID: "default"}, rev1); err != nil {
		t.Fatal(err)
	}

	r, rev, err := st.GetResume(ctx, owner.ID, id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Backend" || rev.ID != rev1.ID || !bytes.Equal(rev.Ciphertext, []byte("v1")) {
		t.Fatalf("unexpected resume %+v revision %+v", r, rev)
	}

	rev2 := Revision{ID: uuid.New(), Ciphertext: []byte("v2")}
	if err := st.UpdateResume(ctx, owner.ID, Resume{ID: id, Name: "Backend v2", TemplateID: "default"}, rev2); err != nil {
		t.Fatal(err)
	}
	_, rev, err = st.GetResume(ctx, owner.ID, id)
	if err != nil {
		t.Fatal(err)
	}
	if rev.ID != rev2.ID {
		t.Fatal("GetResume did not return the latest revision")
	}

	if _, _, err := st.GetResume(ctx, other.ID, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user read resume: %v", err)
	}
	if err := st.UpdateResume(ctx, other.ID, Resume{ID: id, Name: "x"}, Revision{ID: uuid.New()}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user updated resume: %v", err)
	}
	if err := st.DeleteResume(ctx, other.ID, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user deleted resume: %v", err)
	}

	list, err := st.ListResumes(ctx, owner.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListResumes = %d, %v", len(list), err)
	}

	if err := st.DeleteResume(ctx, owner.ID, id); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.GetResume(ctx, owner.ID, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted resume still found: %v", err)
	}
}
