package hub

import (
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRetentionHours_default(t *testing.T) {
	os.Unsetenv("FLOW_RETENTION_HOURS")
	if h := retentionHours(); h != 24 {
		t.Fatalf("expected 24, got %d", h)
	}
}

func TestRetentionHours_custom(t *testing.T) {
	os.Setenv("FLOW_RETENTION_HOURS", "72")
	defer os.Unsetenv("FLOW_RETENTION_HOURS")
	if h := retentionHours(); h != 72 {
		t.Fatalf("expected 72, got %d", h)
	}
}

func TestRetentionHours_invalid(t *testing.T) {
	os.Setenv("FLOW_RETENTION_HOURS", "abc")
	defer os.Unsetenv("FLOW_RETENTION_HOURS")
	if h := retentionHours(); h != 24 {
		t.Fatalf("expected 24 (fallback), got %d", h)
	}
}

func TestRetentionHours_zero(t *testing.T) {
	os.Setenv("FLOW_RETENTION_HOURS", "0")
	defer os.Unsetenv("FLOW_RETENTION_HOURS")
	if h := retentionHours(); h != 24 {
		t.Fatalf("expected 24 (fallback), got %d", h)
	}
}

func TestQueriesCRUD(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	hub := &Hub{App: app, ws: NewWSManager(nil)}

	records, err := hub.FindRecordsByFilter("queries", "1=1", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 queries initially, got %d", len(records))
	}

	col, err := hub.FindCollectionByNameOrId("queries")
	if err != nil {
		t.Fatal(err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", "test-query")
	rec.Set("filter", `{"host":"example"}`)
	if err := app.Save(rec); err != nil {
		t.Fatal(err)
	}

	id := rec.GetString("id")
	if id == "" {
		t.Fatal("expected non-empty id after create")
	}

	records, err = hub.FindRecordsByFilter("queries", "1=1", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 query, got %d", len(records))
	}
	if records[0].GetString("name") != "test-query" {
		t.Fatalf("expected name 'test-query', got '%s'", records[0].GetString("name"))
	}

	if err := app.Delete(rec); err != nil {
		t.Fatal(err)
	}
	records, err = hub.FindRecordsByFilter("queries", "1=1", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 queries after delete, got %d", len(records))
	}
}

func TestIngestAndQueryFlow(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	hub := &Hub{App: app, ws: NewWSManager(nil)}

	col, err := hub.FindCollectionByNameOrId("flows")
	if err != nil {
		t.Fatal(err)
	}

	rec := core.NewRecord(col)
	rec.Set("node", "test-agent")
	rec.Set("captured_at", "2026-07-11T00:00:00Z")
	rec.Set("method", "GET")
	rec.Set("host", "example.com")
	rec.Set("path", "/test")
	rec.Set("status_code", 200)
	rec.Set("req_size", 10)
	rec.Set("resp_size", 20)
	rec.Set("duration_ms", 5)
	if err := app.Save(rec); err != nil {
		t.Fatal(err)
	}

	records, err := hub.FindRecordsByFilter("flows", "host ~ {:host}", "-captured_at", 10, 0, map[string]any{"host": "example"})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(records))
	}
	if records[0].GetString("method") != "GET" {
		t.Fatalf("expected method GET, got %s", records[0].GetString("method"))
	}
}

func TestListNodes(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	hub := &Hub{App: app, ws: NewWSManager(nil)}

	col, err := hub.FindCollectionByNameOrId("nodes")
	if err != nil {
		t.Fatal(err)
	}

	rec := core.NewRecord(col)
	rec.Set("name", "test-node")
	rec.Set("token", "tok123")
	rec.Set("status", "up")
	rec.Set("version", "1.0")
	if err := app.Save(rec); err != nil {
		t.Fatal(err)
	}

	records, err := hub.FindRecordsByFilter("nodes", "1=1", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 node, got %d", len(records))
	}
}

func TestRetentionRuns(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	hub := &Hub{App: app, ws: NewWSManager(nil)}

	col, err := hub.FindCollectionByNameOrId("flows")
	if err != nil {
		t.Fatal(err)
	}

	oldRec := core.NewRecord(col)
	oldRec.Set("node", "old-agent")
	oldRec.Set("captured_at", "2025-01-01T00:00:00Z")
	oldRec.Set("method", "GET")
	oldRec.Set("host", "old.example.com")
	oldRec.Set("path", "/")
	oldRec.Set("status_code", 200)
	if err := app.Save(oldRec); err != nil {
		t.Fatal(err)
	}

	newRec := core.NewRecord(col)
	newRec.Set("node", "new-agent")
	newRec.Set("captured_at", "2026-07-11T00:00:00Z")
	newRec.Set("method", "POST")
	newRec.Set("host", "new.example.com")
	newRec.Set("path", "/new")
	newRec.Set("status_code", 201)
	if err := app.Save(newRec); err != nil {
		t.Fatal(err)
	}

	hub.runRetention(24)

	records, err := hub.FindRecordsByFilter("flows", "1=1", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 flow after retention, got %d", len(records))
	}
	host := records[0].GetString("host")
	if host != "new.example.com" {
		t.Fatalf("expected remaining flow host 'new.example.com', got '%s'", host)
	}
}
