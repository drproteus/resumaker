package render

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"regexp"
	"strings"
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

type ResumeDataSkillGroup struct {
	Category string   `json:"category"`
	Items    []string `json:"items"`
}

type ResumeData struct {
	Fullname   string                 `json:"fullname"`
	Contact    []ResumeDataContact    `json:"contact"`
	Experience []ResumeDataExp        `json:"experience"`
	Education  []ResumeDataEdu        `json:"education"`
	Keywords   []string               `json:"keywords"`
	Skills     []ResumeDataSkillGroup `json:"skills"`
	Style      template.CSS           `json:"-"`
	// Toolbar adds a print button and instructions, used by the web preview.
	Toolbar bool `json:"-"`
}

// Options describes a single render. Data, Template and Style are the only
// inputs; nothing is read from disk unless Assets is set.
type Options struct {
	// Data is the resume JSON document.
	Data []byte
	// Template is the html/template source for the resume page.
	Template string
	// Style is the stylesheet, inlined into the page.
	Style []byte
	// StylePath is the location of Style within Assets, using forward slashes.
	// Relative url(...) references in Style are resolved against its directory.
	StylePath string
	// Assets, when set, is where local url(...) references in Style are read
	// from and inlined as data: URIs. When nil, url(...) references are left
	// untouched.
	Assets fs.FS
	// Toolbar adds a print button and instructions to the document.
	Toolbar bool
}

// cssURLPattern matches url(...) references in the stylesheet so local assets
// can be embedded into the generated document. The PDF is rendered from a
// standalone HTML document, so every asset has to travel with it.
var cssURLPattern = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)

// HTML renders a self-contained resume document.
func HTML(opts Options) ([]byte, error) {
	var resumeData ResumeData
	if err := json.Unmarshal(opts.Data, &resumeData); err != nil {
		return nil, err
	}

	style := opts.Style
	if opts.Assets != nil {
		style = inlineAssets(style, opts.Assets, opts.StylePath)
	}
	resumeData.Style = template.CSS(style)
	resumeData.Toolbar = opts.Toolbar

	tpl, err := template.New("resume").Parse(opts.Template)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, resumeData); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// inlineAssets replaces relative url(...) references in a stylesheet with
// data: URIs. This keeps the HTML self-contained and lets the PDF renderer
// pick up fonts without relying on a document base URL.
//
// References that cannot be resolved inside fsys, including any that escape
// it, are left untouched.
func inlineAssets(css []byte, fsys fs.FS, cssPath string) []byte {
	baseDir := path.Dir(cssPath)
	return cssURLPattern.ReplaceAllFunc(css, func(match []byte) []byte {
		sub := cssURLPattern.FindSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		ref := strings.TrimSpace(string(sub[1]))
		if ref == "" ||
			strings.HasPrefix(ref, "/") ||
			strings.HasPrefix(ref, "data:") ||
			strings.HasPrefix(ref, "#") ||
			strings.HasPrefix(ref, "//") ||
			strings.Contains(ref, "://") {
			return match
		}

		assetPath := path.Join(baseDir, ref)
		if !fs.ValidPath(assetPath) {
			return match
		}
		data, err := fs.ReadFile(fsys, assetPath)
		if err != nil {
			return match
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		return []byte(fmt.Sprintf(`url("data:%s;base64,%s")`, assetMIMEType(assetPath), encoded))
	})
}

func assetMIMEType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
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
