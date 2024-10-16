import os
import json
import click
from jinja2 import Environment, PackageLoader, select_autoescape

env = Environment(
    loader=PackageLoader("app"),
    autoescape=select_autoescape(),
)

WINDOW_TEMPLATE = "wizard/window.html"
PAGES_DIR = "./pages"
DATA_DIR = "./data/wizard"
PAGES_INDEX = os.path.join(PAGES_DIR, "index.html")
PAGES_JSON = os.path.join(DATA_DIR, "pages.json")


def build_pages():
    template = env.get_template("wizard/window.html")
    os.makedirs(PAGES_DIR, exist_ok=True)
    with open(PAGES_JSON, "r") as f:
        data = json.load(f)
    with open(PAGES_INDEX, "w") as f:
        for i, page_json in enumerate(data):
            if i == 0:
                filename = "index.html"
                page_json["prev_page"] = filename
                if len(data) > 1:
                    page_json["next_page"] = f"step_{i + 1}.html"
                else:
                    page_json["next_page"] = filename
            else:
                filename = f"step_{i}.html"
                if i - 1 == 0:
                    page_json["prev_page"] = "index.html"
                else:
                    page_json["prev_page"] = f"step_{i - 1}.html"
                if i < len(data) - 1:
                    page_json["next_page"] = f"step_{i + 1}.html"
                else:
                    page_json["next_page"] = filename

            page_json["pages"] = len(data)
            page_json["page_num"] = i
            with open(os.path.join(PAGES_DIR, filename), "w")  as f:
                f.write(template.render(page_json))


@click.command("wizard")
def main():
    build_pages()


if __name__ == "__main__":
    main()
