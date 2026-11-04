package render

import (
	"context"
	"strings"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const defaultTimeout = 60 * time.Second

// Renderer prints HTML documents to PDF with headless Chrome. It owns a single
// browser process; each call opens its own tab, so a Renderer can be reused
// across jobs.
type Renderer struct {
	allocCtx context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
}

type RendererOptions struct {
	// Timeout bounds a single print. Defaults to 60 seconds.
	Timeout time.Duration
	// ExecPath overrides the Chrome/Edge binary to launch.
	ExecPath string
}

func NewRenderer(opts RendererOptions) (*Renderer, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	allocOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocOpts = append(allocOpts,
		chromedp.NoSandbox,
		chromedp.Flag("disable-gpu", true),
	)
	if opts.ExecPath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(opts.ExecPath))
	}

	// The browser process outlives individual jobs, so it can't inherit a
	// per-job deadline. Each PDF call applies its own timeout.
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	return &Renderer{allocCtx: allocCtx, cancel: cancel, timeout: timeout}, nil
}

func (r *Renderer) Close() {
	r.cancel()
}

// PDF prints a self-contained HTML document. All network access is blocked:
// only data:, blob: and about: URLs are allowed, so a document can't reach
// internal services or the local filesystem.
func (r *Renderer) PDF(ctx context.Context, doc []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	tabCtx, tabCancel := chromedp.NewContext(ctx)
	defer tabCancel()

	chromedp.ListenTarget(tabCtx, func(ev any) {
		paused, ok := ev.(*fetch.EventRequestPaused)
		if !ok {
			return
		}
		go func() {
			var action chromedp.Action
			if allowedURL(paused.Request.URL) {
				action = fetch.ContinueRequest(paused.RequestID)
			} else {
				action = fetch.FailRequest(paused.RequestID, network.ErrorReasonBlockedByClient)
			}
			if err := chromedp.Run(tabCtx, action); err != nil {
				// The tab closed or timed out; nothing left to do.
				return
			}
		}()
	})

	var pdfBuffer []byte
	err := chromedp.Run(tabCtx,
		chromedp.Navigate("about:blank"),
		fetch.Enable().WithPatterns([]*fetch.RequestPattern{{
			URLPattern:   "*",
			RequestStage: fetch.RequestStageRequest,
		}}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(tree.Frame.ID, string(doc)).Do(ctx)
		}),
		// Make sure webfonts are ready before measuring the page for print.
		chromedp.Evaluate(`document.fonts.ready.then(() => true)`, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuffer, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}
	return pdfBuffer, nil
}

// allowedURL reports whether a request may proceed. Everything else, including
// file:, http: and https:, is blocked.
func allowedURL(raw string) bool {
	u := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(u, "data:") ||
		strings.HasPrefix(u, "blob:") ||
		strings.HasPrefix(u, "about:")
}

// PDF prints one document using a temporary browser. Prefer a Renderer when
// printing repeatedly.
func PDF(ctx context.Context, doc []byte) ([]byte, error) {
	r, err := NewRenderer(RendererOptions{})
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return r.PDF(ctx, doc)
}
