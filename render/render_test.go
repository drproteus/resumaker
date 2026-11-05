package render

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHTMLRendersResumeWithInlinedAssets(t *testing.T) {
	data, err := os.ReadFile("testdata/resume.json")
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := os.ReadFile("../templates/resume.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	style, err := os.ReadFile("../assets/css/style.css")
	if err != nil {
		t.Fatal(err)
	}

	out, err := HTML(Options{
		Data:      data,
		Template:  string(tpl),
		Style:     style,
		StylePath: "assets/css/style.css",
		Assets:    os.DirFS(".."),
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := string(out)
	for _, want := range []string{
		"<h1>Jane Example</h1>",
		"Software Engineer",
		"data:font/woff2;base64,",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if strings.Contains(doc, "../fonts/") {
		t.Error("output still references stylesheet fonts by path")
	}
}

func TestHTMLWithoutAssetsLeavesURLsUntouched(t *testing.T) {
	out, err := HTML(Options{
		Data:      []byte(`{"fullname": "Jane"}`),
		Template:  `<style>{{.Style}}</style>`,
		Style:     []byte(`@font-face{src:url("../fonts/x.woff2")}`),
		StylePath: "assets/css/style.css",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `url("../fonts/x.woff2")`) {
		t.Errorf("expected url to be left untouched, got %q", out)
	}
}

func TestInlineAssetsDoesNotReadOutsideAssets(t *testing.T) {
	fsys := fstest.MapFS{
		"secret.txt":           {Data: []byte("secret")},
		"assets/css/style.css": {Data: []byte("")},
	}
	css := `a{src:url("../../../secret.txt")} b{src:url("/secret.txt")}`

	out := string(inlineAssets([]byte(css), fsys, "assets/css/style.css"))
	if out != css {
		t.Errorf("escaping references were modified: %q", out)
	}
}

func TestInlineAssetsResolvesRelativeToStylesheet(t *testing.T) {
	fsys := fstest.MapFS{
		"assets/fonts/f.woff2": {Data: []byte("font")},
	}
	css := `@font-face{src:url("../fonts/f.woff2")}`

	out := string(inlineAssets([]byte(css), fsys, "assets/css/style.css"))
	if !strings.Contains(out, "data:font/woff2;base64,") {
		t.Errorf("expected asset to be inlined, got %q", out)
	}
}

func TestToolbarAddsPrintButton(t *testing.T) {
	out, err := HTML(Options{
		Data:     []byte(`{"fullname": "Jane"}`),
		Template: `{{if .Toolbar}}<button data-print>Print</button>{{end}}`,
		Toolbar:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "data-print") {
		t.Errorf("expected print button, got %q", out)
	}
}
