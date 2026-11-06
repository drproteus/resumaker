# Resumaker
Resumaker is a simple alternative to WYSIWYG editing that uses templates to render a
resume from JSON into a self-contained HTML page, ready to print or save as PDF.

# Screenshots

![The resumes list with preview, duplicate and delete actions](docs/screenshots/app-list.png)

The editor has a structured form and a raw JSON view that stay in sync:

![The resume editor with its Form and JSON toggle](docs/screenshots/app-editor.png)

## Resume styles

| Classic | Modern | Sidebar |
| --- | --- | --- |
| [![Classic resume style](docs/screenshots/template-classic.png)](docs/screenshots/template-classic.png) | [![Modern resume style](docs/screenshots/template-modern.png)](docs/screenshots/template-modern.png) | [![Sidebar resume style](docs/screenshots/template-sidebar.png)](docs/screenshots/template-sidebar.png) |

| Comic | Dingbats |
| --- | --- |
| [![Comic resume style](docs/screenshots/template-comic.png)](docs/screenshots/template-comic.png) | [![Dingbats resume style](docs/screenshots/template-dingbats.png)](docs/screenshots/template-dingbats.png) |

Screenshots use synthetic resume data.

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

Resumes are edited with a structured form, with a toggle for editing the raw JSON
instead. Drag the handle beside a row to reorder contact details, experience,
education or skill groups, and use Duplicate on the resume list to start from an
existing version.

## Tailoring with Gemini
If `GEMINI_API_KEY` is set, the resume editor offers **Tailor with Gemini**. Paste a
job description, one or more job posting links, skills to highlight, or any other
instructions, and Gemini creates a new resume instead of changing the original. The
generation runs in the background, so the browser stays responsive while it works. The
new resume is named after the role and company when the instructions give them. Gemini
explains its changes in the resume's `"comments"` array, each starting with `_Gemini:`.
The editor shows those notes with a Gemini icon instead of the prefix.
Users' own comments (to-dos, reminders) live in the same array and are never printed.
Job links are passed to Gemini through its URL context tool, so the server does not
fetch user-supplied URLs.

`GEMINI_MODEL` defaults to `gemini-flash-latest`; set `GEMINI_API_BASE_URL` only when
using a compatible proxy or a different API host. Resume JSON is sent to Google when
you use this feature; leave `GEMINI_API_KEY` unset to keep it entirely local.

Five styles ship with the app: Classic, Modern, Sidebar, Comic (a Comic Sans homage)
and Dingbats (Wingdings-adjacent). The Help page explains the editor, printing and
privacy. To render a style from the command line, pass its template and stylesheet:

```bash
$ ./resumaker render/testdata/resume.json templates/sidebar.gohtml assets/css/sidebar.css
```

## Running locally
Start PostgreSQL (uses `compose.yaml`) and set the required environment variables:

```bash
$ docker compose up -d db
$ export RESUMAKER_DATABASE_URL='postgres://resumaker:resumaker@localhost:5432/resumaker?sslmode=disable'
$ export RESUMAKER_MASTER_KEY="$(openssl rand -base64 32)"   # keep this safe; losing it makes stored resumes unreadable
$ export GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=...      # and/or GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET
$ export GEMINI_API_KEY=...                                  # optional: enables Tailor with Gemini
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

## Deployment
An Ansible playbook in `deploy/` provisions the production host with PostgreSQL,
nginx, a Let's Encrypt certificate and a systemd service. It cross-compiles the Go
binary on the controller and deploys it to `resumaker.skinet.org`, using the
existing SSH config for the connection. See [`deploy/README.md`](deploy/README.md)
for secrets setup and the deployment command.

## Tests
Database tests run only when `TEST_DATABASE_URL` is set:

```bash
$ TEST_DATABASE_URL='postgres://resumaker:resumaker@localhost:5432/resumaker?sslmode=disable' go test ./...
```
