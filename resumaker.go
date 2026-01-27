package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
)

type ResumeDataExp struct {
	Title      string   `json:"title"`
	Company    string   `json:"company"`
	Location   string   `json:"location"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	Highlights []string `json:"highlights"`
}

type ResumeDataEdu struct {
	School     string   `json:"school"`
	Degree     string   `json:"degree"`
	Highlights []string `json:"highlights"`
}

type ResumeData struct {
	Fullname   string          `json:"fullname"`
	Email      string          `json:"email"`
	Website    string          `json:"website"`
	Experience []ResumeDataExp `json:"experience"`
	Education  []ResumeDataEdu `json:"education"`
	Keywords   []string        `json:"keywords"`
	Telephone  string          `json:"telephone"`
	Style      template.HTML
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Insufficient arguments!")
		return
	}
	resumeDataPath := os.Args[1]
	resumeTemplatePath := os.Args[2]
	resumeStylePath := os.Args[3]

	jsonData, _ := os.ReadFile(resumeDataPath)
	var resumeData ResumeData
	json.Unmarshal(jsonData, &resumeData)
	styleHTML, _ := os.ReadFile(resumeStylePath)
	resumeData.Style = template.HTML(styleHTML)

	var tpl *template.Template
	tpl = template.Must(template.ParseFiles(resumeTemplatePath))
	tpl.Execute(os.Stdout, resumeData)
}
