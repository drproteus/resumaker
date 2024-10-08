from jinja2 import Environment, PackageLoader, select_autoescape

env = Environment(
    loader=PackageLoader("app"),
    autoescape=select_autoescape()
)


def render():
    template = env.get_template("resume.html")
    data = {
        "fullname": "Jake Goritski",
        "telephone": "(862) 200-5555",
        "email": "jake@gorit.ski",
        "github": "drproteus",
        "keywords": "jake goritski, jacob goritski",
        "experiences": [
            {
                "title": "Warlock",
                "company": "Wizards Inc.",
                "location": "Jupiter",
                "start_date": "Mar. 2020",
                "end_date": "Present",
                "entries": [
                    "Did some stuff.",
                    "More stuff."
                ],
            }
        ],
        "education": [
        ],
    }
    html = template.render(data)
    with open("./build/resume.html", "w") as f:
        f.write(html)


if __name__ == "__main__":
    render()
