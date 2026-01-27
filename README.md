# Resumaker
Resumaker is a simple alternative to WYSIWYG editing that uses templates to render a resume in HTML and PDF formats from JSON.


# Quickstart
```bash
$ go build resumaker.go
$ ./resumaker data/resume.json templates/resume.gohtml assets/css/style.css
```

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
