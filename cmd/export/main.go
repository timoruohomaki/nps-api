// Command export reads all stored feedback, decrypts the PII fields, and writes
// it to stdout as a JSON array. It uses the same configuration as the server
// (DB_PATH, FEEDBACK_ENC_KEY), so run it with the same environment:
//
//	docker compose exec nps-api ./export > feedback.json
//
// There is no read HTTP endpoint by design; this is the supported way to get the
// plaintext feedback back out.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/idefinity/nps-api/internal/config"
	"github.com/idefinity/nps-api/internal/crypto"
	"github.com/idefinity/nps-api/internal/db"
)

func main() {
	cfg := config.Load()

	enc, err := crypto.New(cfg.EncKey)
	if err != nil {
		fail("invalid FEEDBACK_ENC_KEY: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.DBPath, enc)
	if err != nil {
		fail("open database: %v", err)
	}
	defer database.Close(context.Background())

	items, err := database.ListFeedback(ctx)
	if err != nil {
		fail("read feedback: %v", err)
	}

	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "  ")
	if err := w.Encode(items); err != nil {
		fail("encode: %v", err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "export: "+format+"\n", args...)
	os.Exit(1)
}
