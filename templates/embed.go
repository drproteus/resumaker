// Package templates bundles the resume templates into the binary.
package templates

import (
	"embed"
	"io/fs"
)

//go:embed *.gohtml
var files embed.FS

// Template is one resume style: the HTML structure and the stylesheet used to
// render it. Style is a path within the assets package.
type Template struct {
	ID          string
	Name        string
	Description string
	File        string
	Style       string
}

var catalog = []Template{
	{
		ID:          "default",
		Name:        "Classic",
		Description: "LaTeX-inspired, one page",
		File:        "resume.gohtml",
		Style:       "css/style.css",
	},
	{
		ID:          "modern",
		Name:        "Modern",
		Description: "Clean sans-serif",
		File:        "resume.gohtml",
		Style:       "css/modern.css",
	},
	{
		ID:          "sidebar",
		Name:        "Sidebar",
		Description: "Two-column, skills on the left",
		File:        "sidebar.gohtml",
		Style:       "css/sidebar.css",
	},
	{
		ID:          "comic",
		Name:        "Comic",
		Description: "Comic Sans homage",
		File:        "comic.gohtml",
		Style:       "css/comic.css",
	},
	{
		ID:          "dingbats",
		Name:        "Dingbats",
		Description: "Wingdings-adjacent",
		File:        "comic.gohtml",
		Style:       "css/dingbats.css",
	},
}

// All returns the templates in display order.
func All() []Template {
	return append([]Template(nil), catalog...)
}

// Lookup returns the template with the given ID.
func Lookup(id string) (Template, bool) {
	for _, t := range catalog {
		if t.ID == id {
			return t, true
		}
	}
	return Template{}, false
}

// Source returns the template's HTML source.
func (t Template) Source() (string, error) {
	b, err := files.ReadFile(t.File)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FS returns the embedded templates.
func FS() fs.FS {
	return files
}
