package codexswitch

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupExpiredArchivedSessionsDeletesMatchingIndexes(t *testing.T) {
	service := newTestService(t)
	codexHome := t.TempDir()
	archiveDir := filepath.Join(codexHome, archivedSessionsDirName)

	expiredFile := filepath.Join(archiveDir, "expired.jsonl")
	freshFile := filepath.Join(archiveDir, "fresh.jsonl")
	missingExpiredFile := filepath.Join(archiveDir, "missing-expired.jsonl")
	missingFreshFile := filepath.Join(archiveDir, "missing-fresh.jsonl")
	outsideMissingFile := filepath.Join(codexHome, "missing-outside.jsonl")

	writeTestFile(t, expiredFile, "expired")
	writeTestFile(t, freshFile, "fresh")
	setModTime(t, expiredFile, service.now().Add(-31*24*time.Hour))
	setModTime(t, freshFile, service.now().Add(-29*24*time.Hour))

	dbPath := filepath.Join(codexHome, "state_5.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test Codex state database: %v", err)
	}
	if _, err := db.Exec(`
CREATE TABLE threads (
	id TEXT PRIMARY KEY,
	rollout_path TEXT NOT NULL,
	archived INTEGER NOT NULL,
	archived_at INTEGER
);`); err != nil {
		db.Close()
		t.Fatalf("create test threads table: %v", err)
	}

	insertThread := func(id string, rolloutPath string, archivedAt time.Time) {
		t.Helper()
		if _, err := db.Exec(
			`INSERT INTO threads (id, rollout_path, archived, archived_at) VALUES (?, ?, 1, ?)`,
			id,
			rolloutPath,
			archivedAt.Unix(),
		); err != nil {
			t.Fatalf("insert test thread %s: %v", id, err)
		}
	}
	insertThread("expired", expiredFile, service.now().Add(-31*24*time.Hour))
	insertThread("fresh", freshFile, service.now().Add(-29*24*time.Hour))
	insertThread("missing-expired", missingExpiredFile, service.now().Add(-31*24*time.Hour))
	insertThread("missing-fresh", missingFreshFile, service.now().Add(-29*24*time.Hour))
	insertThread("outside-missing", outsideMissingFile, service.now().Add(-31*24*time.Hour))
	if err := db.Close(); err != nil {
		t.Fatalf("close test Codex state database: %v", err)
	}

	service.cleanupExpiredArchivedSessions(AppSettings{
		CodexHomePath:                codexHome,
		ArchivedSessionRetentionDays: 30,
	})

	assertPathMissing(t, expiredFile)
	assertPathExists(t, freshFile)

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("reopen test Codex state database: %v", err)
	}
	defer db.Close()

	assertThreadMissing(t, db, "expired")
	assertThreadMissing(t, db, "missing-expired")
	assertThreadExists(t, db, "fresh")
	assertThreadExists(t, db, "missing-fresh")
	assertThreadExists(t, db, "outside-missing")
}

func assertThreadMissing(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if threadExists(t, db, id) {
		t.Fatalf("expected thread %s to be deleted", id)
	}
}

func assertThreadExists(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if !threadExists(t, db, id) {
		t.Fatalf("expected thread %s to remain", id)
	}
}

func threadExists(t *testing.T, db *sql.DB, id string) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM threads WHERE id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("query test thread %s: %v", id, err)
	}
	return count > 0
}
