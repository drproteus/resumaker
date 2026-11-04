package app

import (
	"errors"
	"strings"
	"testing"
)

const validJSON = `{"fullname": "Jane Example", "skills": [{"category": "Go", "items": ["net/http"]}]}`

func TestValidateAcceptsResume(t *testing.T) {
	name, data, err := validate(ResumeInput{Name: "  Backend  ", TemplateID: "default", Data: []byte(validJSON)})
	if err != nil {
		t.Fatal(err)
	}
	if name != "Backend" {
		t.Errorf("name = %q, want trimmed value", name)
	}
	if !strings.Contains(string(data), "\n  \"fullname\"") {
		t.Errorf("expected indented JSON, got %q", data)
	}
}

func TestValidateRejectsBadInput(t *testing.T) {
	tests := map[string]ResumeInput{
		"empty name":       {Name: "  ", TemplateID: "default", Data: []byte(validJSON)},
		"unknown template": {Name: "x", TemplateID: "nope", Data: []byte(validJSON)},
		"empty data":       {Name: "x", TemplateID: "default", Data: []byte("  ")},
		"malformed json":   {Name: "x", TemplateID: "default", Data: []byte(`{"fullname":`)},
		"unknown field":    {Name: "x", TemplateID: "default", Data: []byte(`{"fullname": "Jane", "fulname": "typo"}`)},
		"missing fullname": {Name: "x", TemplateID: "default", Data: []byte(`{"skills": []}`)},
		"trailing content": {Name: "x", TemplateID: "default", Data: []byte(validJSON + ` {}`)},
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := validate(in)
			var inputErr *InputError
			if !errors.As(err, &inputErr) {
				t.Fatalf("expected *InputError, got %v", err)
			}
		})
	}
}

func TestValidateRejectsLongName(t *testing.T) {
	in := ResumeInput{Name: strings.Repeat("a", maxNameLength+1), TemplateID: "default", Data: []byte(validJSON)}
	if _, _, err := validate(in); err == nil {
		t.Fatal("expected error for long name")
	}
}
