package app

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/assets"
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/render"
	resumetpl "earthmanlabs/resumaker/templates"
)

// PreviewHTML renders a resume to a self-contained HTML document that can be
// printed or saved as PDF from the browser.
func (s *Service) PreviewHTML(ctx context.Context, user store.User, resumeID uuid.UUID) ([]byte, error) {
	detail, err := s.GetResume(ctx, user, resumeID)
	if err != nil {
		return nil, err
	}

	tpl, ok := resumetpl.Lookup(detail.TemplateID)
	if !ok {
		return nil, fmt.Errorf("unknown template %q", detail.TemplateID)
	}
	source, err := tpl.Source()
	if err != nil {
		return nil, err
	}
	style, err := fs.ReadFile(assets.FS(), tpl.Style)
	if err != nil {
		return nil, err
	}

	doc, err := render.HTML(render.Options{
		Data:      detail.Data,
		Template:  source,
		Style:     style,
		StylePath: tpl.Style,
		Assets:    assets.FS(),
		Toolbar:   true,
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}
