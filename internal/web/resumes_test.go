package web

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/internal/store"
)

func TestTailoringPageRendersStates(t *testing.T) {
	srv, err := New(Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	render := func(job app.Tailoring) string {
		var buf bytes.Buffer
		if err := srv.pages["tailoring.html"].ExecuteTemplate(&buf, "layout", page{Title: "Tailoring", Data: job}); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}

	pending := render(app.Tailoring{ID: "job1", Status: app.TailoringPending})
	for _, want := range []string{`data-status-url="/tailoring/job1/status"`, "/static/tailoring.js"} {
		if !strings.Contains(pending, want) {
			t.Errorf("pending page missing %q", want)
		}
	}

	failed := render(app.Tailoring{ID: "job1", Status: app.TailoringFailed, Error: "Gemini could not generate a resume."})
	if !strings.Contains(failed, "Gemini could not generate a resume.") || strings.Contains(failed, "data-tailoring ") {
		t.Error("failed page did not show the error")
	}
}

func TestResumeListActions(t *testing.T) {
	srv, err := New(Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	var buf bytes.Buffer
	p := page{Title: "Your resumes", Data: []resumeRow{{
		Resume: store.Resume{
			ID:         id,
			Name:       "Backend",
			TemplateID: "default",
			UpdatedAt:  time.Now(),
		},
		TemplateName: "Classic",
	}}}
	if err := srv.pages["resumes.html"].ExecuteTemplate(&buf, "layout", p); err != nil {
		t.Fatal(err)
	}

	body := buf.String()
	for _, want := range []string{
		`>Classic</td>`,
		`href="/resumes/` + id.String() + `/preview"`,
		`>Preview &amp; print</a>`,
		`action="/resumes/` + id.String() + `/duplicate"`,
		`>Duplicate</button>`,
		`action="/resumes/` + id.String() + `/delete"`,
		`data-confirm="Delete “Backend”? This cannot be undone."`,
		`>Delete</button>`,
		`/static/delete.js`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("resume list missing %q", want)
		}
	}
}
