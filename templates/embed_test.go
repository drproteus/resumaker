package templates

import (
	"io/fs"
	"strings"
	"testing"

	"earthmanlabs/resumaker/assets"
)

func TestCatalog(t *testing.T) {
	all := All()
	if len(all) < 4 {
		t.Fatalf("catalog has %d templates, want several styles", len(all))
	}

	seen := map[string]bool{}
	for _, tpl := range all {
		if tpl.ID == "" || tpl.Name == "" || tpl.Description == "" {
			t.Errorf("incomplete template metadata: %+v", tpl)
		}
		if seen[tpl.ID] {
			t.Errorf("duplicate template ID %q", tpl.ID)
		}
		seen[tpl.ID] = true

		src, err := tpl.Source()
		if err != nil {
			t.Errorf("%s: read source: %v", tpl.ID, err)
		} else if !strings.Contains(src, "{{if .Toolbar}}") {
			t.Errorf("%s: source has no print toolbar", tpl.ID)
		}

		if _, err := fs.ReadFile(assets.FS(), tpl.Style); err != nil {
			t.Errorf("%s: read style %q: %v", tpl.ID, tpl.Style, err)
		}
	}

	if _, ok := Lookup("missing"); ok {
		t.Error("Lookup found an unknown template")
	}
}
