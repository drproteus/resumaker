import json
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
    with open("./build/resume.html", "w") as f:
        f.write(html)


if __name__ == "__main__":
    render()
