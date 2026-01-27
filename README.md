# Resumaker
Resumaker is a simple alternative to WYSIWYG editing that uses templates to render a resume in HTML and PDF formats from JSON.

## Latest Builds
* [html](https://drproteus.github.io/resumaker/resume.html)
* [pdf](https://drproteus.github.io/resumaker/resume.pdf)

# Quickstart
```bash
$ go build resumaker.go
$ ./resumaker data/resume.json templates/resume.gohtml assets/css/style.css
```
