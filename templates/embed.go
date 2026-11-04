// Package templates bundles the resume templates into the binary.
package templates

import (
	"embed"
	"io/fs"
)

//go:embed *.gohtml
var files embed.FS

// FS returns the embedded templates.
func FS() fs.FS {
	return files
}

// source maps a template ID to its embedded file.
var source = map[string]string{
	"default": "resume.gohtml",
}

// Source returns the template source for an ID.
func Source(id string) (string, bool) {
	name, ok := source[id]
	if !ok {
		return "", false
	}
	b, err := files.ReadFile(name)
	if err != nil {
		return "", false
	}
	return string(b), true
}
