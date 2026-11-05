package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/render"
	resumetpl "earthmanlabs/resumaker/templates"
)

const (
	maxNameLength = 100
	copySuffix    = " (copy)"
)

// InputError reports user-correctable problems with submitted resume data.
type InputError struct {
	Msg string
}

func (e *InputError) Error() string {
	return e.Msg
}

type Template struct {
	ID          string
	Name        string
	Description string
}

// Templates lists the built-in resume styles, in display order.
func Templates() []Template {
	all := resumetpl.All()
	out := make([]Template, len(all))
	for i, t := range all {
		out[i] = Template{ID: t.ID, Name: t.Name, Description: t.Description}
	}
	return out
}

// ResumeInput is the user-submitted content for creating or updating a resume.
type ResumeInput struct {
	Name       string
	TemplateID string
	Data       []byte
}

// ResumeDetail is a resume with its latest JSON content decrypted.
type ResumeDetail struct {
	store.Resume
	Data []byte
}

func (s *Service) ListResumes(ctx context.Context, user store.User) ([]store.Resume, error) {
	return s.store.ListResumes(ctx, user.ID)
}

func (s *Service) CreateResume(ctx context.Context, user store.User, in ResumeInput) (uuid.UUID, error) {
	name, data, err := validate(in)
	if err != nil {
		return uuid.Nil, err
	}
	dek, err := s.dataKey(user)
	if err != nil {
		return uuid.Nil, err
	}
	rev, err := newRevision(dek, data)
	if err != nil {
		return uuid.Nil, err
	}
	resume := store.Resume{ID: uuid.New(), Name: name, TemplateID: in.TemplateID}
	if err := s.store.CreateResume(ctx, user.ID, resume, rev); err != nil {
		return uuid.Nil, err
	}
	return resume.ID, nil
}

// GetResume returns a resume and its decrypted latest content.
// It returns store.ErrNotFound if the resume doesn't exist or belongs to someone else.
func (s *Service) GetResume(ctx context.Context, user store.User, id uuid.UUID) (ResumeDetail, error) {
	resume, rev, err := s.store.GetResume(ctx, user.ID, id)
	if err != nil {
		return ResumeDetail{}, err
	}
	dek, err := s.dataKey(user)
	if err != nil {
		return ResumeDetail{}, err
	}
	data, err := envelope.Decrypt(dek, rev.Ciphertext, rev.ID[:])
	if err != nil {
		return ResumeDetail{}, fmt.Errorf("decrypt resume %s: %w", id, err)
	}
	return ResumeDetail{Resume: resume, Data: data}, nil
}

// DuplicateResume copies a resume's latest content into a new resume. The copy
// starts with no revision history, so editing it doesn't touch the original.
func (s *Service) DuplicateResume(ctx context.Context, user store.User, id uuid.UUID) (uuid.UUID, error) {
	detail, err := s.GetResume(ctx, user, id)
	if err != nil {
		return uuid.Nil, err
	}
	return s.CreateResume(ctx, user, ResumeInput{
		Name:       duplicateName(detail.Name),
		TemplateID: detail.TemplateID,
		Data:       detail.Data,
	})
}

// duplicateName appends a marker while keeping the result within the name limit.
func duplicateName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Resume"
	}
	limit := maxNameLength - utf8.RuneCountInString(copySuffix)
	if runes := []rune(name); len(runes) > limit {
		name = string(runes[:limit])
	}
	return name + copySuffix
}

func (s *Service) UpdateResume(ctx context.Context, user store.User, id uuid.UUID, in ResumeInput) error {
	name, data, err := validate(in)
	if err != nil {
		return err
	}
	dek, err := s.dataKey(user)
	if err != nil {
		return err
	}
	rev, err := newRevision(dek, data)
	if err != nil {
		return err
	}
	return s.store.UpdateResume(ctx, user.ID, store.Resume{ID: id, Name: name, TemplateID: in.TemplateID}, rev)
}

func (s *Service) DeleteResume(ctx context.Context, user store.User, id uuid.UUID) error {
	return s.store.DeleteResume(ctx, user.ID, id)
}

// newRevision encrypts resume content. The revision ID is used as associated
// data, so the ciphertext only decrypts under the row it was written to.
func newRevision(dek, data []byte) (store.Revision, error) {
	id := uuid.New()
	ciphertext, err := envelope.Encrypt(dek, data, id[:])
	if err != nil {
		return store.Revision{}, err
	}
	return store.Revision{ID: id, Ciphertext: ciphertext}, nil
}

// validate checks user input and returns the trimmed name and normalized JSON.
func validate(in ResumeInput) (string, []byte, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength {
		return "", nil, &InputError{Msg: fmt.Sprintf("Name must be between 1 and %d characters.", maxNameLength)}
	}
	if !validTemplate(in.TemplateID) {
		return "", nil, &InputError{Msg: "Choose a template from the list."}
	}
	data, err := normalizeResume(in.Data)
	if err != nil {
		return "", nil, &InputError{Msg: err.Error()}
	}
	return name, data, nil
}

// normalizeResume checks the JSON against the resume schema and re-indents it.
// Unknown fields are rejected so typos surface instead of being silently dropped.
func normalizeResume(raw []byte) ([]byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("Resume data is empty.")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var rd render.ResumeData
	if err := dec.Decode(&rd); err != nil {
		return nil, fmt.Errorf("Resume data: %v", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("Resume data: unexpected content after the document.")
	}
	if strings.TrimSpace(rd.Fullname) == "" {
		return nil, errors.New("Full name is required.")
	}

	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func validTemplate(id string) bool {
	_, ok := resumetpl.Lookup(id)
	return ok
}
