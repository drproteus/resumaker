# Resumaker
Resumaker is a simple alternative to WYSIWYG editing that uses templates to render a resume in HTML and PDF formats from JSON.

## Latest Builds
Download `resume.html` and `resume.pdf` from the
[latest release](https://github.com/drproteus/resumaker/releases/latest).

## Releases
Publishing a release runs the **Build release artifacts** workflow, which builds
the HTML and PDF and attaches both files to that release. The workflow can also
be run manually from the Actions tab by supplying an existing release tag.

# Quickstart
```bash
$ go build resumaker.go
$ ./resumaker data/resume.json templates/resume.gohtml assets/css/style.css
```

The included resume renders as a single Letter-sized page. Local assets referenced
from `style.css` are embedded as data URIs at build time, so both the HTML and PDF
are self-contained.

## Fonts
The resume is typeset in [Latin Modern Roman](https://www.gust.org.pl/projects/e-foundry/latin-modern),
bundled under the GUST Font License. See `assets/fonts/README.md` for details.
