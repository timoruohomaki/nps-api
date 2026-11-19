package integration

// Payload schema 1.1 (ROK-311): app_version and platform are consent-gated and
// may be omitted. 1.0 keeps requiring both.

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"
)

func postBody(t *testing.T, url, body string) int {
	t.Helper()
	resp, err := http.Post(url+"/nps/api/v1/feedback", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST feedback: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

const noConsentBody = `{
	"schema_version": "%s",
	"app": "rokdsk",
	"timestamp": "2026-11-19T10:00:00Z",
	"nps_rating": 7,
	"nps_category": "passive",
	"comment": "no consent"
}`

func TestSchema11_ConsentFieldsOmitted_StoredAsNull(t *testing.T) {
	srv, database := newTestServer(t)

	if got := postBody(t, srv.URL, strings.Replace(noConsentBody, "%s", "1.1", 1)); got != http.StatusCreated {
		t.Fatalf("expected 201 for 1.1 without app_version/platform, got %d", got)
	}

	var appVersion, platform sql.NullString
	if err := database.DB().QueryRowContext(context.Background(),
		`SELECT app_version, platform FROM feedback WHERE app = 'rokdsk'`).
		Scan(&appVersion, &platform); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if appVersion.Valid || platform.Valid {
		t.Errorf("omitted fields must be NULL, got app_version=%v platform=%v", appVersion, platform)
	}

	items, err := database.ListFeedback(context.Background(), 0)
	if err != nil {
		t.Fatalf("list feedback with NULL columns: %v", err)
	}
	if len(items) != 1 || items[0].AppVersion != "" || items[0].Platform != "" ||
		items[0].Comment != "no consent" {
		t.Errorf("unexpected decoded row: %+v", items)
	}
}

func TestSchema11_WithConsentFields_Accepted(t *testing.T) {
	srv, _ := newTestServer(t)
	body := strings.Replace(validBody, `"1.0"`, `"1.1"`, 1)
	if got := postBody(t, srv.URL, body); got != http.StatusCreated {
		t.Errorf("expected 201 for 1.1 with consented fields, got %d", got)
	}
}

func TestSchema11_InvalidPlatformRejected(t *testing.T) {
	srv, _ := newTestServer(t)
	body := strings.Replace(validBody, `"1.0"`, `"1.1"`, 1)
	body = strings.Replace(body, `"macOS"`, `"Plan9"`, 1)
	if got := postBody(t, srv.URL, body); got != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for 1.1 with disallowed platform, got %d", got)
	}
}

func TestSchema10_ConsentFieldsOmitted_Rejected(t *testing.T) {
	srv, _ := newTestServer(t)
	if got := postBody(t, srv.URL, strings.Replace(noConsentBody, "%s", "1.0", 1)); got != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for 1.0 without app_version/platform, got %d", got)
	}
}
