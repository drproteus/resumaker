package web

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/store"
)

func TestResumeListHasDuplicateButton(t *testing.T) {
	srv, err := New(Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	var buf bytes.Buffer
	p := page{Title: "Your resumes", Data: []store.Resume{{
		ID:         id,
		Name:       "Backend",
		TemplateID: "default",
		UpdatedAt:  time.Now(),
	}}}
	if err := srv.pages["resumes.html"].ExecuteTemplate(&buf, "layout", p); err != nil {
		t.Fatal(err)
	}

	body := buf.String()
	for _, want := range []string{
		`action="/resumes/` + id.String() + `/duplicate"`,
		`>Duplicate</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("resume list missing %q", want)
		}
	}
}
