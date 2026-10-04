package model

import (
	"fmt"
	"time"
)

// Feedback represents an NPS feedback submission.
type Feedback struct {
	ID            int64     `json:"id,omitempty"`
	SchemaVersion string    `json:"schema_version"`
	App           string    `json:"app"`
	AppVersion    string    `json:"app_version"`
	Platform      string    `json:"platform"`
	Timestamp     string    `json:"timestamp"`
	NPSRating     int       `json:"nps_rating"`
	NPSCategory   string    `json:"nps_category"`
	Timezone      string    `json:"timezone,omitempty"`
	Comment       string    `json:"comment,omitempty"`
	ReceivedAt    time.Time `json:"received_at"`
}

var validPlatforms = map[string]bool{
	"macOS":   true,
	"Windows": true,
}

var validCategories = map[string]bool{
	"detractor": true,
	"passive":   true,
	"promoter":  true,
}

// Validate checks that all required fields are present and valid.
func (f *Feedback) Validate() error {
	if f.SchemaVersion != "1.0" {
		return fmt.Errorf("unsupported schema_version: %q", f.SchemaVersion)
	}
	if f.App == "" {
		return fmt.Errorf("app is required")
	}
	if f.AppVersion == "" {
		return fmt.Errorf("app_version is required")
	}
	if !validPlatforms[f.Platform] {
		return fmt.Errorf("invalid platform: %q", f.Platform)
	}
	if f.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}
	if f.NPSRating < 1 || f.NPSRating > 10 {
		return fmt.Errorf("nps_rating must be between 1 and 10")
	}
	if !validCategories[f.NPSCategory] {
		return fmt.Errorf("invalid nps_category: %q", f.NPSCategory)
	}
	if len(f.Comment) > 2000 {
		return fmt.Errorf("comment exceeds 2000 characters")
	}
	return nil
}
