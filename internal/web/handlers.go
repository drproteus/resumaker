package web

import (
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/internal/store"
)

type providerLink struct {
	Slug  string
	Label string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(r); ok {
		http.Redirect(w, r, "/resumes", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// helpView is the data shown on the help and about page.
type helpView struct {
	Templates []app.Template
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	p := page{Title: "Help & about", Data: helpView{Templates: app.Templates()}}
	if user, ok := s.currentUser(r); ok {
		p.User = &user
	}
	s.render(w, http.StatusOK, "about.html", p)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(r); ok {
		http.Redirect(w, r, "/resumes", http.StatusSeeOther)
		return
	}
	links := make([]providerLink, 0, len(s.providers))
	for _, p := range s.providers {
		links = append(links, providerLink{Slug: p.Slug, Label: p.Label})
	}
	s.render(w, http.StatusOK, "login.html", page{Title: "Sign in", Data: links})
}

// resumeRow adds the display name of the resume's style to a stored resume.
type resumeRow struct {
	store.Resume
	TemplateName string
}

func (s *Server) handleResumeList(w http.ResponseWriter, r *http.Request, user store.User) {
	resumes, err := s.app.ListResumes(r.Context(), user)
	if err != nil {
		s.fail(w, err)
		return
	}

	names := make(map[string]string, len(app.Templates()))
	for _, tpl := range app.Templates() {
		names[tpl.ID] = tpl.Name
	}
	rows := make([]resumeRow, len(resumes))
	for i, resume := range resumes {
		name := names[resume.TemplateID]
		if name == "" {
			name = resume.TemplateID
		}
		rows[i] = resumeRow{Resume: resume, TemplateName: name}
	}

	s.render(w, http.StatusOK, "resumes.html", page{Title: "Your resumes", User: &user, Data: rows})
}

func (s *Server) handleResumeNew(w http.ResponseWriter, r *http.Request, user store.User) {
	s.render(w, http.StatusOK, "resume_form.html", page{Title: "New resume", User: &user, Data: newResumeForm("/resumes")})
}

func (s *Server) handleResumeCreate(w http.ResponseWriter, r *http.Request, user store.User) {
	form, ok := readResumeForm(w, r, "/resumes")
	if !ok {
		return
	}
	in, err := form.input()
	if err != nil {
		s.fail(w, err)
		return
	}
	id, err := s.app.CreateResume(r.Context(), user, in)
	if err != nil {
		var inputErr *app.InputError
		if errors.As(err, &inputErr) {
			form.Error = inputErr.Msg
			s.render(w, http.StatusUnprocessableEntity, "resume_form.html", page{Title: "New resume", User: &user, Data: form})
			return
		}
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/resumes/"+id.String(), http.StatusSeeOther)
}

func (s *Server) handleResumeEdit(w http.ResponseWriter, r *http.Request, user store.User) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	form, err := s.editForm(r, user, id)
	if err != nil {
		s.resumeError(w, r, err)
		return
	}
	s.render(w, http.StatusOK, "resume_form.html", page{Title: "Edit resume", User: &user, Data: form})
}

// editForm loads a stored resume into the editor model, including the optional
// Gemini controls.
func (s *Server) editForm(r *http.Request, user store.User, id uuid.UUID) (resumeForm, error) {
	detail, err := s.app.GetResume(r.Context(), user, id)
	if err != nil {
		return resumeForm{}, err
	}
	form := formFromData(detail.Data).withEditor("/resumes/" + id.String())
	form.ID = id.String()
	form.Name = detail.Name
	form.TemplateID = detail.TemplateID
	form.AIAvailable = s.app.AIGenerationAvailable()
	return form, nil
}

func (s *Server) handleResumeUpdate(w http.ResponseWriter, r *http.Request, user store.User) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	form, ok := readResumeForm(w, r, "/resumes/"+id.String())
	if !ok {
		return
	}
	in, err := form.input()
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.app.UpdateResume(r.Context(), user, id, in); err != nil {
		var inputErr *app.InputError
		if errors.As(err, &inputErr) {
			form.ID = id.String()
			form.AIAvailable = s.app.AIGenerationAvailable()
			form.Error = inputErr.Msg
			s.render(w, http.StatusUnprocessableEntity, "resume_form.html", page{Title: "Edit resume", User: &user, Data: form})
			return
		}
		s.resumeError(w, r, err)
		return
	}
	http.Redirect(w, r, "/resumes/"+id.String(), http.StatusSeeOther)
}

func (s *Server) handleResumeDuplicate(w http.ResponseWriter, r *http.Request, user store.User) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	newID, err := s.app.DuplicateResume(r.Context(), user, id)
	if err != nil {
		s.resumeError(w, r, err)
		return
	}
	http.Redirect(w, r, "/resumes/"+newID.String(), http.StatusSeeOther)
}

// handleResumeTailor creates a new AI-generated version of a resume. The
// original stays untouched, so the user can compare the two.
func (s *Server) handleResumeTailor(w http.ResponseWriter, r *http.Request, user store.User) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Request is too large or malformed.", http.StatusBadRequest)
		return
	}
	prompt := r.PostFormValue("prompt")

	newID, err := s.app.TailorResume(r.Context(), user, id, prompt)
	if err != nil {
		var inputErr *app.InputError
		if errors.As(err, &inputErr) {
			s.renderTailorError(w, r, user, id, prompt, inputErr.Msg, http.StatusUnprocessableEntity)
			return
		}
		log.Printf("tailor resume %s: %v", id, err)
		s.renderTailorError(w, r, user, id, prompt,
			"Gemini could not generate a resume. Please try again in a moment.",
			http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/resumes/"+newID.String(), http.StatusSeeOther)
}

func (s *Server) renderTailorError(w http.ResponseWriter, r *http.Request, user store.User, id uuid.UUID, prompt, message string, status int) {
	form, err := s.editForm(r, user, id)
	if err != nil {
		s.resumeError(w, r, err)
		return
	}
	form.Prompt = prompt
	form.Error = message
	s.render(w, status, "resume_form.html", page{Title: "Edit resume", User: &user, Data: form})
}

func (s *Server) handleResumeDelete(w http.ResponseWriter, r *http.Request, user store.User) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.app.DeleteResume(r.Context(), user, id); err != nil {
		s.resumeError(w, r, err)
		return
	}
	http.Redirect(w, r, "/resumes", http.StatusSeeOther)
}

