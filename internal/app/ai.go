package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/gemini"
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/render"
)

const (
	maxPromptLength = 20000
	aiSuffix        = " (AI)"

	// tailorTimeout bounds one generation, including Gemini reading job links.
	tailorTimeout = 3 * time.Minute
	// finishedTailoringTTL is how long a completed job stays available to poll.
	finishedTailoringTTL = time.Hour
	tailorFailedMessage  = "Gemini could not generate a resume. Please try again in a moment."
)

// ResumeGenerator produces tailored resume JSON from an existing resume and
// natural-language instructions. The Gemini client implements it.
type ResumeGenerator interface {
	TailorResume(ctx context.Context, resumeJSON []byte, prompt string) (gemini.Result, error)
}

type TailoringStatus string

const (
	TailoringPending TailoringStatus = "pending"
	TailoringDone    TailoringStatus = "done"
	TailoringFailed  TailoringStatus = "failed"
)

// Tailoring is a snapshot of a background generation job.
type Tailoring struct {
	ID       string
	Status   TailoringStatus
	ResumeID uuid.UUID
	// Error is safe to show to the user.
	Error string
}

type tailoringJob struct {
	Tailoring
	userID     uuid.UUID
	finishedAt time.Time
}

// tailorings tracks generation jobs in memory. Jobs are not persisted, so a
// restart drops any that are still running.
type tailorings struct {
	mu   sync.Mutex
	jobs map[string]*tailoringJob
}

func newTailorings() *tailorings {
	return &tailorings{jobs: map[string]*tailoringJob{}}
}

func (t *tailorings) add(userID uuid.UUID) *tailoringJob {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for id, job := range t.jobs {
		if !job.finishedAt.IsZero() && now.Sub(job.finishedAt) > finishedTailoringTTL {
			delete(t.jobs, id)
		}
	}
	job := &tailoringJob{
		Tailoring: Tailoring{ID: uuid.NewString(), Status: TailoringPending},
		userID:    userID,
	}
	t.jobs[job.ID] = job
	return job
}

func (t *tailorings) finish(id string, resumeID uuid.UUID, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	job, ok := t.jobs[id]
	if !ok {
		return
	}
	job.finishedAt = time.Now()
	if err != nil {
		job.Status = TailoringFailed
		job.Error = tailorFailedMessage
		return
	}
	job.Status = TailoringDone
	job.ResumeID = resumeID
}

// Tailoring returns the status of a generation job started by the user.
// Jobs owned by someone else are reported as missing.
func (s *Service) Tailoring(user store.User, jobID string) (Tailoring, error) {
	s.tailorings.mu.Lock()
	defer s.tailorings.mu.Unlock()

	job, ok := s.tailorings.jobs[jobID]
	if !ok || job.userID != user.ID {
		return Tailoring{}, store.ErrNotFound
	}
	return job.Tailoring, nil
}

// StartTailor validates the request, then generates the tailored resume in the
// background so the HTTP request returns straight away. Poll Tailoring with the
// returned job ID for the result. The original resume is never changed.
func (s *Service) StartTailor(ctx context.Context, user store.User, id uuid.UUID, prompt string) (string, error) {
	if s.generator == nil {
		return "", &InputError{Msg: "Gemini is not configured on this server."}
	}
	enabled, err := s.FeatureEnabled(ctx, user, FeatureGemini)
	if err != nil {
		return "", err
	}
	if !enabled {
		return "", &InputError{Msg: "Tailor with Gemini is not enabled for your account."}
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", &InputError{Msg: "Add a job description, link or instructions for Gemini."}
	}
	if utf8.RuneCountInString(prompt) > maxPromptLength {
		return "", &InputError{Msg: fmt.Sprintf("Instructions must be at most %d characters.", maxPromptLength)}
	}

	detail, err := s.GetResume(ctx, user, id)
	if err != nil {
		return "", err
	}

	job := s.tailorings.add(user.ID)
	go s.runTailor(job.ID, user, detail, prompt)
	return job.ID, nil
}

func (s *Service) runTailor(jobID string, user store.User, detail ResumeDetail, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), tailorTimeout)
	defer cancel()

	newID, err := s.tailor(ctx, user, detail, prompt)
	if err != nil {
		log.Printf("tailor resume %s: %v", detail.ID, err)
	}
	s.tailorings.finish(jobID, newID, err)
}

// tailor saves the generated version as a new resume.
func (s *Service) tailor(ctx context.Context, user store.User, detail ResumeDetail, prompt string) (uuid.UUID, error) {
	original, err := decodeResume(detail.Data)
	if err != nil {
		return uuid.Nil, err
	}
	generated, err := s.generator.TailorResume(ctx, detail.Data, prompt)
	if err != nil {
		return uuid.Nil, err
	}
	data, err := normalizeAIResume(generated.Resume, original.Comments, generated.Todos)
	if err != nil {
		return uuid.Nil, err
	}
	return s.CreateResume(ctx, user, ResumeInput{
		Name:       generatedResumeName(generated.Name, detail.Name),
		TemplateID: detail.TemplateID,
		Data:       data,
	})
}

// normalizeAIResume validates the generated JSON and merges comments. The user's
// existing comments are kept first, then Gemini's notes and to-dos. Each is
// marked with its prefix.
func normalizeAIResume(raw []byte, userComments, todos []string) ([]byte, error) {
	rd, err := decodeResume(raw)
	if err != nil {
		return nil, fmt.Errorf("Gemini returned invalid resume data: %w", err)
	}
	if strings.TrimSpace(rd.Fullname) == "" {
		return nil, errors.New("Gemini returned invalid resume data: full name is required.")
	}

	existing := map[string]bool{}
	comments := []string{}
	for _, c := range userComments {
		existing[normalizeComment(c)] = true
		comments = append(comments, c)
	}
	var added []string
	for _, c := range rd.Comments {
		text, _, _ := render.ParseComment(c)
		note := render.FormatComment(text, render.CommentGemini, false)
		if text == "" || existing[normalizeComment(c)] || existing[normalizeComment(note)] {
			continue
		}
		added = append(added, note)
	}
	if len(added) == 0 {
		return nil, errors.New("Gemini returned invalid resume data: no comments explaining the changes.")
	}
	for _, t := range todos {
		text, _, _ := render.ParseComment(t)
		todo := render.FormatComment(text, render.CommentTodo, false)
		if text == "" || existing[normalizeComment(todo)] {
			continue
		}
		existing[normalizeComment(todo)] = true
		added = append(added, todo)
	}
	rd.Comments = append(comments, added...)

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

// generatedResumeName uses the name Gemini suggested, or falls back to the
// original name with a marker when it suggests none.
func generatedResumeName(suggested, original string) string {
	if strings.TrimSpace(suggested) == "" {
		return resumeNameWithSuffix(original, aiSuffix)
	}
	return resumeNameWithSuffix(suggested, "")
}

func normalizeComment(c string) string {
	return strings.ToLower(strings.TrimSpace(c))
}
