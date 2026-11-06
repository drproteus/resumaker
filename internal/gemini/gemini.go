// Package gemini tailors resume JSON with Google's Gemini API.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultModel is a Gemini Flash alias, which supports URL context and
	// structured output.
	DefaultModel = "gemini-flash-latest"
	// DefaultBaseURL is the Generative Language API endpoint.
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

	maxResponseBytes = 8 << 20
)

// Options configures a Client. APIKey is required.
type Options struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// Client calls the Gemini generateContent endpoint.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func New(opts Options) *Client {
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = DefaultModel
	}
	baseURL := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		apiKey:     strings.TrimSpace(opts.APIKey),
		model:      model,
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// Result is a tailored resume. Resume is the resume JSON without the name, and
// Name is the suggested name for the new resume.
type Result struct {
	Name   string
	Resume []byte
}

// TailorResume asks Gemini to rewrite a resume for the supplied instructions.
func (c *Client) TailorResume(ctx context.Context, resumeJSON []byte, prompt string) (Result, error) {
	if c == nil || c.apiKey == "" {
		return Result{}, errors.New("gemini API key is not configured")
	}

	body := generateRequest{
		SystemInstruction: content{Parts: []part{{Text: systemPrompt}}},
		Contents: []content{{
			Role: "user",
			Parts: []part{{
				Text: fmt.Sprintf("ORIGINAL RESUME JSON:\n%s\n\nINSTRUCTIONS FROM THE USER:\n%s", resumeJSON, prompt),
			}},
		}},
		GenerationConfig: generationConfig{
			Temperature:      0.4,
			MaxOutputTokens:  8192,
			ResponseMIMEType: "application/json",
			ResponseSchema:   responseSchema,
		},
	}
	// Let Gemini retrieve job descriptions and other pages linked in the
	// prompt. Google performs the fetch, so the app never requests user URLs.
	if containsURL(prompt) {
		body.Tools = []map[string]any{{"url_context": map[string]any{}}}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, url.PathEscape(c.model))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Result{}, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, apiError(raw))
	}

	text, err := responseText(raw)
	if err != nil {
		return Result{}, err
	}
	object, err := extractJSON(text)
	if err != nil {
		return Result{}, err
	}
	return splitName(object)
}

// splitName removes resume_name from the model's object, so the stored resume
// matches the render.ResumeData schema.
func splitName(object []byte) (Result, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(object, &fields); err != nil {
		return Result{}, fmt.Errorf("gemini: response JSON: %w", err)
	}
	var name string
	if raw, ok := fields["resume_name"]; ok {
		if err := json.Unmarshal(raw, &name); err != nil {
			return Result{}, fmt.Errorf("gemini: resume_name: %w", err)
		}
		delete(fields, "resume_name")
	}
	resume, err := json.Marshal(fields)
	if err != nil {
		return Result{}, err
	}
	return Result{Name: strings.TrimSpace(name), Resume: resume}, nil
}

func containsURL(s string) bool {
	return strings.Contains(s, "http://") || strings.Contains(s, "https://")
}

type generateRequest struct {
	SystemInstruction content          `json:"systemInstruction"`
	Contents          []content        `json:"contents"`
	Tools             []map[string]any `json:"tools,omitempty"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature      float64         `json:"temperature"`
	MaxOutputTokens  int             `json:"maxOutputTokens"`
	ResponseMIMEType string          `json:"responseMimeType"`
	ResponseSchema   json.RawMessage `json:"responseSchema"`
}

type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text    string `json:"text"`
				Thought bool   `json:"thought"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func responseText(raw []byte) (string, error) {
	var resp generateResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("gemini: %s", resp.Error.Message)
	}
	if resp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("gemini: prompt blocked: %s", resp.PromptFeedback.BlockReason)
	}
	if len(resp.Candidates) == 0 {
		return "", errors.New("gemini: no candidates returned")
	}

	var b strings.Builder
	for _, p := range resp.Candidates[0].Content.Parts {
		if !p.Thought {
			b.WriteString(p.Text)
		}
	}
	if strings.TrimSpace(b.String()) == "" {
		reason := resp.Candidates[0].FinishReason
		if reason == "" {
			reason = "unknown"
		}
		return "", fmt.Errorf("gemini: empty response (finish reason %s)", reason)
	}
	return b.String(), nil
}

