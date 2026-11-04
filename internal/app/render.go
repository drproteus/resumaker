package app

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/assets"
	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/render"
	resumetpl "earthmanlabs/resumaker/templates"
)

const renderHistoryLimit = 10

// RequestRender queues a render of a resume's latest saved content.
func (s *Service) RequestRender(ctx context.Context, user store.User, resumeID uuid.UUID) (uuid.UUID, error) {
	_, rev, err := s.store.GetResume(ctx, user.ID, resumeID)
	if err != nil {
		return uuid.Nil, err
	}
	job, err := s.store.EnqueueRender(ctx, resumeID, rev.ID)
	if err != nil {
		return uuid.Nil, err
	}
	return job.ID, nil
}

func (s *Service) RenderJobs(ctx context.Context, user store.User, resumeID uuid.UUID) ([]store.RenderJob, error) {
	return s.store.ListRenderJobs(ctx, user.ID, resumeID, renderHistoryLimit)
}

// RenderArtifact decrypts a finished render. kind is "html" or "pdf".
func (s *Service) RenderArtifact(ctx context.Context, user store.User, jobID uuid.UUID, kind string) ([]byte, string, error) {
	job, html, pdf, err := s.store.RenderArtifact(ctx, user.ID, jobID)
	if err != nil {
		return nil, "", err
	}
	dek, err := s.dataKey(user)
	if err != nil {
		return nil, "", err
	}

	var ciphertext []byte
	var contentType string
	switch kind {
	case "html":
		ciphertext, contentType = html, "text/html; charset=utf-8"
	case "pdf":
		ciphertext, contentType = pdf, "application/pdf"
	default:
		return nil, "", fmt.Errorf("unknown artifact %q", kind)
	}

	plaintext, err := envelope.Decrypt(dek, ciphertext, job.ID[:])
	if err != nil {
		return nil, "", fmt.Errorf("decrypt %s for job %s: %w", kind, jobID, err)
	}
	return plaintext, contentType, nil
}

func (s *Service) ClaimRenderJob(ctx context.Context) (store.ClaimedRenderJob, error) {
	return s.store.ClaimRenderJob(ctx)
}

func (s *Service) RequeueStaleRenderJobs(ctx context.Context, olderThan time.Duration) (int64, error) {
	return s.store.RequeueStaleRenderJobs(ctx, olderThan)
}

func (s *Service) FailRenderJob(ctx context.Context, jobID uuid.UUID, message string) error {
	return s.store.FailRenderJob(ctx, jobID, message)
}

// ProcessRenderJob renders a claimed job and stores the encrypted result.
// The caller is expected to mark the job failed if this returns an error.
func (s *Service) ProcessRenderJob(ctx context.Context, job store.ClaimedRenderJob, renderer *render.Renderer) error {
	src, err := s.store.RenderSource(ctx, job.ResumeID, job.RevisionID)
	if err != nil {
		return err
	}

	tpl, ok := resumetpl.Source(src.TemplateID)
	if !ok {
		return fmt.Errorf("unknown template %q", src.TemplateID)
	}
	style, err := fs.ReadFile(assets.FS(), "css/style.css")
	if err != nil {
		return err
	}

	dek, err := s.keys.UnwrapDataKey(src.User.WrappedKey, src.User.ID[:])
	if err != nil {
		return err
	}
	data, err := envelope.Decrypt(dek, src.Ciphertext, job.RevisionID[:])
	if err != nil {
		return err
	}

	htmlDoc, err := render.HTML(render.Options{
		Data:      data,
		Template:  tpl,
		Style:     style,
		StylePath: "css/style.css",
		Assets:    assets.FS(),
	})
	if err != nil {
		return err
	}
	pdf, err := renderer.PDF(ctx, htmlDoc)
	if err != nil {
		return err
	}

	htmlEnc, err := envelope.Encrypt(dek, htmlDoc, job.ID[:])
	if err != nil {
		return err
	}
	pdfEnc, err := envelope.Encrypt(dek, pdf, job.ID[:])
	if err != nil {
		return err
	}
	return s.store.CompleteRenderJob(ctx, job.ID, htmlEnc, pdfEnc)
}
