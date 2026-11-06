package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTailorResumeSendsPromptAndSchema(t *testing.T) {
	var gotPath, gotKey string
	var got map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-goog-api-key")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {"parts": [{"text": "{\"fullname\":\"Jane\",\"ai\":true,\"comments\":\"Because.\"}"}]},
				"finishReason": "STOP"
			}]
		}`))
	}))
	defer srv.Close()

	client := New(Options{
		APIKey:     "test-key",
		Model:      "test-model",
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	out, err := client.TailorResume(context.Background(), []byte(`{"fullname":"Jane"}`), "Read https://example.com/job and emphasise Go.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"ai":true`) {
		t.Errorf("output = %s", out)
	}
	if gotPath != "/models/test-model:generateContent" {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != "test-key" {
		t.Errorf("API key header = %q", gotKey)
	}

	contents, _ := got["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("contents = %#v", got["contents"])
	}
	parts, _ := contents[0].(map[string]any)["parts"].([]any)
	if len(parts) != 1 {
		t.Fatalf("parts = %#v", parts)
	}
	text, _ := parts[0].(map[string]any)["text"].(string)
	for _, want := range []string{`"fullname":"Jane"`, "https://example.com/job", "emphasise Go"} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt missing %q: %s", want, text)
		}
	}

	config, _ := got["generationConfig"].(map[string]any)
	if config["responseMimeType"] != "application/json" {
		t.Errorf("responseMimeType = %#v", config["responseMimeType"])
	}
	if _, ok := config["responseSchema"]; !ok {
		t.Error("request did not include a response schema")
	}

	tools, _ := got["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("URL prompts should enable URL context, tools = %#v", tools)
	}
	if _, ok := tools[0].(map[string]any)["url_context"]; !ok {
		t.Errorf("tools = %#v", tools)
	}
}

func TestTailorResumeOmitsURLContextWithoutLinks(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{}"}]}}]}`))
	}))
	defer srv.Close()

	client := New(Options{APIKey: "k", BaseURL: srv.URL, HTTPClient: srv.Client()})
	if _, err := client.TailorResume(context.Background(), []byte(`{"fullname":"Jane"}`), "Emphasise Go."); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["tools"]; ok {
		t.Errorf("tools should be omitted without links: %#v", got["tools"])
	}
}

func TestTailorResumeReportsAPIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"API key invalid"}}`))
	}))
	defer srv.Close()

	client := New(Options{APIKey: "k", BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := client.TailorResume(context.Background(), []byte(`{}`), "x")
	if err == nil || !strings.Contains(err.Error(), "API key invalid") {
		t.Fatalf("err = %v", err)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := map[string]string{
		"plain":          `{"fullname":"Jane"}`,
		"fenced":         "```json\n{\"fullname\":\"Jane\"}\n```",
		"trailing text":  "Here is the resume:\n{\"fullname\":\"Jane\"}\nDone.",
		"braces in text": `{"fullname":"Jane","comments":"Use Go {and} SQL"}`,
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			out, err := extractJSON(in)
			if err != nil {
				t.Fatal(err)
			}
			var v map[string]any
			if err := json.Unmarshal(out, &v); err != nil {
				t.Fatalf("output is not valid JSON: %q", out)
			}
			if v["fullname"] != "Jane" {
				t.Errorf("fullname = %#v", v["fullname"])
			}
		})
	}
}
