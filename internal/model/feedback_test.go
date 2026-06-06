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

func TestSetAllowedPlatforms(t *testing.T) {
	t.Cleanup(func() {
		SetAllowedPlatforms([]string{"macOS", "Windows"})
	})

	SetAllowedPlatforms([]string{"macOS", "Windows", "iOS", "Android"})

	for _, p := range []string{"macOS", "Windows", "iOS", "Android"} {
		fb := validFeedback()
		fb.Platform = p
		if err := fb.Validate(); err != nil {
			t.Errorf("platform %q should be allowed, got %v", p, err)
		}
	}

	fb := validFeedback()
	fb.Platform = "Linux"
	if err := fb.Validate(); err == nil {
		t.Error("expected Linux to still be rejected")
	}
}

func TestSetAllowedPlatforms_EmptyKeepsDefaults(t *testing.T) {
	SetAllowedPlatforms(nil)
	SetAllowedPlatforms([]string{"   ", ""})

	fb := validFeedback()
	fb.Platform = "macOS"
	if err := fb.Validate(); err != nil {
		t.Errorf("default platforms should still apply, got %v", err)
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
