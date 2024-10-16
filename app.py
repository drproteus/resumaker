import json
import pdfgen
import click
from urllib.parse import urlparse
from jinja2 import Environment, PackageLoader, select_autoescape
from lib.util import load_resume_data

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


def write(html, out_path=RESUME_HTML_PATH):
    with open(out_path, "w") as f:
        f.write(html)


def write_pdf(html, pdf_path=RESUME_PDF_PATH, pdf_options=PDF_OPTIONS):
    pdfgen.sync.from_string(html, pdf_path, options=pdf_options)


@click.command("resumaker")
@click.option(
    "--html",
    "html_out",
    is_flag=True,
    default=False,
)
@click.option(
    "--pdf",
    "pdf_out",
    is_flag=True,
    default=False,
)
@click.option(
    "--data",
    "data_path",
    type=click.Path(exists=True),
    default=RESUME_JSON_PATH,
)
def main(html_out, pdf_out, data_path):
    template = env.get_template(RESUME_TEMPLATE)
    data = load_resume_data(data_path)
    html = template.render(data)
    if html_out:
        write(html)
    if pdf_out:
        write_pdf(html)
    if not html_out and not pdf_out:
        click.echo(html)


if __name__ == "__main__":
    main()