func apiError(raw []byte) string {
	var resp struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &resp) == nil && resp.Error != nil && resp.Error.Message != "" {
		return resp.Error.Message
	}
	if len(raw) > 500 {
		raw = raw[:500]
	}
	return strings.TrimSpace(string(raw))
}

// extractJSON accepts either plain JSON or a fenced code block and returns the
// first JSON value in the text. Structured output usually needs no cleanup, but
// models occasionally add a fence despite responseMimeType.
func extractJSON(text string) ([]byte, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		if newline := strings.IndexByte(text, '\n'); newline >= 0 {
			text = text[newline+1:]
		}
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "```"))
	}

	start := strings.IndexByte(text, '{')
	if start < 0 {
		return nil, errors.New("gemini: response did not contain a JSON object")
	}
	dec := json.NewDecoder(strings.NewReader(text[start:]))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("gemini: response JSON: %w", err)
	}
	return raw, nil
}

const systemPrompt = `You tailor resumes to a specific job, opportunity, or set of instructions.

Return the complete resume as JSON using exactly the supplied schema. Keep every fact
truthful: never invent employers, dates, degrees, skills, metrics, or achievements.
You may reorder and rewrite existing content, emphasise relevant experience, and add
keywords that are directly supported by the original resume. If the instructions
mention URLs, use the URL context tool to read them before writing.

The "comments" array holds only your own notes about the choices you made: which parts
you reordered or rephrased, which skills or achievements you highlighted, and how the
instructions or job description influenced the result. Write one short note per array
item, addressed to the user. Do not repeat the user's existing comments; the application
keeps those itself. Leave comments empty only if you changed nothing.

The "resume_name" field is the name for this new version. If the instructions name a
company, position, title or other detail, combine what is given into a short name, for
example "Senior Go Engineer - Acme". Otherwise, base it on the original resume name.

Do not include markdown, code fences, or any text outside the JSON object.`

// The schema mirrors render.ResumeData. Gemini expects its OpenAPI-style subset
// of JSON Schema for responseSchema.
var responseSchema = json.RawMessage(`{
  "type": "OBJECT",
  "properties": {
    "fullname": {"type": "STRING"},
    "contact": {
      "type": "ARRAY",
      "items": {
        "type": "OBJECT",
        "properties": {
          "type": {"type": "STRING", "enum": ["email", "telephone", "website", "other"]},
          "value": {"type": "STRING"}
        },
        "required": ["type", "value"]
      }
    },
    "experience": {
      "type": "ARRAY",
      "items": {
        "type": "OBJECT",
        "properties": {
          "title": {"type": "STRING"},
          "company": {"type": "STRING"},
          "location": {"type": "STRING"},
          "start_date": {"type": "STRING"},
          "end_date": {"type": "STRING"},
          "highlights": {"type": "ARRAY", "items": {"type": "STRING"}}
        },
        "required": ["title", "company", "location", "start_date", "end_date", "highlights"]
      }
    },
    "education": {
      "type": "ARRAY",
      "items": {
        "type": "OBJECT",
        "properties": {
          "school": {"type": "STRING"},
          "degree": {"type": "STRING"},
          "highlights": {"type": "ARRAY", "items": {"type": "STRING"}}
        },
        "required": ["school", "degree", "highlights"]
      }
    },
    "keywords": {"type": "ARRAY", "items": {"type": "STRING"}},
    "skills": {
      "type": "ARRAY",
      "items": {
        "type": "OBJECT",
        "properties": {
          "category": {"type": "STRING"},
          "items": {"type": "ARRAY", "items": {"type": "STRING"}}
        },
        "required": ["category", "items"]
      }
    },
    "comments": {"type": "ARRAY", "items": {"type": "STRING"}},
    "resume_name": {"type": "STRING"}
  },
  "required": ["fullname", "contact", "experience", "education", "keywords", "skills", "comments", "resume_name"]
}`)
