package integration

// End-to-end tests for the NPS feedback API.
//
// With the SQLite backend the database is embedded, so these tests spin up a
// temporary on-disk database and the real HTTP handler — no external service is
// required and they run in CI by default.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idefinity/nps-api/internal/crypto"
	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/handler"
)

func testKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func newTestServer(t *testing.T) (*httptest.Server, *db.Database) {
	t.Helper()
	// No keys: write-auth disabled, read endpoint disabled (not exercised here).
	return newTestServerWithKeys(t, nil, nil)
}

func newTestServerWithKeys(t *testing.T, apiKeys, readKeys []string) (*httptest.Server, *db.Database) {
	t.Helper()

	enc, err := crypto.New(testKey(t))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Connect(context.Background(), dbPath, enc)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	t.Cleanup(func() { database.Close(context.Background()) })

	srv := httptest.NewServer(handler.RegisterRoutes(database, apiKeys, readKeys))
	t.Cleanup(srv.Close)

	return srv, database
}

const validBody = `{
	"schema_version": "1.0",
	"app": "rokdsk",
	"app_version": "1.2.3",
	"platform": "macOS",
	"timestamp": "2026-01-01T00:00:00Z",
	"nps_rating": 9,
	"nps_category": "promoter",
	"comment": "auth test"
}`

func postFeedback(t *testing.T, url, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/nps/api/v1/feedback", strings.NewReader(validBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestAPIKey_RequiredWhenConfigured(t *testing.T) {
	srv, _ := newTestServerWithKeys(t, []string{"secret-key-1", "secret-key-2"}, nil)

	t.Run("valid key accepted", func(t *testing.T) {
		resp := postFeedback(t, srv.URL, "secret-key-2")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201 with valid key, got %d", resp.StatusCode)
		}
	})

	t.Run("missing key rejected", func(t *testing.T) {
		resp := postFeedback(t, srv.URL, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 without key, got %d", resp.StatusCode)
		}
	})

	t.Run("wrong key rejected", func(t *testing.T) {
		resp := postFeedback(t, srv.URL, "nope")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 with wrong key, got %d", resp.StatusCode)
		}
	})

	t.Run("health stays open", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/nps/health")
		if err != nil {
			t.Fatalf("GET health: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("health should not require a key, got %d", resp.StatusCode)
		}
	})
}

func TestAPIKey_OpenWhenUnset(t *testing.T) {
	srv, _ := newTestServerWithKeys(t, nil, nil)
	resp := postFeedback(t, srv.URL, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 with auth disabled, got %d", resp.StatusCode)
	}
}

func getFeedback(t *testing.T, url, query, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url+"/nps/api/v1/feedback"+query, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestReadEndpoint_DisabledWithoutReadKey(t *testing.T) {
	// Write enabled, read keys unset => GET is fail-closed (503).
	srv, _ := newTestServerWithKeys(t, nil, nil)
	resp := getFeedback(t, srv.URL, "", "anything")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when READ_API_KEYS unset, got %d", resp.StatusCode)
	}
}

func TestReadEndpoint_AuthAndQuery(t *testing.T) {
	readKey := "consumer-key"
	srv, _ := newTestServerWithKeys(t, nil, []string{readKey})

	// Seed: two 2026 rows (via POST) — received_at is "now", so filtering by the
	// current year must include them and a past year must exclude them.
	for i := 0; i < 2; i++ {
		resp := postFeedback(t, srv.URL, "")
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("seed POST %d: got %d", i, resp.StatusCode)
		}
	}

	t.Run("missing key rejected", func(t *testing.T) {
		resp := getFeedback(t, srv.URL, "", "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 without key, got %d", resp.StatusCode)
		}
	})

	t.Run("valid key returns decrypted rows", func(t *testing.T) {
		resp := getFeedback(t, srv.URL, "", readKey)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var items []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(items))
		}
		if items[0]["comment"] != "auth test" {
			t.Errorf("comment not decrypted in response: %v", items[0]["comment"])
		}
	})

	t.Run("year filter excludes other years", func(t *testing.T) {
		// 2000 is valid but before any feedback existed (received_at is "now").
		resp := getFeedback(t, srv.URL, "?year=2000", readKey)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var items []map[string]any
		json.NewDecoder(resp.Body).Decode(&items)
		if len(items) != 0 {
			t.Errorf("expected 0 rows for year=2000, got %d", len(items))
		}
	})

	t.Run("invalid year rejected", func(t *testing.T) {
		resp := getFeedback(t, srv.URL, "?year=abc", readKey)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for bad year, got %d", resp.StatusCode)
		}
	})
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

	// The comment must be encrypted AT REST: reading the raw column must not
	// reveal the plaintext.
	var rawComment string
	if err := database.DB().QueryRowContext(context.Background(),
		`SELECT comment FROM feedback WHERE app = ?`, "rokdsk").Scan(&rawComment); err != nil {
		t.Fatalf("query raw comment: %v", err)
	}
	if strings.Contains(rawComment, "integration test") {
		t.Errorf("comment stored in plaintext: %q", rawComment)
	}
	if !strings.HasPrefix(rawComment, "enc:v1:") {
		t.Errorf("comment not encrypted, raw value: %q", rawComment)
	}

	// ListFeedback must decrypt it back to the original.
	items, err := database.ListFeedback(context.Background(), 0)
	if err != nil {
		t.Fatalf("list feedback: %v", err)
	}
	if len(items) != 1 || items[0].Comment != "integration test" {
		t.Errorf("decrypted comment mismatch: %+v", items)
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
