package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	})
}

func TestAPIKey_NoKeysIsNoOp(t *testing.T) {
	h := APIKey(nil, []string{"/nps/api/"})(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/nps/api/v1/feedback", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no keys configured, got %d", w.Code)
	}
}

func TestAPIKey_SkipsNonMatchingPrefix(t *testing.T) {
	h := APIKey([]string{"secret"}, []string{"/nps/api/"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/nps/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected health to bypass auth, got %d", w.Code)
	}
}

func TestAPIKey_RejectsMissingHeader(t *testing.T) {
	h := APIKey([]string{"secret"}, []string{"/nps/api/"})(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/nps/api/v1/feedback", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without key, got %d", w.Code)
	}
}

func TestAPIKey_RejectsWrongKey(t *testing.T) {
	h := APIKey([]string{"secret"}, []string{"/nps/api/"})(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/nps/api/v1/feedback", nil)
	req.Header.Set("X-API-Key", "nope")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong key, got %d", w.Code)
	}
}

func TestAPIKey_AcceptsValidKey(t *testing.T) {
	h := APIKey([]string{"alpha", "beta"}, []string{"/nps/api/"})(okHandler())

	for _, k := range []string{"alpha", "beta"} {
		req := httptest.NewRequest(http.MethodPost, "/nps/api/v1/feedback", nil)
		req.Header.Set("X-API-Key", k)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for key %q, got %d", k, w.Code)
		}
	}
}

func TestAPIKey_BlankKeysIgnored(t *testing.T) {
	h := APIKey([]string{"  ", ""}, []string{"/nps/api/"})(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/nps/api/v1/feedback", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected blank-only keys to act as no-op, got %d", w.Code)
	}
}
