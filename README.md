# Resumaker
Resumaker is a simple alternative to WYSIWYG editing that uses templates to render a resume in HTML and PDF formats from JSON.

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

# Web app
A web app for storing several resumes per account, editing them in the browser,
and rendering them to HTML and PDF. Resume JSON and rendered output are encrypted
at rest. Sign-in uses GitHub or Google OAuth.

Rendering runs in a separate worker process, so the web app stays responsive and
headless Chrome has no access to user traffic. The worker blocks all network
requests while printing: only `data:`, `blob:` and `about:` URLs are allowed, so
a document can't reach internal services or the local filesystem.

## Running locally
Start PostgreSQL (uses `compose.yaml`) and set the required environment variables:

```bash
$ docker compose up -d db
$ export RESUMAKER_DATABASE_URL='postgres://resumaker:resumaker@localhost:5432/resumaker?sslmode=disable'
$ export RESUMAKER_MASTER_KEY="$(openssl rand -base64 32)"   # keep this safe; losing it makes stored resumes unreadable
$ export GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=...      # and/or GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET
$ go run ./cmd/web
```

In a second terminal with the same environment, start the render worker. It needs
Chrome, Chromium or Edge installed; set `RESUMAKER_CHROME_PATH` if the browser
isn't found automatically.

```bash
$ go run ./cmd/worker
```

Register the OAuth callback URLs with your provider:
- GitHub: `http://localhost:8080/auth/github/callback`
- Google: `http://localhost:8080/auth/google/callback`

Optional settings: `RESUMAKER_LISTEN_ADDR` (default `:8080`) and `RESUMAKER_BASE_URL`
(default `http://localhost:8080`; set it to the public `https://` URL in production,
which also enables secure cookies). Worker settings: `RESUMAKER_RENDER_CONCURRENCY`
(default `2`), `RESUMAKER_RENDER_TIMEOUT` (default `60s`) and `RESUMAKER_CHROME_PATH`.

Database migrations run automatically on startup. Migrations and job claims are
safe to run from several instances at once.

## Tests
Database tests run only when `TEST_DATABASE_URL` is set:

```bash
$ TEST_DATABASE_URL='postgres://resumaker:resumaker@localhost:5432/resumaker?sslmode=disable' go test ./...
```

The end-to-end render test also needs a Chrome/Chromium/Edge binary. Use
`go test -short ./...` to skip browser tests.
