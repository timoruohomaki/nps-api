package integration

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/idefinity/nps-api/internal/crypto"
	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/model"
)

// legacySchema is the pre-migration (user_version 0) table as deployed before
// schema 1.1, with app_version and platform NOT NULL.
const legacySchema = `CREATE TABLE feedback (
    id INTEGER PRIMARY KEY AUTOINCREMENT, schema_version TEXT NOT NULL,
    app TEXT NOT NULL, app_version TEXT NOT NULL, platform TEXT NOT NULL,
    timestamp TEXT NOT NULL, nps_rating INTEGER NOT NULL, nps_category TEXT NOT NULL,
    timezone TEXT, comment TEXT, received_at TEXT NOT NULL)`

func TestMigration_LegacyDBBecomesNullableAndKeepsRows(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.db")

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	for _, s := range []string{
		legacySchema,
		`INSERT INTO feedback (schema_version, app, app_version, platform, timestamp,
            nps_rating, nps_category, timezone, comment, received_at) VALUES
            ('1.0','rokdsk','1.0.0','macOS','2026-01-01T00:00:00Z',9,'promoter','','old one','2026-01-01T00:00:00Z'),
            ('1.0','idefinity','0.1.0','Windows','2026-02-01T00:00:00Z',5,'detractor','','old two','2026-02-01T00:00:00Z'),
            ('1.0','rokdsk','1.0.0','macOS','2026-03-01T00:00:00Z',8,'passive','','deleted','2026-03-01T00:00:00Z')`,
		`DELETE FROM feedback WHERE id = 3`,
	} {
		if _, err := raw.Exec(s); err != nil {
			t.Fatalf("seed legacy db: %v", err)
		}
	}
	raw.Close()

	enc, _ := crypto.New("") // passthrough: legacy rows above are plaintext
	database, err := db.Connect(ctx, path, enc)
	if err != nil {
		t.Fatalf("connect (migrate) legacy db: %v", err)
	}
	t.Cleanup(func() { database.Close(ctx) })

	var version int
	database.DB().QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != 1 {
		t.Errorf("expected user_version 1 after migration, got %d", version)
	}

	rows, err := database.DB().Query(`SELECT name, "notnull" FROM pragma_table_info('feedback')
        WHERE name IN ('app_version', 'platform')`)
	if err != nil {
		t.Fatalf("table_info: %v", err)
	}
	for rows.Next() {
		var name string
		var notNull int
		rows.Scan(&name, &notNull)
		if notNull != 0 {
			t.Errorf("column %s still NOT NULL after migration", name)
		}
	}
	rows.Close()

	items, err := database.ListFeedback(ctx, 0)
	if err != nil {
		t.Fatalf("list migrated rows: %v", err)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 ||
		items[0].Comment != "old one" || items[1].Platform != "Windows" || items[1].AppVersion != "0.1.0" {
		t.Fatalf("migrated rows not preserved: %+v", items)
	}

	// AUTOINCREMENT counter survives the rebuild: id 3 was used, so next is 4.
	fb := &model.Feedback{SchemaVersion: "1.1", App: "rokdsk", Timestamp: "2026-11-19T00:00:00Z",
		NPSRating: 10, NPSCategory: "promoter", ReceivedAt: time.Now()}
	if err := database.InsertFeedback(ctx, fb); err != nil {
		t.Fatalf("insert 1.1 row after migration: %v", err)
	}
	if fb.ID != 4 {
		t.Errorf("expected new id 4 (no reuse of deleted id 3), got %d", fb.ID)
	}

	// Reconnecting is a no-op: migrations are not re-applied.
	database.Close(ctx)
	again, err := db.Connect(ctx, path, enc)
	if err != nil {
		t.Fatalf("reconnect migrated db: %v", err)
	}
	defer again.Close(ctx)
	if items, _ := again.ListFeedback(ctx, 0); len(items) != 3 {
		t.Errorf("expected 3 rows after reconnect, got %d", len(items))
	}
}
