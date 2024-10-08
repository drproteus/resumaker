import json
import pdfgen
from jinja2 import Environment, PackageLoader, select_autoescape

env = Environment(
    loader=PackageLoader("app"),
    autoescape=select_autoescape()
)


def render():
    template = env.get_template("resume.html")
    with open("./data/resume.json", "r") as f:
        data = json.load(f)
    html = template.render(data)
    return html


def write(html):
    with open("./build/resume.html", "w") as f:
        f.write(html)


def write_pdf(html):
    options = {
        "scale": 1.0,
        "format": "Letter",
        "margin": {
            "top": "0.75in",
            "right": "0.75in",
            "bottom": "0.75in",
            "left": "0.75in",
        }
    }
    pdfgen.sync.from_string(html, "./build/resume.pdf", options=options)


if __name__ == "__main__":
    html = render()
    write(html)
    write_pdf(html)
