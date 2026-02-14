package integration

// Integration tests for the NPS feedback API.
//
// These tests require a running MongoDB instance.
// Set MONGODB_URI environment variable before running:
//
//   MONGODB_URI="mongodb://localhost:27017" go test ./test/integration/ -v
//
// Tests are skipped automatically if MONGODB_URI is not set.

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("MONGODB_URI") == "" {
		// Skip integration tests when no MongoDB is available
		os.Exit(0)
	}
	os.Exit(m.Run())
}
