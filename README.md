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
![Screenshot 2024-10-16 at 11-07-38 ](https://github.com/user-attachments/assets/301504c1-6ef4-406e-86a2-33cdb5ad047b)

```json
{
  "fullname": "Cool Guy",
  "email": "coolguy@website.biz",
  "telephone": "973-808-1337",
  "website": "github.com/coolguy",
  "experience": [
    {
      "title": "Assistant Warlock",
      "company": "Wizards Inc.",
      "location": "Inner Eye Subsystems",
      "start_date": "Mar. 1892",
      "end_date": "Tomorrow",
      "highlights": [
        "Fought back eldritch terror.",
        "Repaired routine paper jams."
      ]
    }
  ],
  "education": [
    {
      "school": "Gizmonics Institute",
      "degree": "MSc. N-Dimensional Mathematics",
      "highlights": [
        "Brought forth the end of days.",
        "Averted the end of days."
      ]
    }
  ],
  "skills": [
    "Black magic",
    "Blood magic",
    "Orb pondering"
  ]
}
```
