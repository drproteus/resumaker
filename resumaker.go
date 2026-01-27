package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/grahms/html2pdf"
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
	Skills     []string        `json:"skills"`
	Style      template.HTML
}

const usage = `Usage:
	resumaker <data> <template> <style>`

func main() {
	if len(os.Args) < 4 {
		fmt.Println(usage)
		return
	}
	var buffer bytes.Buffer
	generateResumeHTML(os.Args[1], os.Args[2], os.Args[3], &buffer)
	os.WriteFile("build/resume.html", buffer.Bytes(), 0644)
	generateResumePDF(buffer.String(), "build/resume.pdf")
}

func generateResumeHTML(dataPath string, templatePath string, stylePath string, buffer io.Writer) {
	jsonData, _ := os.ReadFile(dataPath)
	var resumeData ResumeData
	json.Unmarshal(jsonData, &resumeData)
	styleHTML, _ := os.ReadFile(stylePath)
	resumeData.Style = template.HTML(styleHTML)
	tpl := template.Must(template.ParseFiles(templatePath))
	tpl.Execute(buffer, resumeData)
}

func generateResumePDF(htmlString string, outPath string) {
	pdfGen, err := html2pdf.New()
	if err != nil {
		panic(err)
	}
	pdfBuffer, err := pdfGen.GeneratePDF(&htmlString)
	if err != nil {
		panic(err)
	}
	os.WriteFile(outPath, pdfBuffer, 0644)
}
