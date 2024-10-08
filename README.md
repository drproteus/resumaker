# Resumaker
I prefer writing my resume in HTML vs. a WYSIWYG editor like Word. Unfortunately, hand-writing all the tags for each new entry or experience update is tiresome. Resumaker is a simple alternative that uses Jinja2 templates to output an HTML resume based on a JSON representation of the resume information, such as name, experiences, education, etc.


# Quickstart
```bash
pip install -r requirements.txt
# TODO: copy example JSON and update with personal info.
make
```
The generated `resume.html` can be found in `build/`.
