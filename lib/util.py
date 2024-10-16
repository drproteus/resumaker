import json
from urllib.parse import urlparse


def parse_website(url):
    try:
        parsed = urlparse(url)
        if parsed.netloc:
            return {
                "name": f"{parsed.netloc}{parsed.path}",
                "href": url,
            }
    except Exception:
        pass
    return {
        "name": url,
        "href": url
    }


def load_resume_data(data_path):
    with open(data_path, "r") as f:
        data = json.load(f)
    if "website" in data:
        data["website"] = parse_website(data["website"])
    return data
