package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type ResumeDataExp struct {
	Title      string   `json:"title"`
	Company    string   `json:"company"`
	Location   string   `json:"location"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	Highlights []string `json:"highlights"`
}

type ResumeDataEdu struct {
	School     string   `json:"school"`
	Degree     string   `json:"degree"`
	Highlights []string `json:"highlights"`
}

type ResumeDataContact struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ResumeData struct {
	Fullname   string              `json:"fullname"`
	Contact    []ResumeDataContact `json:"contact"`
	Experience []ResumeDataExp     `json:"experience"`
	Education  []ResumeDataEdu     `json:"education"`
	Keywords   []string            `json:"keywords"`
	Skills     []string            `json:"skills"`
	Style      template.CSS
}

const usage = `Usage:
	resumaker <data> <template> <style>`

// cssURLPattern matches url(...) references in the stylesheet so local assets
// can be embedded into the generated document. The PDF is rendered from a
// standalone HTML file, so every asset has to travel with it.
var cssURLPattern = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)

func main() {
	if len(os.Args) < 4 {
		fmt.Println(usage)
		return
	}

	const buildDir = "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		log.Fatal(err)
	}

	var buffer bytes.Buffer
	if err := generateResumeHTML(os.Args[1], os.Args[2], os.Args[3], &buffer); err != nil {
		log.Fatal(err)
	}

	htmlPath := filepath.Join(buildDir, "resume.html")
	if err := os.WriteFile(htmlPath, buffer.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
	if err := generateResumePDF(htmlPath, filepath.Join(buildDir, "resume.pdf")); err != nil {
		log.Fatal(err)
	}
}

func generateResumeHTML(dataPath string, templatePath string, stylePath string, buffer io.Writer) error {
	jsonData, err := os.ReadFile(dataPath)
	if err != nil {
		return err
	}
	var resumeData ResumeData
	if err := json.Unmarshal(jsonData, &resumeData); err != nil {
		return err
	}

	styleCSS, err := os.ReadFile(stylePath)
	if err != nil {
		return err
	}
	resumeData.Style = template.CSS(inlineAssets(styleCSS, stylePath))

	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	return tpl.Execute(buffer, resumeData)
}

// inlineAssets replaces relative url(...) references in a stylesheet with
// data: URIs. This keeps the HTML self-contained and lets the PDF renderer
// pick up fonts without relying on a document base URL.
func inlineAssets(css []byte, cssPath string) []byte {
	baseDir := filepath.Dir(cssPath)
	return cssURLPattern.ReplaceAllFunc(css, func(match []byte) []byte {
		sub := cssURLPattern.FindSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		ref := strings.TrimSpace(string(sub[1]))
		if ref == "" ||
			strings.HasPrefix(ref, "data:") ||
			strings.HasPrefix(ref, "#") ||
			strings.HasPrefix(ref, "//") ||
			strings.Contains(ref, "://") {
			return match
		}

		assetPath := filepath.Join(baseDir, filepath.FromSlash(ref))
		data, err := os.ReadFile(assetPath)
		if err != nil {
			return match
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		return []byte(`url("data:` + assetMIMEType(assetPath) + `;base64,` + encoded + `")`)
	})
}

func assetMIMEType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func generateResumePDF(htmlPath string, outPath string) error {
	absPath, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	slashPath := filepath.ToSlash(absPath)
	if !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	htmlURL := (&url.URL{Scheme: "file", Path: slashPath}).String()

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var pdfBuffer []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate(htmlURL),
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
		return err
	}

	return os.WriteFile(outPath, pdfBuffer, 0644)
}