// resumeError maps "not found" to a 404 so other users' resumes look the same as missing ones.
func (s *Server) resumeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	s.fail(w, err)
}

// previewCSP allows the resume's inlined stylesheet and fonts. The document is
// generated by html/template, so no untrusted markup reaches the page.
const previewCSP = "default-src 'none'; style-src 'unsafe-inline'; font-src data:; " +
	"img-src data:; script-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// handlePreview renders the resume as a standalone page, ready for the
// browser's print dialog.
func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request, user store.User) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	doc, err := s.app.PreviewHTML(r.Context(), user, id)
	if err != nil {
		s.resumeError(w, r, err)
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", previewCSP)
	h.Set("Cache-Control", "no-store")
	w.Write(doc)
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return uuid.Nil, false
	}
	return id, true
}

// readResumeForm parses the editor's fields, rebuilding the form so a
// validation error doesn't lose the user's edits. The JSON view uses its
// textarea as the source of truth; the form view parses the repeatable fields.
func readResumeForm(w http.ResponseWriter, r *http.Request, action string) (resumeForm, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Request is too large or malformed.", http.StatusBadRequest)
		return resumeForm{}, false
	}

	parsed := parseResumeForm(r.PostForm)
	if r.PostFormValue("editor_view") != "json" {
		parsed.View = "form"
		return parsed.withEditor(action), true
	}

	raw := r.PostFormValue("data")
	form := formFromData([]byte(raw))
	form.Name = parsed.Name
	form.TemplateID = parsed.TemplateID
	form.View = "json"
	form = form.withEditor(action)
	// Keep the exact text the user submitted, even when it doesn't parse.
	form.Data = raw
	return form, true
}
