package integration

// End-to-end tests for the NPS feedback API.
//
// With the SQLite backend the database is embedded, so these tests spin up a
// temporary on-disk database and the real HTTP handler — no external service is
// required and they run in CI by default.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/handler"
)

func newTestServer(t *testing.T) (*httptest.Server, *db.Database) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Connect(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	t.Cleanup(func() { database.Close(context.Background()) })

	srv := httptest.NewServer(handler.RegisterRoutes(database))
	t.Cleanup(srv.Close)

	return srv, database
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/nps/health")
	if err != nil {
		t.Fatalf("GET /nps/health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSubmitFeedback_StoresRow(t *testing.T) {
	srv, database := newTestServer(t)

	body := `{
		"schema_version": "1.0",
		"app": "rokdsk",
		"app_version": "1.2.3",
		"platform": "macOS",
		"timestamp": "2026-01-01T00:00:00Z",
		"nps_rating": 9,
		"nps_category": "promoter",
		"comment": "integration test"
	}`

	resp, err := http.Post(srv.URL+"/nps/api/v1/feedback", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST feedback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var count int
	if err := database.DB().QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM feedback WHERE app = ? AND nps_rating = ?`,
		"rokdsk", 9).Scan(&count); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 stored row, got %d", count)
	}
}

func TestSubmitFeedback_InvalidPlatformRejected(t *testing.T) {
	srv, _ := newTestServer(t)

	body := `{
		"schema_version": "1.0",
		"app": "rokdsk",
		"app_version": "1.2.3",
		"platform": "Linux",
		"timestamp": "2026-01-01T00:00:00Z",
		"nps_rating": 9,
		"nps_category": "promoter"
	}`

	resp, err := http.Post(srv.URL+"/nps/api/v1/feedback", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST feedback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid platform, got %d", resp.StatusCode)
	}
}
