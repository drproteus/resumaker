import json
import pdfgen
from jinja2 import Environment, PackageLoader, select_autoescape

env = Environment(
    loader=PackageLoader("app"),
    autoescape=select_autoescape(),
)

PDF_OPTIONS = {
    "scale": 0.8,
    "format": "Letter",
    "margin": {
        "top": "0.50in",
        "right": "0.50in",
        "bottom": "0.50in",
        "left": "0.50in",
    },
    "pageRanges": "1",
}

RESUME_TEMPLATE = "resume.html"
RESUME_JSON_PATH = "./data/resume.json"
RESUME_HTML_PATH = "./build/resume.html"
RESUME_PDF_PATH = "./build/resume.pdf"


def render():
    template = env.get_template(RESUME_TEMPLATE)
    with open(RESUME_JSON_PATH, "r") as f:
        data = json.load(f)
    html = template.render(data)
    return html


def write(html):
    with open(RESUME_HTML_PATH, "w") as f:
        f.write(html)


def write_pdf(html):
    pdfgen.sync.from_string(html, RESUME_PDF_PATH, options=PDF_OPTIONS)


if __name__ == "__main__":
    html = render()
    write(html)
    write_pdf(html)
