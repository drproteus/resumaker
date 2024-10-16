import os
import json
import click
import re
from jinja2 import Environment, PackageLoader, select_autoescape

env = Environment(
    loader=PackageLoader("app"),
    autoescape=select_autoescape(),
    extensions=["jinja_markdown.MarkdownExtension"]
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
        pages_json = json.load(f)
    pages = []
    for filename in os.listdir(DATA_DIR):
        if filename.endswith(".md"):
            page_name = filename[:-3]
            page_output = f"{page_name}.html"
            page_data = pages_json.get(page_name, {})
            page_data["page_name"] = page_name
            with open(os.path.join(DATA_DIR, filename)) as f:
                page_data["content"] = f.read()
            page_data["output"] = page_output
            pages.append(page_data)

    def get_index(page_name):
        if page_name == "index":
            # index.md is required to start
            return 0
        else:
            # Assuming step_N.md for remaining templates
            return int(re.search(r"step_(\d+)", page_name).group(1))

    pages = sorted(pages, key=lambda p: get_index(p["page_name"]))

    with open(PAGES_INDEX, "w") as f:
        for i, page_data in enumerate(pages):
            filename = page_data["output"]
            print(page_data)
            page_data["page_num"] = i
            page_data["pages"] = len(pages)
            try:
                page_data["prev_page"] = pages[i - 1]["output"]
            except (IndexError, KeyError):
                page_data["prev_page"] = filename
            try:
                page_data["next_page"] = pages[i + 1]["output"]
            except (IndexError, KeyError):
                page_data["next_page"] = filename
            with open(os.path.join(PAGES_DIR, filename), "w") as f:
                f.write(template.render(page_data))


@click.command("wizard")
def main():
    build_pages()


if __name__ == "__main__":
    main()
