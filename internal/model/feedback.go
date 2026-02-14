package model

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Feedback represents an NPS feedback submission.
type Feedback struct {
	ID            bson.ObjectID `bson:"_id,omitempty"      json:"id,omitempty"`
	SchemaVersion string        `bson:"schema_version"     json:"schema_version"`
	App           string        `bson:"app"                json:"app"`
	AppVersion    string        `bson:"app_version"        json:"app_version"`
	Platform      string        `bson:"platform"           json:"platform"`
	Timestamp     string        `bson:"timestamp"          json:"timestamp"`
	NPSRating     int           `bson:"nps_rating"         json:"nps_rating"`
	NPSCategory   string        `bson:"nps_category"       json:"nps_category"`
	Timezone      string        `bson:"timezone,omitempty" json:"timezone,omitempty"`
	Comment       string        `bson:"comment,omitempty"  json:"comment,omitempty"`
	ReceivedAt    time.Time     `bson:"received_at"        json:"received_at"`
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
