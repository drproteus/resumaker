package web

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"earthmanlabs/resumaker/render"
)

func TestNewResumeFormRenders(t *testing.T) {
	srv, err := New(Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	p := page{Title: "New resume", Data: newResumeForm("/resumes")}
	if err := srv.pages["resume_form.html"].ExecuteTemplate(&buf, "layout", p); err != nil {
		t.Fatal(err)
	}

	body := buf.String()
	for _, want := range []string{
		`name="fullname"`,
		`name="exp_highlights"`,
		`data-add="contact"`,
		`data-drag-handle`,
		`data-template="experience"`,
		`name="editor_view"`,
		`id="view-json"`,
		`id="data"`,
		`class="resume-editor editor-view-form"`,
		`>Classic`,
		`>Comic`,
		`Wingdings-adjacent`,
		`/static/form.js`,
		`/static/form.css`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("form missing %q", want)
		}
	}
}

func TestParseResumeForm(t *testing.T) {
	values := url.Values{
		"name":           {"  Backend  "},
		"template":       {"default"},
		"fullname":       {" Jane Example "},
		"keywords":       {"go, postgres, , http"},
		"contact_type":   {"email", "website"},
		"contact_value":  {"jane@example.com", ""},
		"exp_title":      {"Engineer", ""},
		"exp_company":    {"Acme", ""},
		"exp_location":   {"Remote", ""},
		"exp_start":      {"2021", ""},
		"exp_end":        {"Present", ""},
		"exp_highlights": {"Built things\nShipped them", ""},
		"edu_school":     {"University", ""},
		"edu_degree":     {"BSc", ""},
		"edu_highlights": {"Honours", ""},
		"skill_category": {"Languages", ""},
		"skill_items":    {"Go, SQL", ""},
	}

	form := parseResumeForm(values)
	if form.Name != "Backend" || form.Fullname != "Jane Example" {
		t.Errorf("trimmed fields = %q / %q", form.Name, form.Fullname)
	}
	if len(form.Contacts) != 1 || form.Contacts[0].Type != "email" {
		t.Errorf("empty contact row was not dropped: %+v", form.Contacts)
	}
	if len(form.Experience) != 1 || len(form.Education) != 1 || len(form.Skills) != 1 {
		t.Errorf("empty rows were not dropped: %+v", form)
	}

	in, err := form.input()
	if err != nil {
		t.Fatal(err)
	}
	if in.Name != "Backend" || in.TemplateID != "default" {
		t.Errorf("input = %+v", in)
	}

	var data render.ResumeData
	if err := json.Unmarshal(in.Data, &data); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(data.Keywords, "|"); got != "go|postgres|http" {
		t.Errorf("keywords = %q", got)
	}
	if got := strings.Join(data.Experience[0].Highlights, "|"); got != "Built things|Shipped them" {
		t.Errorf("highlights = %q", got)
	}
	if got := strings.Join(data.Skills[0].Items, "|"); got != "Go|SQL" {
		t.Errorf("skill items = %q", got)
	}
}

