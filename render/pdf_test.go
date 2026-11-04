package render

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAllowedURL(t *testing.T) {
	allowed := []string{"data:font/woff2;base64,abc", "blob:https://x/y", "about:blank"}
	for _, u := range allowed {
		if !allowedURL(u) {
			t.Errorf("expected %q to be allowed", u)
		}
	}
	blocked := []string{
		"http://169.254.169.254/latest/meta-data/",
		"https://example.com/x.css",
		"file:///C:/Windows/win.ini",
		"//example.com/x.png",
		"",
	}
	for _, u := range blocked {
		if allowedURL(u) {
			t.Errorf("expected %q to be blocked", u)
		}
	}
}

// A rendered document must not be able to reach the network, which would
// otherwise let a resume template probe internal services.
func TestPDFBlocksNetworkRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping headless Chrome test in short mode")
	}

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("should not be reached"))
	}))
	defer server.Close()

	doc := []byte(`<html><head><link rel="stylesheet" href="` + server.URL + `/x.css"></head>` +
		`<body><img src="` + server.URL + `/x.png"><p>Hello</p></body></html>`)

	r, err := NewRenderer(RendererOptions{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	pdf, err := r.PDF(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Error("output is not a PDF")
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("blocked document reached the network %d time(s)", n)
	}
}

// A Renderer is reused across jobs, so printing twice must work.
func TestRendererReuse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping headless Chrome test in short mode")
	}
	r, err := NewRenderer(RendererOptions{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	for i := 0; i < 2; i++ {
		pdf, err := r.PDF(context.Background(), []byte("<html><body><p>Run</p></body></html>"))
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		if !strings.HasPrefix(string(pdf), "%PDF") {
			t.Fatalf("run %d: not a PDF", i)
		}
	}
}
