package web

import (
	"encoding/json"
	"net/url"
	"strings"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/render"
)

type contactForm struct {
	Type  string
	Value string
}

type experienceForm struct {
	Title      string
	Company    string
	Location   string
	StartDate  string
	EndDate    string
	Highlights string
}

type educationForm struct {
	School     string
	Degree     string
	Highlights string
}

type skillForm struct {
	Category string
	Items    string
}

// resumeForm is the editor's model. Repeated sections are rendered as rows and
// parsed back from repeated form fields, so no JavaScript is needed to submit.
type resumeForm struct {
	ID         string
	Action     string
	Name       string
	TemplateID string
	Templates  []app.Template
	// View is "form" or "json". Data holds the JSON shown in the textarea.
	View string
	Data string

	Fullname   string
	Contacts   []contactForm
	Experience []experienceForm
	Education  []educationForm
	Keywords   string
	Skills     []skillForm

	Error string

	// Empty rows are used as prototypes for the "add" buttons.
	EmptyContact    contactForm
	EmptyExperience experienceForm
	EmptyEducation  educationForm
	EmptySkill      skillForm
}

func newResumeForm(action string) resumeForm {
	return resumeForm{TemplateID: app.Templates()[0].ID}.withEditor(action)
}

// withEditor fills in the fields the template needs but that don't come from
// the submitted form or stored resume.
func (f resumeForm) withEditor(action string) resumeForm {
	f.Action = action
	f.Templates = app.Templates()
	f.EmptyContact = contactForm{Type: "email"}
	if f.View == "" {
		f.View = "form"
	}
	// The JSON view starts from the current form values. JSON submissions set
	// Data afterwards so their exact text is preserved.
	if f.Data == "" {
		if data, err := json.MarshalIndent(f.resumeData(), "", "  "); err == nil {
			f.Data = string(data)
		}
	}
	return f
}

// formFromData rebuilds the editor from stored resume JSON.
func formFromData(data []byte) resumeForm {
	var rd render.ResumeData
	if err := json.Unmarshal(data, &rd); err != nil {
		return resumeForm{}
	}

	f := resumeForm{
		Fullname: rd.Fullname,
		Keywords: strings.Join(rd.Keywords, ", "),
	}
	for _, c := range rd.Contact {
		f.Contacts = append(f.Contacts, contactForm{Type: normalizeContactType(c.Type), Value: c.Value})
	}
	for _, e := range rd.Experience {
		f.Experience = append(f.Experience, experienceForm{
			Title:      e.Title,
			Company:    e.Company,
			Location:   e.Location,
			StartDate:  e.StartDate,
			EndDate:    e.EndDate,
			Highlights: strings.Join(e.Highlights, "\n"),
		})
	}
	for _, e := range rd.Education {
		f.Education = append(f.Education, educationForm{
			School:     e.School,
			Degree:     e.Degree,
			Highlights: strings.Join(e.Highlights, "\n"),
		})
	}
	for _, s := range rd.Skills {
		f.Skills = append(f.Skills, skillForm{
			Category: s.Category,
			Items:    strings.Join(s.Items, ", "),
		})
	}
	return f
}

