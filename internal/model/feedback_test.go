package model

import "testing"

func validFeedback() Feedback {
	return Feedback{
		SchemaVersion: "1.0",
		App:           "idefinity",
		AppVersion:    "0.1.0",
		Platform:      "macOS",
		Timestamp:     "2025-06-15T14:23:00+03:00",
		NPSRating:     9,
		NPSCategory:   "promoter",
	}
}

func TestValidate_ValidFeedback(t *testing.T) {
	fb := validFeedback()
	if err := fb.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_InvalidSchemaVersion(t *testing.T) {
	fb := validFeedback()
	fb.SchemaVersion = "2.0"
	if err := fb.Validate(); err == nil {
		t.Error("expected error for unsupported schema version")
	}
}

func TestValidate_InvalidPlatform(t *testing.T) {
	fb := validFeedback()
	fb.Platform = "Linux"
	if err := fb.Validate(); err == nil {
		t.Error("expected error for invalid platform")
	}
}

func TestValidate_RatingOutOfRange(t *testing.T) {
	tests := []struct {
		name   string
		rating int
	}{
		{"too low", 0},
		{"too high", 11},
		{"negative", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := validFeedback()
			fb.NPSRating = tt.rating
			if err := fb.Validate(); err == nil {
				t.Errorf("expected error for rating %d", tt.rating)
			}
		})
	}
}

func TestValidate_InvalidCategory(t *testing.T) {
	fb := validFeedback()
	fb.NPSCategory = "unknown"
	if err := fb.Validate(); err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestValidate_CommentTooLong(t *testing.T) {
	fb := validFeedback()
	fb.Comment = string(make([]byte, 2001))
	if err := fb.Validate(); err == nil {
		t.Error("expected error for comment exceeding 2000 chars")
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Feedback)
	}{
		{"missing app", func(f *Feedback) { f.App = "" }},
		{"missing app_version", func(f *Feedback) { f.AppVersion = "" }},
		{"missing timestamp", func(f *Feedback) { f.Timestamp = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := validFeedback()
			tt.modify(&fb)
			if err := fb.Validate(); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}
