package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/render"
)

const (
	maxPromptLength = 20000
	aiSuffix        = " (AI)"
)

// ResumeGenerator produces tailored resume JSON from an existing resume and
// natural-language instructions. The Gemini client implements it.
type ResumeGenerator interface {
	TailorResume(ctx context.Context, resumeJSON []byte, prompt string) ([]byte, error)
}

// AIGenerationAvailable reports whether a generator is configured, so the UI
// can hide the feature rather than offering a form that always fails.
func (s *Service) AIGenerationAvailable() bool {
	return s != nil && s.generator != nil
}

// TailorResume copies a resume and replaces its content with a version produced
// by the configured AI generator. The original resume is never changed.
func (s *Service) TailorResume(ctx context.Context, user store.User, id uuid.UUID, prompt string) (uuid.UUID, error) {
	if s.generator == nil {
		return uuid.Nil, &InputError{Msg: "Gemini is not configured on this server."}
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return uuid.Nil, &InputError{Msg: "Add a job description, link or instructions for Gemini."}
	}
	if utf8.RuneCountInString(prompt) > maxPromptLength {
		return uuid.Nil, &InputError{Msg: fmt.Sprintf("Instructions must be at most %d characters.", maxPromptLength)}
	}

	detail, err := s.GetResume(ctx, user, id)
	if err != nil {
		return uuid.Nil, err
	}
	generated, err := s.generator.TailorResume(ctx, detail.Data, prompt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("tailor resume %s: %w", id, err)
	}
	data, err := normalizeAIResume(generated)
	if err != nil {
		return uuid.Nil, fmt.Errorf("tailor resume %s: %w", id, err)
	}

	return s.CreateResume(ctx, user, ResumeInput{
		Name:       aiResumeName(detail.Name),
		TemplateID: detail.TemplateID,
		Data:       data,
	})
}

// normalizeAIResume validates the generated JSON, records that it came from the
// AI integration, and requires an explanation of the changes.
func normalizeAIResume(raw []byte) ([]byte, error) {
	rd, err := decodeResume(raw)
	if err != nil {
		return nil, fmt.Errorf("Gemini returned invalid resume data: %w", err)
	}
	if strings.TrimSpace(rd.Fullname) == "" {
		return nil, errors.New("Gemini returned invalid resume data: full name is required.")
	}
	rd.Comments = strings.TrimSpace(rd.Comments)
	if rd.Comments == "" {
		return nil, errors.New("Gemini returned invalid resume data: comments are required.")
	}
	rd.AI = true

	// Keep empty sections as arrays rather than null so the editor and JSON
	// view see the same shape as a hand-written resume.
	if rd.Contact == nil {
		rd.Contact = []render.ResumeDataContact{}
	}
	if rd.Experience == nil {
		rd.Experience = []render.ResumeDataExp{}
	}
	if rd.Education == nil {
		rd.Education = []render.ResumeDataEdu{}
	}
	if rd.Keywords == nil {
		rd.Keywords = []string{}
	}
	if rd.Skills == nil {
		rd.Skills = []render.ResumeDataSkillGroup{}
	}

	out, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}

// aiResumeName appends a marker while keeping the result within the name limit.
func aiResumeName(name string) string {
	return resumeNameWithSuffix(name, aiSuffix)
}