// parseResumeForm reads the editor's fields. Empty rows are dropped, so the
// placeholder row shown for an empty section never becomes an entry.
func parseResumeForm(v url.Values) resumeForm {
	f := resumeForm{
		Name:       strings.TrimSpace(v.Get("name")),
		TemplateID: v.Get("template"),
		Fullname:   strings.TrimSpace(v.Get("fullname")),
		Keywords:   strings.TrimSpace(v.Get("keywords")),
	}

	types, values := v["contact_type"], v["contact_value"]
	for i := 0; i < maxLen(types, values); i++ {
		value := strings.TrimSpace(at(values, i))
		if value == "" {
			continue
		}
		f.Contacts = append(f.Contacts, contactForm{
			Type:  normalizeContactType(at(types, i)),
			Value: value,
		})
	}

	titles := v["exp_title"]
	companies := v["exp_company"]
	locations := v["exp_location"]
	starts := v["exp_start"]
	ends := v["exp_end"]
	highlights := v["exp_highlights"]
	for i := 0; i < maxLen(titles, companies, locations, starts, ends, highlights); i++ {
		e := experienceForm{
			Title:      strings.TrimSpace(at(titles, i)),
			Company:    strings.TrimSpace(at(companies, i)),
			Location:   strings.TrimSpace(at(locations, i)),
			StartDate:  strings.TrimSpace(at(starts, i)),
			EndDate:    strings.TrimSpace(at(ends, i)),
			Highlights: strings.TrimSpace(at(highlights, i)),
		}
		if e.Title == "" && e.Company == "" && e.Location == "" &&
			e.StartDate == "" && e.EndDate == "" && e.Highlights == "" {
			continue
		}
		f.Experience = append(f.Experience, e)
	}

	schools, degrees, eduHighlights := v["edu_school"], v["edu_degree"], v["edu_highlights"]
	for i := 0; i < maxLen(schools, degrees, eduHighlights); i++ {
		e := educationForm{
			School:     strings.TrimSpace(at(schools, i)),
			Degree:     strings.TrimSpace(at(degrees, i)),
			Highlights: strings.TrimSpace(at(eduHighlights, i)),
		}
		if e.School == "" && e.Degree == "" && e.Highlights == "" {
			continue
		}
		f.Education = append(f.Education, e)
	}

	categories, items := v["skill_category"], v["skill_items"]
	for i := 0; i < maxLen(categories, items); i++ {
		s := skillForm{
			Category: strings.TrimSpace(at(categories, i)),
			Items:    strings.TrimSpace(at(items, i)),
		}
		if s.Category == "" && s.Items == "" {
			continue
		}
		f.Skills = append(f.Skills, s)
	}

	return f
}

// input converts the editor model into the JSON the service validates and stores.
// In JSON view the textarea is the source of truth, including its formatting.
func (f resumeForm) input() (app.ResumeInput, error) {
	if f.View == "json" {
		return app.ResumeInput{Name: f.Name, TemplateID: f.TemplateID, Data: []byte(f.Data)}, nil
	}
	data, err := json.MarshalIndent(f.resumeData(), "", "  ")
	if err != nil {
		return app.ResumeInput{}, err
	}
	return app.ResumeInput{Name: f.Name, TemplateID: f.TemplateID, Data: data}, nil
}

func (f resumeForm) resumeData() render.ResumeData {
	rd := render.ResumeData{
		Fullname:   strings.TrimSpace(f.Fullname),
		Contact:    []render.ResumeDataContact{},
		Experience: []render.ResumeDataExp{},
		Education:  []render.ResumeDataEdu{},
		Keywords:   splitList(f.Keywords),
		Skills:     []render.ResumeDataSkillGroup{},
	}
	for _, c := range f.Contacts {
		if c.Value == "" {
			continue
		}
		rd.Contact = append(rd.Contact, render.ResumeDataContact{
			Type:  normalizeContactType(c.Type),
			Value: strings.TrimSpace(c.Value),
		})
	}
	for _, e := range f.Experience {
		rd.Experience = append(rd.Experience, render.ResumeDataExp{
			Title:      strings.TrimSpace(e.Title),
			Company:    strings.TrimSpace(e.Company),
			Location:   strings.TrimSpace(e.Location),
			StartDate:  strings.TrimSpace(e.StartDate),
			EndDate:    strings.TrimSpace(e.EndDate),
			Highlights: splitLines(e.Highlights),
		})
	}
	for _, e := range f.Education {
		rd.Education = append(rd.Education, render.ResumeDataEdu{
			School:     strings.TrimSpace(e.School),
			Degree:     strings.TrimSpace(e.Degree),
			Highlights: splitLines(e.Highlights),
		})
	}
	for _, s := range f.Skills {
		rd.Skills = append(rd.Skills, render.ResumeDataSkillGroup{
			Category: strings.TrimSpace(s.Category),
			Items:    splitList(s.Items),
		})
	}
	return rd
}

// normalizeContactType keeps the types the template styles, and treats anything
// else as "other" (rendered as plain text).
func normalizeContactType(t string) string {
	switch t {
	case "email", "telephone", "website":
		return t
	default:
		return "other"
	}
}

func splitLines(s string) []string {
	out := []string{}
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitList(s string) []string {
	out := []string{}
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func at(v []string, i int) string {
	if i < len(v) {
		return v[i]
	}
	return ""
}

func maxLen(lists ...[]string) int {
	n := 0
	for _, l := range lists {
		if len(l) > n {
			n = len(l)
		}
	}
	return n
}