func TestParseResumeFormPreservesOrder(t *testing.T) {
	values := url.Values{
		"name":           {"Backend"},
		"template":       {"default"},
		"fullname":       {"Jane"},
		"exp_title":      {"Second", "First"},
		"exp_company":    {"B", "A"},
		"skill_category": {"Later", "Earlier"},
		"skill_items":    {"Two", "One"},
	}

	form := parseResumeForm(values)
	if got := form.Experience[0].Title; got != "Second" {
		t.Errorf("first experience = %q, want the first submitted row", got)
	}
	if got := form.Skills[0].Category; got != "Later" {
		t.Errorf("first skill group = %q, want the first submitted row", got)
	}

	in, err := form.input()
	if err != nil {
		t.Fatal(err)
	}
	var data render.ResumeData
	if err := json.Unmarshal(in.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Experience[1].Title != "First" || data.Skills[1].Category != "Earlier" {
		t.Errorf("submitted order was lost: %+v / %+v", data.Experience, data.Skills)
	}
}

func TestResumeFormDeleteConfirmation(t *testing.T) {
	srv, err := New(Options{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	form := formFromData([]byte(`{"fullname": "Jane Example"}`)).withEditor("/resumes/abc")
	form.ID = "abc"
	form.Name = "Backend"

	var buf bytes.Buffer
	p := page{Title: "Edit resume", Data: form}
	if err := srv.pages["resume_form.html"].ExecuteTemplate(&buf, "layout", p); err != nil {
		t.Fatal(err)
	}

	body := buf.String()
	for _, want := range []string{
		`action="/resumes/abc/delete"`,
		`data-confirm="Delete “Backend”? This cannot be undone."`,
		`/static/delete.js`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("edit page missing %q", want)
		}
	}
}

func TestReadResumeFormJSONView(t *testing.T) {
	raw := "{\n  \"fullname\": \"Jane Example\",\n  \"skills\": [{\"category\": \"Go\", \"items\": [\"net/http\"]}]\n}"
	values := url.Values{
		"name":        {"  Backend  "},
		"template":    {"default"},
		"editor_view": {"json"},
		"data":        {raw},
	}

	req := httptest.NewRequest("POST", "/resumes", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, ok := readResumeForm(httptest.NewRecorder(), req, "/resumes")
	if !ok {
		t.Fatal("readResumeForm rejected a valid request")
	}
	if form.View != "json" {
		t.Errorf("view = %q, want json", form.View)
	}
	if form.Data != raw {
		t.Errorf("data was reformatted: %q", form.Data)
	}
	if form.Name != "Backend" || form.TemplateID != "default" {
		t.Errorf("metadata = %q / %q", form.Name, form.TemplateID)
	}
	if form.Fullname != "Jane Example" {
		t.Errorf("form fields were not populated from JSON: %q", form.Fullname)
	}

	in, err := form.input()
	if err != nil {
		t.Fatal(err)
	}
	if string(in.Data) != raw {
		t.Errorf("input changed the raw JSON: %q", in.Data)
	}
}

func TestReadResumeFormJSONViewKeepsInvalidText(t *testing.T) {
	raw := `{"fullname":`
	values := url.Values{
		"name":        {"Backend"},
		"template":    {"default"},
		"editor_view": {"json"},
		"data":        {raw},
	}

	req := httptest.NewRequest("POST", "/resumes", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, ok := readResumeForm(httptest.NewRecorder(), req, "/resumes")
	if !ok {
		t.Fatal("readResumeForm rejected the request")
	}
	if form.View != "json" || form.Data != raw {
		t.Errorf("invalid JSON was not preserved: %+v", form)
	}
}

func TestReadResumeFormFormViewBuildsJSONPreview(t *testing.T) {
	values := url.Values{
		"name":          {"Backend"},
		"template":      {"default"},
		"editor_view":   {"form"},
		"fullname":      {"Jane Example"},
		"contact_type":  {"email"},
		"contact_value": {"jane@example.com"},
	}

	req := httptest.NewRequest("POST", "/resumes", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, ok := readResumeForm(httptest.NewRecorder(), req, "/resumes")
	if !ok {
		t.Fatal("readResumeForm rejected the request")
	}
	if form.View != "form" {
		t.Errorf("view = %q, want form", form.View)
	}
	if !strings.Contains(form.Data, `"fullname": "Jane Example"`) {
		t.Errorf("form JSON preview missing fullname: %q", form.Data)
	}
	if !strings.Contains(form.Data, `"contact": [`) {
		t.Errorf("form JSON preview missing contact: %q", form.Data)
	}
}

func TestFormFromDataRoundTrips(t *testing.T) {
	data := []byte(`{
		"fullname": "Jane Example",
		"contact": [{"type": "email", "value": "jane@example.com"}],
		"experience": [{"title": "Engineer", "company": "Acme", "highlights": ["One", "Two"]}],
		"education": [{"school": "University", "degree": "BSc", "highlights": ["Honours"]}],
		"keywords": ["go", "sql"],
		"skills": [{"category": "Languages", "items": ["Go", "SQL"]}]
	}`)

	form := formFromData(data).withEditor("/resumes/x")
	if form.Fullname != "Jane Example" || len(form.Experience) != 1 {
		t.Fatalf("form = %+v", form)
	}
	if form.Experience[0].Highlights != "One\nTwo" {
		t.Errorf("highlights = %q", form.Experience[0].Highlights)
	}
	if form.Keywords != "go, sql" {
		t.Errorf("keywords = %q", form.Keywords)
	}
	if form.Action != "/resumes/x" || len(form.Templates) == 0 {
		t.Errorf("editor defaults missing: %+v", form)
	}

	in, err := form.input()
	if err != nil {
		t.Fatal(err)
	}
	var roundTripped render.ResumeData
	if err := json.Unmarshal(in.Data, &roundTripped); err != nil {
		t.Fatal(err)
	}
	if roundTripped.Experience[0].Highlights[1] != "Two" {
		t.Errorf("round trip lost highlights: %+v", roundTripped.Experience[0])
	}
}
