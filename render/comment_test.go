package render

import "testing"

func TestParseComment(t *testing.T) {
	tests := []struct {
		in, text, kind string
		done           bool
	}{
		{"Call the recruiter", "Call the recruiter", CommentUser, false},
		{"_Gemini: Led with Go.", "Led with Go.", CommentGemini, false},
		{"gemini:Led with Go.", "Led with Go.", CommentGemini, false},
		{"_TODO: Verify dates", "Verify dates", CommentTodo, false},
		{"_TODO[x]: Verify dates", "Verify dates", CommentTodo, true},
		{"  _todo[X]:   Padded  ", "Padded", CommentTodo, true},
	}
	for _, tt := range tests {
		text, kind, done := ParseComment(tt.in)
		if text != tt.text || kind != tt.kind || done != tt.done {
			t.Errorf("ParseComment(%q) = %q, %q, %v; want %q, %q, %v", tt.in, text, kind, done, tt.text, tt.kind, tt.done)
		}
	}
}

func TestFormatCommentRoundTrips(t *testing.T) {
	tests := []struct {
		text, kind string
		done       bool
		want       string
	}{
		{"Mine", CommentUser, false, "Mine"},
		{"Led with Go.", CommentGemini, false, "_Gemini: Led with Go."},
		{"Verify dates", CommentTodo, false, "_TODO: Verify dates"},
		{"Verify dates", CommentTodo, true, "_TODO[x]: Verify dates"},
		{"   ", CommentTodo, false, ""},
	}
	for _, tt := range tests {
		got := FormatComment(tt.text, tt.kind, tt.done)
		if got != tt.want {
			t.Errorf("FormatComment(%q, %q, %v) = %q, want %q", tt.text, tt.kind, tt.done, got, tt.want)
		}
		if got == "" {
			continue
		}
		text, kind, done := ParseComment(got)
		if text != tt.text || kind != tt.kind || done != tt.done {
			t.Errorf("round trip of %q = %q, %q, %v", got, text, kind, done)
		}
	}
}
