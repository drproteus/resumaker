package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"earthmanlabs/resumaker/render"
)

const usage = `Usage:
	resumaker <data> <template> <style>`

func main() {
	if len(os.Args) < 4 {
		fmt.Println(usage)
		return
	}
	dataPath, templatePath, stylePath := os.Args[1], os.Args[2], os.Args[3]

	// Asset references are resolved relative to the working directory, so the
	// stylesheet path has to be relative to it as well.
	stylePathSlash := filepath.ToSlash(filepath.Clean(stylePath))
	if !fs.ValidPath(stylePathSlash) {
		log.Fatalf("style path %q must be relative to the working directory", stylePath)
	}

	data, err := os.ReadFile(dataPath)
	if err != nil {
		log.Fatal(err)
	}
	tpl, err := os.ReadFile(templatePath)
	if err != nil {
		log.Fatal(err)
	}
	style, err := os.ReadFile(stylePath)
	if err != nil {
		log.Fatal(err)
	}

	htmlDoc, err := render.HTML(render.Options{
		Data:      data,
		Template:  string(tpl),
		Style:     style,
		StylePath: stylePathSlash,
		Assets:    os.DirFS("."),
	})
	if err != nil {
		log.Fatal(err)
	}

	const buildDir = "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		log.Fatal(err)
	}
	htmlPath := filepath.Join(buildDir, "resume.html")
	if err := os.WriteFile(htmlPath, htmlDoc, 0644); err != nil {
		log.Fatal(err)
	}

	pdf, err := render.PDF(context.Background(), htmlDoc)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "resume.pdf"), pdf, 0644); err != nil {
		log.Fatal(err)
	}
}
