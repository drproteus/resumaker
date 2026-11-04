// Package assets bundles the stylesheets and fonts into the binary.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed css fonts images
var files embed.FS

// FS returns the embedded assets. Paths are relative to this directory.
func FS() fs.FS {
	return files
}
