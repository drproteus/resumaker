# Resumaker
Resumaker is a simple alternative to WYSIWYG editing that uses Jinja2 templates to render a resume in HTML and PDF formats from JSON.


# Quickstart
```bash
pip install -r requirements.txt
# TODO: copy example JSON and update with personal info.
make
```
The generated `resume.html` can be found in `build/`.


# Example
![image](https://github.com/user-attachments/assets/a86a5d9a-2662-436c-9dce-c001ee3474af)

```json
{
  "fullname": "Tang Ping",
  "telephone": "(862) 200-5555",
  "email": "coolguy@website.biz",
  "github": "tangping",
  "keywords": "keyword",
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
      ]
    }
  ],
  "educations": [
    {
      "school": "Gizmonics Institute",
      "degree": "MSc. Digital Voodoo",
      "description": [
        "Digital Voodoo studies with a minor in Underwater Basketweaving",
        "Graduated with Abnormal Distinction"
      ]
    }
  ],
  "skills": [
    "Dark magic",
    "Light magic",
    "Orb juggling"
  ]
}
```
