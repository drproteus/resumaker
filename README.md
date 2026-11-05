# Resumaker
Resumaker is a simple alternative to WYSIWYG editing that uses templates to render a
resume from JSON into a self-contained HTML page, ready to print or save as PDF.

# Quickstart
```bash
$ go build resumaker.go
$ ./resumaker render/testdata/resume.json templates/resume.gohtml assets/css/style.css
```

The resume renders as a single Letter-sized page at `build/resume.html`. Local assets
referenced from `style.css` are embedded as data URIs at build time, so the HTML is
self-contained. Open it in a browser and print (or save as PDF); the print stylesheet
handles the page setup. Point the command at your own JSON file to render your resume.

## Fonts
The resume is typeset in [Latin Modern Roman](https://www.gust.org.pl/projects/e-foundry/latin-modern),
bundled under the GUST Font License. See `assets/fonts/README.md` for details.

# Web app
A web app for storing several resumes per account and editing them in the browser.
Resume JSON is encrypted at rest. Sign-in uses GitHub or Google OAuth.

Previewing renders the resume to a self-contained HTML page in the browser with a
print stylesheet. Use the Print button (or Ctrl/Cmd+P) to save it as a PDF, so no
browser automation or external renderer is involved.

## Running locally
Start PostgreSQL (uses `compose.yaml`) and set the required environment variables:

```bash
$ docker compose up -d db
$ export RESUMAKER_DATABASE_URL='postgres://resumaker:resumaker@localhost:5432/resumaker?sslmode=disable'
$ export RESUMAKER_MASTER_KEY="$(openssl rand -base64 32)"   # keep this safe; losing it makes stored resumes unreadable
$ export GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=...      # and/or GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET
$ go run ./cmd/web
```

Register the OAuth callback URLs with your provider:
- GitHub: `http://localhost:8080/auth/github/callback`
- Google: `http://localhost:8080/auth/google/callback`

Optional settings: `RESUMAKER_LISTEN_ADDR` (default `:8080`) and `RESUMAKER_BASE_URL`
(default `http://localhost:8080`; set it to the public `https://` URL in production,
which also enables secure cookies).

Database migrations run automatically on startup and take an advisory lock, so
several instances can start at once.

## Tests
Database tests run only when `TEST_DATABASE_URL` is set:

```bash
$ TEST_DATABASE_URL='postgres://resumaker:resumaker@localhost:5432/resumaker?sslmode=disable' go test ./...
```
