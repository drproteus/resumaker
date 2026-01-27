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
![Screenshot 2024-10-16 at 11-36-11 ](https://github.com/user-attachments/assets/4b8450b7-0691-4f11-9139-6a6166d98289)

```json
{
  "fullname": "Cool Guy",
  "email": "coolguy@website.biz",
  "telephone": "973-808-1337",
  "website": "https://github.com/coolguy",
  "experience": [
    {
      "title": "Assistant Warlock",
      "company": "Inner Eye Subsystems",
      "location": "Neo New Amsterdam",
      "start_date": "Mar. 1A92",
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


# Go (v2)

After some struggle with Python environment dependencies setting this project up after some time away I've decided to refactor with Go. So far, the HTML generation is at least supported.

```
go build resumaker.go
./resumaker data/resume.json templates/resume.gohtml templates/style.html
```
